package builtin

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

type searchAllFieldsResult struct {
	RequestID  string `json:"request_id"`
	SessionID  string `json:"session_id,omitempty"`
	Field      string `json:"field"`
	Preview    string `json:"preview,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	Method     string `json:"method,omitempty"`
	Host       string `json:"host,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

func newSearchByHeaderHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		headerName, err := tools.GetRequiredStringArg(args, "header_name")
		if err != nil {
			return nil, err
		}
		headerValue, err := tools.GetRequiredStringArg(args, "header_value")
		if err != nil {
			return nil, err
		}
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)

		requests, err := store.SearchRequestsByHeader(sessionID, headerName, headerValue, tools.NormalizeLimit(limit))
		if err != nil {
			return nil, fmt.Errorf("failed to search requests by header: %w", err)
		}
		return formatSearchResult(requests, "Header match results", headerName, headerValue, func(req *storage.Request) agentRequestMatch {
			preview := truncateForModel(strings.Join(req.Headers[headerName], ", "), maxSearchPreviewChars)
			return makeRequestMatch(req, req.SessionID, headerName, fmt.Sprintf("header %s matched", headerName), preview)
		})
	}
}

func newSearchByBodyHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		value, err := tools.GetRequiredStringArg(args, "value")
		if err != nil {
			return nil, err
		}
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)

		requests, err := store.SearchRequestsByBodyContent(sessionID, value, tools.NormalizeLimit(limit))
		if err != nil {
			return nil, fmt.Errorf("failed to search requests by body: %w", err)
		}
		return formatSearchResult(requests, "Request body match results", "body", value, func(req *storage.Request) agentRequestMatch {
			return makeRequestMatch(req, req.SessionID, "body", "request body matched", truncateForModel(previewBody(req.Body, req.ContentType, req.Headers), maxSearchPreviewChars))
		})
	}
}

func newSearchByResponseHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		value, err := tools.GetRequiredStringArg(args, "value")
		if err != nil {
			return nil, err
		}
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)

		requests, err := store.SearchRequestsByResponseBody(sessionID, value, tools.NormalizeLimit(limit))
		if err != nil {
			return nil, fmt.Errorf("failed to search requests by response body: %w", err)
		}
		return formatSearchResult(requests, "Response body match results", "response_body", value, func(req *storage.Request) agentRequestMatch {
			return makeRequestMatch(req, req.SessionID, "response_body", "response body matched", truncateForModel(previewBody(req.RespBody, req.RespContentType, req.RespHeaders), maxSearchPreviewChars))
		})
	}
}

func newSearchAllFieldsHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		value, err := tools.GetRequiredStringArg(args, "value")
		if err != nil {
			return nil, err
		}
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)

		results, errorSummary, err := searchAllFields(store, sessionID, value, limit)
		if err != nil {
			return nil, err
		}

		requestIDs := make([]string, 0, len(results))
		seen := make(map[string]struct{})
		for _, item := range results {
			if _, ok := seen[item.RequestID]; ok {
				continue
			}
			seen[item.RequestID] = struct{}{}
			requestIDs = append(requestIDs, item.RequestID)
		}

		content := mustMarshalJSON(map[string]any{
			"ok":            true,
			"session_id":    sessionID,
			"value":         value,
			"found_results": results,
			"count":         len(results),
			"error_summary": errorSummary,
		})

		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Found %d matches across all request fields", len(results)),
			RequestIDs: requestIDs,
		}, nil
	}
}

func searchAllFields(store *storage.Storage, sessionID, value string, limit int) ([]searchAllFieldsResult, []string, error) {
	needle := strings.TrimSpace(value)
	if needle == "" {
		return nil, nil, fmt.Errorf("value is required")
	}

	requests, err := store.ListRequests(storage.RequestListOptions{SessionID: sessionID, SortBy: "created_at", SortOrder: "desc"})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list requests: %w", err)
	}

	maxResults := tools.NormalizeLimit(limit)
	results := make([]searchAllFieldsResult, 0, maxResults)
	errorSet := make(map[string]struct{})

	for _, req := range requests {
		if req == nil {
			continue
		}
		budget := maxResults - len(results)
		if budget <= 0 {
			break
		}

		found, warnings := collectMatchesFromRequest(req, sessionID, needle, budget)
		results = append(results, found...)
		for _, warn := range warnings {
			errorSet[warn] = struct{}{}
		}
	}

	errorSummary := make([]string, 0, len(errorSet))
	for warn := range errorSet {
		errorSummary = append(errorSummary, warn)
	}
	sort.Strings(errorSummary)

	return results, errorSummary, nil
}

func collectMatchesFromRequest(req *storage.Request, sessionID, needle string, budget int) ([]searchAllFieldsResult, []string) {
	if budget <= 0 {
		return nil, nil
	}

	matches := make([]searchAllFieldsResult, 0, budget)
	warnings := make([]string, 0, 1)

	appendMatch := func(field, preview string) {
		if len(matches) >= budget {
			return
		}
		matches = append(matches, searchAllFieldsResult{
			RequestID:  req.ID,
			SessionID:  sessionID,
			Field:      field,
			Preview:    truncateForModel(strings.TrimSpace(preview), 200),
			CreatedAt:  req.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
			Method:     req.Method,
			Host:       req.Host,
			Path:       req.Path,
			StatusCode: req.StatusCode,
		})
	}

	for key, values := range req.Headers {
		joined := strings.Join(values, ", ")
		if containsFold(key, needle) || containsFold(joined, needle) {
			appendMatch("headers."+key, joined)
		}
	}
	for key, values := range req.RespHeaders {
		joined := strings.Join(values, ", ")
		if containsFold(key, needle) || containsFold(joined, needle) {
			appendMatch("response_headers."+key, joined)
		}
	}
	for key, val := range req.Cookies {
		if containsFold(key, needle) || containsFold(val, needle) {
			appendMatch("cookies."+key, val)
		}
	}

	if req.QueryString != "" {
		if containsFold(req.QueryString, needle) {
			appendMatch("query", req.QueryString)
			values, err := url.ParseQuery(req.QueryString)
			if err == nil {
				for key, items := range values {
					joined := strings.Join(items, ", ")
					if containsFold(key, needle) || containsFold(joined, needle) {
						appendMatch("query."+key, joined)
					}
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("query parse failed for request %s: %v", req.ID, err))
				appendMatch("query_string", req.QueryString)
			}
		}
	}

	bodyPreview := previewBody(req.Body, req.ContentType, req.Headers)
	if containsFold(bodyPreview, needle) {
		appendMatch("body", bodyPreview)
	}

	respPreview := previewBody(req.RespBody, req.RespContentType, req.RespHeaders)
	if containsFold(respPreview, needle) {
		appendMatch("response_body", respPreview)
	}

	return matches, warnings
}
