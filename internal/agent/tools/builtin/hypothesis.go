package builtin

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

func newTestHypothesisHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		requestIDsRaw := tools.GetStringArg(args, "request_ids", "")
		targetField := tools.GetStringArg(args, "target_field", "")
		targetLocation := tools.GetStringArg(args, "target_location", "")
		hypothesisStr := tools.GetStringArg(args, "hypothesis", "")

		if targetField == "" {
			return nil, fmt.Errorf("target_field is required")
		}
		if targetLocation == "" {
			return nil, fmt.Errorf("target_location is required")
		}
		if hypothesisStr == "" {
			return nil, fmt.Errorf("hypothesis is required")
		}

		requestIDs := parseRequestIDs(requestIDsRaw)
		if len(requestIDs) < 1 {
			return nil, fmt.Errorf("at least 1 request_id is required")
		}

		parsedExpr, err := parseHypothesis(hypothesisStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse hypothesis: %w", err)
		}

		results := make([]map[string]any, 0, len(requestIDs))
		matchCount := 0
		mismatchCount := 0

		for _, reqID := range requestIDs {
			req, err := store.GetRequestByID(reqID)
			if err != nil {
				results = append(results, map[string]any{
					"request_id":      reqID,
					"error":           fmt.Sprintf("request not found: %v", err),
					"match":           false,
					"actual_value":    "",
					"computed_value":  "",
					"inputs_used":     map[string]string{},
					"computation_steps": []string{},
				})
				mismatchCount++
				continue
			}

			accessor := &requestDataAccessor{req: req}
			ctx := &evalContext{req: accessor}

			computed, err := parsedExpr.Eval(ctx)
			actual := extractActualValue(req, targetField, targetLocation)

			match := false
			if err == nil && computed != "" && actual != "" {
				match = strings.EqualFold(computed, actual) || strings.Contains(actual, computed) || strings.Contains(computed, actual)
			}

			inputs := accessor.inputsUsed()
			steps := []string{fmt.Sprintf("hypothesis=%s", hypothesisStr)}
			if err != nil {
				steps = append(steps, fmt.Sprintf("eval error: %v", err))
			} else {
				steps = append(steps, fmt.Sprintf("computed=%s", truncateForModel(computed, 40)))
			}

			result := map[string]any{
				"request_id":        reqID,
				"actual_value":      truncateForModel(actual, 40),
				"computed_value":    truncateForModel(computed, 40),
				"match":             match,
				"inputs_used":       inputs,
				"computation_steps": steps,
			}
			results = append(results, result)

			if match {
				matchCount++
			} else {
				mismatchCount++
			}
		}

		testCount := len(requestIDs)
		matchRate := float64(0)
		if testCount > 0 {
			matchRate = float64(matchCount) / float64(testCount)
		}

		confidence := "low"
		verdict := fmt.Sprintf("%d/%d requests match.", matchCount, testCount)
		if matchRate == 1.0 && testCount >= 3 {
			confidence = "high"
			verdict = fmt.Sprintf("ALL %d requests MATCH. Hypothesis likely confirmed.", testCount)
		} else if matchRate >= 0.6 {
			confidence = "medium"
			verdict = fmt.Sprintf("%d/%d requests match (%.0f%%). Hypothesis partially supported.", matchCount, testCount, matchRate*100)
		} else if matchRate > 0 {
			verdict = fmt.Sprintf("Only %d/%d requests match (%.0f%%). Hypothesis likely incorrect.", matchCount, testCount, matchRate*100)
		} else {
			verdict = fmt.Sprintf("No requests match. Hypothesis is incorrect.")
		}

		content := mustMarshalJSON(map[string]any{
			"ok":         true,
			"hypothesis": hypothesisStr,
			"target_field":    targetField,
			"test_count":      testCount,
			"results":         results,
			"summary": map[string]any{
				"match_count":    matchCount,
				"mismatch_count": mismatchCount,
				"match_rate":     matchRate,
				"confidence":     confidence,
				"verdict":        verdict,
			},
			"alternative_hypotheses": generateAlternativeHypotheses(hypothesisStr, matchRate),
		})

		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Hypothesis test: %s (%d/%d match)", targetField, matchCount, testCount),
			RequestIDs: requestIDs,
		}, nil
	}
}

func parseRequestIDs(raw string) []string {
	// Handle both comma-separated and JSON array formats
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		// Simple JSON array parsing
		raw = raw[1 : len(raw)-1]
	}
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			ids = append(ids, p)
		}
	}
	return ids
}

func extractActualValue(req *storage.Request, field, location string) string {
	switch strings.ToLower(location) {
	case "body":
		if field == "body_raw" {
			return string(storage.DecodeBodyBytes(req.Body, req.Headers))
		}
		bodyText := string(storage.DecodeBodyBytes(req.Body, req.Headers))
		return extractJSONPath(bodyText, field)
	case "header":
		vals := req.Headers[field]
		if len(vals) > 0 {
			return strings.Join(vals, ", ")
		}
		return ""
	case "query":
		if req.QueryString != "" {
			values, _ := url.ParseQuery(req.QueryString)
			if v, ok := values[field]; ok && len(v) > 0 {
				return v[0]
			}
		}
		return ""
	case "response_body":
		if field == "body_raw" {
			return string(storage.DecodeBodyBytes(req.RespBody, req.RespHeaders))
		}
		bodyText := string(storage.DecodeBodyBytes(req.RespBody, req.RespHeaders))
		return extractJSONPath(bodyText, field)
	case "response_header":
		vals := req.RespHeaders[field]
		if len(vals) > 0 {
			return strings.Join(vals, ", ")
		}
		return ""
	case "cookie":
		if v, ok := req.Cookies[field]; ok {
			return v
		}
		return ""
	default:
		return ""
	}
}

func generateAlternativeHypotheses(hypothesis string, matchRate float64) []string {
	alts := make([]string, 0, 3)
	if matchRate < 1.0 {
		// Suggest hash algorithm variants
		if strings.Contains(hypothesis, "MD5") {
			alts = append(alts, strings.Replace(hypothesis, "MD5", "SHA256", 1))
		}
		if strings.Contains(hypothesis, "SHA256") {
			alts = append(alts, strings.Replace(hypothesis, "SHA256", "MD5", 1))
		}
		if !strings.Contains(hypothesis, "HMAC") {
			alts = append(alts, "Try adding a secret key: HMAC_SHA256(CONCAT(...), secret=EXTRACT(response_body, $.data.secret))")
		}
		if !strings.Contains(hypothesis, "LOWER") && !strings.Contains(hypothesis, "UPPER") {
			alts = append(alts, "Try case normalization: LOWER(CONCAT(...))")
		}
	}
	if len(alts) == 0 {
		alts = append(alts, "Try varying the concatenation order or adding additional fields.")
	}
	return alts
}

// requestDataAccessor implements requestAccessor for the hypothesis parser.
type requestDataAccessor struct {
	req        *storage.Request
	_extracted map[string]string
}

func (a *requestDataAccessor) BodyText() string {
	return string(storage.DecodeBodyBytes(a.req.Body, a.req.Headers))
}

func (a *requestDataAccessor) HeaderText(name string) string {
	vals := a.req.Headers[name]
	if len(vals) > 0 {
		return strings.Join(vals, ", ")
	}
	return ""
}

func (a *requestDataAccessor) QueryText(key string) string {
	if a.req.QueryString != "" {
		values, _ := url.ParseQuery(a.req.QueryString)
		if v, ok := values[key]; ok && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func (a *requestDataAccessor) RespBodyText() string {
	return string(storage.DecodeBodyBytes(a.req.RespBody, a.req.RespHeaders))
}

func (a *requestDataAccessor) RespHeaderText(name string) string {
	vals := a.req.RespHeaders[name]
	if len(vals) > 0 {
		return strings.Join(vals, ", ")
	}
	return ""
}

func (a *requestDataAccessor) CookieText(key string) string {
	if v, ok := a.req.Cookies[key]; ok {
		return v
	}
	return ""
}

func (a *requestDataAccessor) inputsUsed() map[string]string {
	return map[string]string{}
}
