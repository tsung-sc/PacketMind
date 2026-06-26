package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ParamLocation string

const (
	ParamLocationQuery          ParamLocation = "query"
	ParamLocationHeader         ParamLocation = "header"
	ParamLocationCookie         ParamLocation = "cookie"
	ParamLocationFormBody       ParamLocation = "form_body"
	ParamLocationJSONBody       ParamLocation = "json_body"
	ParamLocationResponseHeader ParamLocation = "response_header"
	ParamLocationResponseCookie ParamLocation = "response_cookie"
	ParamLocationResponseJSON   ParamLocation = "response_json"
)

type ParamArtifact struct {
	Location ParamLocation `json:"location"`
	Name     string        `json:"name"`
	Value    string        `json:"value"`
	Path     string        `json:"path,omitempty"`
	RawType  string        `json:"raw_type,omitempty"`
}

type ValueOccurrence struct {
	RequestID  string        `json:"request_id"`
	SessionID  string        `json:"session_id,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	Method     string        `json:"method,omitempty"`
	Host       string        `json:"host,omitempty"`
	Path       string        `json:"path,omitempty"`
	StatusCode int           `json:"status_code,omitempty"`
	Artifact   ParamArtifact `json:"artifact"`
	IsResponse bool          `json:"is_response"`
}

type ProvenanceLink struct {
	SourceRequestID    string        `json:"source_request_id"`
	SourceArtifact     ParamArtifact `json:"source_artifact"`
	TargetRequestID    string        `json:"target_request_id"`
	TargetArtifact     ParamArtifact `json:"target_artifact"`
	Confidence         float64       `json:"confidence"`
	SameSession        bool          `json:"same_session"`
	SameHost           bool          `json:"same_host"`
	TimeDeltaMS        int64         `json:"time_delta_ms"`
	TransformType      string        `json:"transform_type,omitempty"`
	SemanticSimilarity float64       `json:"semantic_similarity"`
}

type ProvenanceChain struct {
	TargetRequestID string           `json:"target_request_id"`
	TargetArtifact  ParamArtifact    `json:"target_artifact"`
	Links           []ProvenanceLink `json:"links"`
	Confidence      float64          `json:"confidence"`
	Evidence        []string         `json:"evidence"`
}

func ExtractRequestArtifacts(req *Request) []ParamArtifact {
	if req == nil {
		return nil
	}

	artifacts := make([]ParamArtifact, 0, 16)
	artifacts = append(artifacts, extractQueryArtifacts(req.QueryString)...)
	artifacts = append(artifacts, extractHeaderArtifacts(req.Headers, ParamLocationHeader)...)
	artifacts = append(artifacts, extractCookieArtifacts(req.Cookies, ParamLocationCookie)...)
	reqBody := DecodeBodyBytes(req.Body, req.Headers)
	artifacts = append(artifacts, extractBodyArtifacts(reqBody, req.ContentType, false)...)
	return artifacts
}

func ExtractResponseArtifacts(req *Request) []ParamArtifact {
	if req == nil {
		return nil
	}

	artifacts := make([]ParamArtifact, 0, 16)
	artifacts = append(artifacts, extractHeaderArtifacts(req.RespHeaders, ParamLocationResponseHeader)...)
	artifacts = append(artifacts, extractResponseCookieArtifacts(req.RespHeaders)...)
	respBody := DecodeBodyBytes(req.RespBody, req.RespHeaders)
	artifacts = append(artifacts, extractBodyArtifacts(respBody, req.RespContentType, true)...)
	return artifacts
}

func BuildProvenanceLinks(sources []ValueOccurrence, target ValueOccurrence) []ProvenanceLink {
	links := make([]ProvenanceLink, 0, len(sources))
	for _, source := range sources {
		if !source.CreatedAt.Before(target.CreatedAt) {
			continue
		}

		link := ProvenanceLink{
			SourceRequestID:    source.RequestID,
			SourceArtifact:     source.Artifact,
			TargetRequestID:    target.RequestID,
			TargetArtifact:     target.Artifact,
			SameSession:        source.SessionID != "" && source.SessionID == target.SessionID,
			SameHost:           source.Host != "" && source.Host == target.Host,
			TimeDeltaMS:        target.CreatedAt.Sub(source.CreatedAt).Milliseconds(),
			SemanticSimilarity: CalculateSemanticSimilarity(source.Artifact.Name, target.Artifact.Name),
		}

		link.TransformType = detectTransform(source.Artifact.Value, target.Artifact.Value)
		link.Confidence = CalculateProvenanceConfidence(&link)
		links = append(links, link)
	}

	sort.Slice(links, func(i, j int) bool {
		if links[i].Confidence == links[j].Confidence {
			return links[i].TimeDeltaMS < links[j].TimeDeltaMS
		}
		return links[i].Confidence > links[j].Confidence
	})
	return links
}

func CalculateProvenanceConfidence(link *ProvenanceLink) float64 {
	confidence := 0.0
	if link == nil {
		return confidence
	}

	if link.TimeDeltaMS >= 0 {
		confidence += 0.2
	}
	if link.SameSession {
		confidence += 0.25
	}
	if link.SameHost {
		confidence += 0.15
	}

	switch link.TransformType {
	case "exact":
		confidence += 0.3
	case "prefix", "suffix":
		confidence += 0.2
	case "url_encode", "url_decode":
		confidence += 0.15
	}

	confidence += 0.1 * link.SemanticSimilarity

	if confidence > 1 {
		return 1
	}
	if confidence < 0 {
		return 0
	}
	return confidence
}

func CalculateSemanticSimilarity(sourceField, targetField string) float64 {
	source := normalizeSemanticToken(sourceField)
	target := normalizeSemanticToken(targetField)
	if source == "" || target == "" {
		return 0
	}
	if source == target {
		return 1
	}

	groups := [][]string{
		{"token", "access_token", "auth_token", "authorization", "bearer"},
		{"id", "user_id", "uid", "userid"},
		{"session", "session_id", "sessid", "sessionid"},
		{"key", "api_key", "apikey"},
	}
	for _, group := range groups {
		hasSource := false
		hasTarget := false
		for _, item := range group {
			if source == item {
				hasSource = true
			}
			if target == item {
				hasTarget = true
			}
		}
		if hasSource && hasTarget {
			return 0.85
		}
	}

	if strings.Contains(source, target) || strings.Contains(target, source) {
		return 0.6
	}
	return 0
}

func extractQueryArtifacts(queryString string) []ParamArtifact {
	if strings.TrimSpace(queryString) == "" {
		return nil
	}

	values, err := url.ParseQuery(queryString)
	if err != nil {
		return nil
	}

	artifacts := make([]ParamArtifact, 0, len(values))
	for key, vals := range values {
		for _, value := range vals {
			artifacts = append(artifacts, ParamArtifact{Location: ParamLocationQuery, Name: key, Value: value})
		}
	}
	return artifacts
}

func extractHeaderArtifacts(headers Headers, location ParamLocation) []ParamArtifact {
	if len(headers) == 0 {
		return nil
	}
	artifacts := make([]ParamArtifact, 0, len(headers))
	for key, vals := range headers {
		for _, value := range vals {
			artifacts = append(artifacts, ParamArtifact{Location: location, Name: key, Value: value})
		}
	}
	return artifacts
}

func extractCookieArtifacts(cookies Cookies, location ParamLocation) []ParamArtifact {
	if len(cookies) == 0 {
		return nil
	}
	artifacts := make([]ParamArtifact, 0, len(cookies))
	for key, value := range cookies {
		artifacts = append(artifacts, ParamArtifact{Location: location, Name: key, Value: value})
	}
	return artifacts
}

func extractResponseCookieArtifacts(headers Headers) []ParamArtifact {
	values, ok := headers["Set-Cookie"]
	if !ok || len(values) == 0 {
		return nil
	}

	artifacts := make([]ParamArtifact, 0, len(values))
	for _, item := range values {
		first := strings.TrimSpace(strings.Split(item, ";")[0])
		parts := strings.SplitN(first, "=", 2)
		if len(parts) != 2 {
			continue
		}
		artifacts = append(artifacts, ParamArtifact{Location: ParamLocationResponseCookie, Name: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])})
	}
	return artifacts
}

func extractBodyArtifacts(body []byte, contentType string, isResponse bool) []ParamArtifact {
	if len(body) == 0 {
		return nil
	}

	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "application/json") || strings.Contains(ct, "+json"):
		location := ParamLocationJSONBody
		if isResponse {
			location = ParamLocationResponseJSON
		}
		return extractJSONArtifacts(body, location)
	case strings.Contains(ct, "application/x-www-form-urlencoded"):
		return extractFormArtifacts(string(body))
	default:
		return nil
	}
}

func extractFormArtifacts(body string) []ParamArtifact {
	values, err := url.ParseQuery(body)
	if err != nil {
		return nil
	}
	artifacts := make([]ParamArtifact, 0, len(values))
	for key, vals := range values {
		for _, value := range vals {
			artifacts = append(artifacts, ParamArtifact{Location: ParamLocationFormBody, Name: key, Value: value})
		}
	}
	return artifacts
}

func extractJSONArtifacts(body []byte, location ParamLocation) []ParamArtifact {
	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	artifacts := make([]ParamArtifact, 0, 8)
	walkJSONArtifacts(payload, "", location, &artifacts)
	return artifacts
}

func walkJSONArtifacts(value interface{}, path string, location ParamLocation, artifacts *[]ParamArtifact) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, nested := range current {
			nextPath := key
			if path != "" {
				nextPath = path + "." + key
			}
			walkJSONArtifacts(nested, nextPath, location, artifacts)
		}
	case []interface{}:
		for index, nested := range current {
			nextPath := fmt.Sprintf("%s[%d]", path, index)
			walkJSONArtifacts(nested, nextPath, location, artifacts)
		}
	case string:
		*artifacts = append(*artifacts, ParamArtifact{Location: location, Name: leafName(path), Value: current, Path: path, RawType: "string"})
	case float64:
		*artifacts = append(*artifacts, ParamArtifact{Location: location, Name: leafName(path), Value: strconv.FormatFloat(current, 'f', -1, 64), Path: path, RawType: "number"})
	case bool:
		*artifacts = append(*artifacts, ParamArtifact{Location: location, Name: leafName(path), Value: strconv.FormatBool(current), Path: path, RawType: "boolean"})
	}
}

func leafName(path string) string {
	if path == "" {
		return "value"
	}
	parts := strings.Split(path, ".")
	last := parts[len(parts)-1]
	if idx := strings.Index(last, "["); idx >= 0 {
		return last[:idx]
	}
	return last
}

func detectTransform(source, target string) string {
	if source == target {
		return "exact"
	}
	if strings.HasPrefix(target, source) || strings.HasSuffix(target, source) {
		if strings.HasPrefix(target, source) {
			return "suffix"
		}
		return "prefix"
	}
	if decoded, err := url.QueryUnescape(target); err == nil && decoded == source {
		return "url_encode"
	}
	if encoded := url.QueryEscape(source); encoded == target {
		return "url_decode"
	}
	return ""
}

func normalizeSemanticToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}
