package storage

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FindPriorResponseSources 向前溯源：在目标请求之前的响应中查找包含指定值的来源。
func (d *Storage) FindPriorResponseSources(sessionID, beforeRequestID, value string, limit int) ([]ValueOccurrence, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()
	limit = normalizeOccurrenceLimit(limit)

	beforeReq, beforeTime, err := d.resolveBeforeConstraint(sessionID, beforeRequestID)
	if err != nil {
		return nil, err
	}

	occurrences := make([]ValueOccurrence, 0, limit)
	requests, err := d.collectRequestCandidatesTxn(txn, RequestListOptions{SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	for _, req := range requests {
		if beforeReq != nil && !req.CreatedAt.Before(beforeTime) {
			continue
		}
		for _, artifact := range ExtractResponseArtifacts(req) {
			if !artifactMatchesValue(artifact.Value, value) {
				continue
			}
			occurrences = append(occurrences, ValueOccurrence{
				RequestID:  req.ID,
				SessionID:  sessionID,
				CreatedAt:  req.CreatedAt,
				Method:     req.Method,
				Host:       req.Host,
				Path:       req.Path,
				StatusCode: req.StatusCode,
				Artifact:   artifact,
				IsResponse: true,
			})
			if len(occurrences) >= limit {
				sort.Slice(occurrences, func(i, j int) bool {
					return occurrences[i].CreatedAt.After(occurrences[j].CreatedAt)
				})
				return occurrences, nil
			}
		}
	}

	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].CreatedAt.After(occurrences[j].CreatedAt)
	})
	return occurrences, nil
}

// FindLaterRequestUsages 向后追踪：在源请求之后的请求中查找复用指定值的位置。
func (d *Storage) FindLaterRequestUsages(sessionID, afterRequestID, value string, limit int) ([]ValueOccurrence, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()
	limit = normalizeOccurrenceLimit(limit)

	afterReq, afterTime, err := d.resolveAfterConstraint(sessionID, afterRequestID)
	if err != nil {
		return nil, err
	}

	occurrences := make([]ValueOccurrence, 0, limit)
	requests, err := d.collectRequestCandidatesTxn(txn, RequestListOptions{SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	for _, req := range requests {
		if afterReq != nil && !req.CreatedAt.After(afterTime) {
			continue
		}
		for _, artifact := range ExtractRequestArtifacts(req) {
			if !artifactMatchesValue(artifact.Value, value) {
				continue
			}
			occurrences = append(occurrences, ValueOccurrence{
				RequestID:  req.ID,
				SessionID:  sessionID,
				CreatedAt:  req.CreatedAt,
				Method:     req.Method,
				Host:       req.Host,
				Path:       req.Path,
				StatusCode: req.StatusCode,
				Artifact:   artifact,
				IsResponse: false,
			})
			if len(occurrences) >= limit {
				sort.Slice(occurrences, func(i, j int) bool {
					return occurrences[i].CreatedAt.Before(occurrences[j].CreatedAt)
				})
				return occurrences, nil
			}
		}
	}

	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].CreatedAt.Before(occurrences[j].CreatedAt)
	})
	return occurrences, nil
}

// TraceValueFlow 端到端值溯源：从目标请求的指定字段出发，追踪其值的来源链路。
func (d *Storage) TraceValueFlow(sessionID, requestID, fieldName string, location ParamLocation, limit int) (*ProvenanceChain, error) {
	targetReq, err := d.GetRequest(sessionID, requestID)
	if err != nil {
		return nil, err
	}

	var targetArtifact *ParamArtifact
	var targetArtifacts []ParamArtifact

	switch location {
	case ParamLocationResponseHeader, ParamLocationResponseCookie, ParamLocationResponseJSON:
		targetArtifacts = ExtractResponseArtifacts(targetReq)
	default:
		targetArtifacts = ExtractRequestArtifacts(targetReq)
	}

	for _, artifact := range targetArtifacts {
		if artifact.Location == location && strings.EqualFold(artifact.Name, fieldName) {
			copied := artifact
			targetArtifact = &copied
			break
		}
	}
	if targetArtifact == nil {
		return nil, fmt.Errorf("target artifact not found")
	}

	sources, err := d.FindPriorResponseSources(sessionID, requestID, targetArtifact.Value, limit)
	if err != nil {
		return nil, err
	}

	targetOccurrence := ValueOccurrence{
		RequestID:  targetReq.ID,
		SessionID:  sessionID,
		CreatedAt:  targetReq.CreatedAt,
		Method:     targetReq.Method,
		Host:       targetReq.Host,
		Path:       targetReq.Path,
		StatusCode: targetReq.StatusCode,
		Artifact:   *targetArtifact,
	}
	links := BuildProvenanceLinks(sources, targetOccurrence)

	chain := &ProvenanceChain{
		TargetRequestID: targetReq.ID,
		TargetArtifact:  *targetArtifact,
		Links:           links,
		Confidence:      0,
		Evidence:        make([]string, 0, len(links)),
	}
	if len(links) > 0 {
		chain.Confidence = links[0].Confidence
		for _, link := range links[:min(len(links), 3)] {
			chain.Evidence = append(chain.Evidence, fmt.Sprintf("%s.%s -> %s.%s (confidence=%.2f)", link.SourceArtifact.Location, link.SourceArtifact.Name, link.TargetArtifact.Location, link.TargetArtifact.Name, link.Confidence))
		}
	}
	return chain, nil
}

// --- Provenance query helpers ---

func (d *Storage) resolveBeforeConstraint(sessionID, beforeRequestID string) (*Request, time.Time, error) {
	if strings.TrimSpace(beforeRequestID) == "" {
		return nil, time.Time{}, nil
	}
	req, err := d.GetRequest(sessionID, beforeRequestID)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("request not found")
	}
	return req, req.CreatedAt, nil
}

func (d *Storage) resolveAfterConstraint(sessionID, afterRequestID string) (*Request, time.Time, error) {
	if strings.TrimSpace(afterRequestID) == "" {
		return nil, time.Time{}, nil
	}
	req, err := d.GetRequest(sessionID, afterRequestID)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("request not found")
	}
	return req, req.CreatedAt, nil
}

func artifactMatchesValue(artifactValue, targetValue string) bool {
	if strings.TrimSpace(targetValue) == "" {
		return false
	}
	left := strings.ToLower(strings.TrimSpace(artifactValue))
	right := strings.ToLower(strings.TrimSpace(targetValue))
	if left == right {
		return true
	}
	return strings.Contains(left, right) || strings.Contains(right, left)
}

func normalizeOccurrenceLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}
