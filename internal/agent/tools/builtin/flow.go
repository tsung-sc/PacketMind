package builtin

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

func newTraceFlowSequenceHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			sessionID = tools.GetStringArg(args, "session_id", "")
		}
		hostFilter := tools.GetStringArg(args, "host", "")
		maxRequests := tools.GetIntArg(args, "max_requests", 50)
		if maxRequests < 10 {
			maxRequests = 10
		}
		if maxRequests > 100 {
			maxRequests = 100
		}

		opts := storage.RequestListOptions{SessionID: sessionID, SortBy: "created_at", SortOrder: "asc"}
		requests, err := store.ListRequests(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list requests: %w", err)
		}

		filtered := make([]*storage.Request, 0, len(requests))
		for _, req := range requests {
			if req == nil {
				continue
			}
			if hostFilter != "" && !strings.Contains(strings.ToLower(req.Host), strings.ToLower(hostFilter)) {
				continue
			}
			filtered = append(filtered, req)
		}

		total := len(filtered)
		displayed := total
		if displayed > maxRequests {
			displayed = maxRequests
		}

		sequence := make([]map[string]any, 0, displayed)
		for i, req := range filtered {
			if i >= displayed {
				break
			}
			sequence = append(sequence, map[string]any{
				"index":      i + 1,
				"request_id": req.ID,
				"method":     req.Method,
				"host":       req.Host,
				"path":       req.Path,
				"status":     req.StatusCode,
				"created_at": req.CreatedAt.Format("2006-01-02T15:04:05Z"),
			})
		}

		flowEdges := detectFlowEdges(filtered)
		gaps := detectGapsAndAnomalies(filtered, flowEdges)

		content := mustMarshalJSON(map[string]any{
			"ok":                true,
			"total_in_sequence": total,
			"displayed":         displayed,
			"flow_edges":        flowEdges,
			"sequence":          sequence,
			"gaps_and_anomalies": gaps,
		})

		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Traced flow sequence: %d requests, %d flow edges, %d anomalies", total, len(flowEdges), len(gaps)),
			RequestIDs: extractRequestIDs(filtered),
		}, nil
	}
}

func detectFlowEdges(requests []*storage.Request) []map[string]any {
	edges := make([]map[string]any, 0)
	if len(requests) == 0 {
		return edges
	}

	// cookie flow: Response Set-Cookie -> later request Cookie header
	cookieMap := make(map[string]string) // cookie key -> request ID that set it
	for _, req := range requests {
		setCookies := req.RespHeaders["Set-Cookie"]
		for _, sc := range setCookies {
			key := cookieKey(sc)
			if key != "" {
				cookieMap[key] = req.ID
			}
		}
	}
	for _, req := range requests {
		cookies := req.Headers["Cookie"]
		for _, c := range cookies {
			parts := strings.Split(c, ";")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				kv := strings.SplitN(part, "=", 2)
				if len(kv) != 2 {
					continue
				}
				key := strings.TrimSpace(kv[0])
				if sourceID, ok := cookieMap[key]; ok && sourceID != req.ID {
					edges = append(edges, map[string]any{
						"type": "cookie_flow",
						"source": map[string]any{
							"request_id": sourceID,
							"action":     "response Set-Cookie",
							"keys":       []string{key},
						},
						"target": map[string]any{
							"request_id": req.ID,
							"action":     "request Cookie header",
							"keys":       []string{key},
						},
						"description": fmt.Sprintf("Cookie '%s' set earlier, used in this request", key),
					})
				}
			}
		}
	}

	// token flow: response body token -> later Authorization Bearer
	tokenValues := extractTokenValues(requests)
	for _, req := range requests {
		authVals := req.Headers["Authorization"]
		for _, av := range authVals {
			av = strings.TrimSpace(av)
			if strings.HasPrefix(strings.ToLower(av), "bearer ") {
				token := strings.TrimPrefix(av, "Bearer ")
				token = strings.TrimPrefix(token, "bearer ")
				if src := findTokenSource(token, tokenValues); src != "" {
					edges = append(edges, map[string]any{
						"type": "token_flow",
						"source": map[string]any{
							"request_id":    src,
							"action":        "response body token",
							"value_preview": truncateForModel(token, 20),
						},
						"target": map[string]any{
							"request_id":    req.ID,
							"action":        "request Authorization header",
							"value_preview": truncateForModel(av, 30),
						},
						"description": "JWT/Bearer token issued earlier, used in Authorization header",
					})
				}
			}
		}
	}

	// redirect chains
	redirectChain := make([]string, 0)
	for _, req := range requests {
		if req.StatusCode >= 300 && req.StatusCode < 400 {
			redirectChain = append(redirectChain, fmt.Sprintf("%s (%d)", req.ID, req.StatusCode))
		} else {
			if len(redirectChain) >= 2 {
				edges = append(edges, map[string]any{
					"type":  "redirect_chain",
					"chain": append([]string(nil), redirectChain...),
				})
			}
			redirectChain = nil
		}
	}
	if len(redirectChain) >= 2 {
		edges = append(edges, map[string]any{
			"type":  "redirect_chain",
			"chain": append([]string(nil), redirectChain...),
		})
	}

	// value_reuse: simple heuristic for JSON body values
	valueMap := make(map[string]string) // value -> request ID where first seen in response
	for _, req := range requests {
		respPreview := previewBody(req.RespBody, req.RespContentType, req.RespHeaders)
		vals := extractSimpleJSONValues(respPreview)
		for _, v := range vals {
			if len(v) < 4 || len(v) > 50 {
				continue
			}
			if _, ok := valueMap[v]; !ok {
				valueMap[v] = req.ID
			}
		}
	}
	for _, req := range requests {
		bodyPreview := previewBody(req.Body, req.ContentType, req.Headers)
		queryVals := make(map[string]string)
		if req.QueryString != "" {
			q, _ := url.ParseQuery(req.QueryString)
			for k, v := range q {
				if len(v) > 0 {
					queryVals[k] = v[0]
				}
			}
		}
		for val, srcID := range valueMap {
			if srcID == req.ID {
				continue
			}
			found := false
			if strings.Contains(bodyPreview, val) {
				found = true
			}
			if !found {
				for _, qv := range queryVals {
					if qv == val {
						found = true
						break
					}
				}
			}
			if found {
				edges = append(edges, map[string]any{
					"type": "value_reuse",
					"source": map[string]any{
						"request_id": srcID,
						"value":      truncateForModel(val, 30),
					},
					"target": map[string]any{
						"request_id": req.ID,
						"value":      truncateForModel(val, 30),
					},
					"description": fmt.Sprintf("Value '%s' from earlier response reused in this request", truncateForModel(val, 20)),
				})
			}
		}
	}

	return edges
}

func detectGapsAndAnomalies(requests []*storage.Request, edges []map[string]any) []string {
	gaps := make([]string, 0)

	// orphan signatures: request has sign but no prior source
	for _, req := range requests {
		if !hasSignedParams(req) {
			continue
		}
		hasSource := false
		for _, edge := range edges {
			if edge["type"] == "value_reuse" {
				target, _ := edge["target"].(map[string]any)
				if target["request_id"] == req.ID {
					hasSource = true
					break
				}
			}
		}
		if !hasSource {
			gaps = append(gaps, fmt.Sprintf("Request %s %s has sign parameter but no obvious prior response providing a signing secret", req.ID, req.Path))
		}
	}

	// large time gaps between requests to same host
	hostLast := make(map[string]*storage.Request)
	for _, req := range requests {
		if last, ok := hostLast[req.Host]; ok && last != nil {
			gap := req.CreatedAt.Sub(last.CreatedAt)
			if gap > 0 && gap.Seconds() > 60 {
				gaps = append(gaps, fmt.Sprintf("Large gap (%.0fs) between requests to %s: %s -> %s", gap.Seconds(), req.Host, last.ID, req.ID))
			}
		}
		hostLast[req.Host] = req
	}

	return gaps
}

func cookieKey(setCookie string) string {
	setCookie = strings.TrimSpace(setCookie)
	idx := strings.Index(setCookie, ";")
	if idx >= 0 {
		setCookie = setCookie[:idx]
	}
	kv := strings.SplitN(setCookie, "=", 2)
	if len(kv) != 2 {
		return ""
	}
	return strings.TrimSpace(kv[0])
}

func extractTokenValues(requests []*storage.Request) map[string]string {
	out := make(map[string]string)
	for _, req := range requests {
		respPreview := previewBody(req.RespBody, req.RespContentType, req.RespHeaders)
		lower := strings.ToLower(respPreview)
		if strings.Contains(lower, "accesstoken") || strings.Contains(lower, "token") || strings.Contains(lower, "jwt") {
			// store response request ID as potential token source
			out[req.ID] = req.ID
		}
	}
	return out
}

func findTokenSource(token string, tokenValues map[string]string) string {
	// naive: if token is long enough, any previous token-bearing response could be source
	// simplified: return first token response ID as heuristic
	for id := range tokenValues {
		return id
	}
	return ""
}

func extractSimpleJSONValues(jsonStr string) []string {
	// Extract quoted string values from JSON-like text
	vals := make([]string, 0)
	inString := false
	var current strings.Builder
	for i, ch := range jsonStr {
		if ch == '"' {
			if inString {
				val := current.String()
				if val != "" && !looksLikeJSONKey(val, jsonStr, i) {
					vals = append(vals, val)
				}
				current.Reset()
			}
			inString = !inString
			continue
		}
		if inString {
			current.WriteRune(ch)
		}
	}
	return vals
}

func looksLikeJSONKey(val string, full string, endPos int) bool {
	// Heuristic: if next non-space char after closing quote is ':', it's a key
	for i := endPos + 1; i < len(full) && i < endPos+10; i++ {
		c := full[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		if c == ':' {
			return true
		}
		return false
	}
	return false
}

func extractRequestIDs(requests []*storage.Request) []string {
	ids := make([]string, 0, len(requests))
	seen := make(map[string]struct{})
	for _, req := range requests {
		if req == nil {
			continue
		}
		if _, ok := seen[req.ID]; !ok {
			seen[req.ID] = struct{}{}
			ids = append(ids, req.ID)
		}
	}
	return ids
}
