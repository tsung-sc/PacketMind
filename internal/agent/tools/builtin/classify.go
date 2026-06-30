package builtin

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

func newClassifyRequestsHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			sessionID = tools.GetStringArg(args, "session_id", "")
		}
		hostFilter := tools.GetStringArg(args, "host", "")
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)
		limit = tools.NormalizeLimit(limit)

		opts := storage.RequestListOptions{SessionID: sessionID, SortBy: "created_at", SortOrder: "asc"}
		requests, err := store.ListRequests(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list requests: %w", err)
		}

		categories := newClassifyCategories()
		for _, req := range requests {
			if req == nil {
				continue
			}
			if hostFilter != "" && !strings.Contains(strings.ToLower(req.Host), strings.ToLower(hostFilter)) {
				continue
			}
			cat := classifySingleRequest(req)
			categories.add(cat, req)
		}

		catsMap := categories.toMap(limit)
		suggestion := generateAutoFocusSuggestion(categories)

		content := mustMarshalJSON(map[string]any{
			"ok":                   true,
			"session_id":           sessionID,
			"categories":           catsMap,
			"auto_focus_suggestion": suggestion,
		})

		return &agentruntime.ToolExecutionResult{
			Content: content,
			Summary: fmt.Sprintf("Classified %d requests into %d categories", len(requests), len(catsMap)),
		}, nil
	}
}

type classifyCategories struct {
	AuthEntry      *classifyCategory
	TokenIssuance  *classifyCategory
	SignedRequest  *classifyCategory
	DataQuery      *classifyCategory
	AuthRequest    *classifyCategory
	ConfigFetch    *classifyCategory
	StaticResource *classifyCategory
	Redirect       *classifyCategory
	Error          *classifyCategory
	Other          *classifyCategory
}

type classifyCategory struct {
	Label       string                     `json:"label"`
	Description string                     `json:"description"`
	Count       int                        `json:"count"`
	Requests    []classifyRequestSummary   `json:"requests,omitempty"`
	Hosts       []string                   `json:"hosts,omitempty"`
}

type classifyRequestSummary struct {
	RequestID   string   `json:"request_id"`
	Method      string   `json:"method"`
	Host        string   `json:"host"`
	Path        string   `json:"path"`
	Status      int      `json:"status"`
	Indicators  []string `json:"indicators"`
}

func newClassifyCategories() *classifyCategories {
	return &classifyCategories{
		AuthEntry:      &classifyCategory{Label: "Authentication Entry", Description: "Requests that appear to be login or registration endpoints"},
		TokenIssuance:  &classifyCategory{Label: "Token Issuance", Description: "Responses that issue cookies, tokens, or session identifiers"},
		SignedRequest:  &classifyCategory{Label: "Requests with Signatures", Description: "Requests containing sign, signature, hmac, or hash parameters"},
		DataQuery:      &classifyCategory{Label: "Data Query", Description: "API data queries and mutations"},
		AuthRequest:    &classifyCategory{Label: "Authenticated Request", Description: "Requests with Authorization or session cookies"},
		ConfigFetch:    &classifyCategory{Label: "Config Fetch", Description: "Configuration or bootstrap requests"},
		StaticResource: &classifyCategory{Label: "Static Resources", Description: "Static assets like JS, CSS, images, fonts"},
		Redirect:       &classifyCategory{Label: "Redirects", Description: "HTTP 3xx redirect responses"},
		Error:          &classifyCategory{Label: "Errors", Description: "4xx and 5xx responses"},
		Other:          &classifyCategory{Label: "Other", Description: "Requests that do not match other categories"},
	}
}

func (c *classifyCategories) add(cat string, req *storage.Request) {
	var target *classifyCategory
	switch cat {
	case "auth_entry":
		target = c.AuthEntry
	case "token_issuance":
		target = c.TokenIssuance
	case "signed_request":
		target = c.SignedRequest
	case "data_query":
		target = c.DataQuery
	case "auth_request":
		target = c.AuthRequest
	case "config_fetch":
		target = c.ConfigFetch
	case "static_resource":
		target = c.StaticResource
	case "redirect":
		target = c.Redirect
	case "error":
		target = c.Error
	default:
		target = c.Other
	}

	target.Count++
	indicators := buildIndicators(req, cat)
	target.Requests = append(target.Requests, classifyRequestSummary{
		RequestID:  req.ID,
		Method:     req.Method,
		Host:       req.Host,
		Path:       req.Path,
		Status:     req.StatusCode,
		Indicators: indicators,
	})

	// track unique hosts for static/config
	if cat == "static_resource" || cat == "config_fetch" {
		found := false
		for _, h := range target.Hosts {
			if h == req.Host {
				found = true
				break
			}
		}
		if !found {
			target.Hosts = append(target.Hosts, req.Host)
		}
	}
}

func (c *classifyCategories) toMap(limit int) map[string]any {
	out := make(map[string]any)
	addCat := func(key string, cat *classifyCategory) {
		if cat.Count == 0 {
			return
		}
		copyCat := *cat
		if len(copyCat.Requests) > limit {
			copyCat.Requests = copyCat.Requests[:limit]
		}
		if len(copyCat.Hosts) > 5 {
			copyCat.Hosts = copyCat.Hosts[:5]
		}
		out[key] = map[string]any{
			"label":       copyCat.Label,
			"description": copyCat.Description,
			"count":       copyCat.Count,
			"requests":    copyCat.Requests,
			"hosts":       copyCat.Hosts,
		}
	}
	addCat("auth_entry", c.AuthEntry)
	addCat("token_issuance", c.TokenIssuance)
	addCat("signed_request", c.SignedRequest)
	addCat("data_query", c.DataQuery)
	addCat("auth_request", c.AuthRequest)
	addCat("config_fetch", c.ConfigFetch)
	addCat("static_resource", c.StaticResource)
	addCat("redirect", c.Redirect)
	addCat("error", c.Error)
	addCat("other", c.Other)
	return out
}

func classifySingleRequest(req *storage.Request) string {
	// Priority order matters
	if isRedirect(req) {
		return "redirect"
	}
	if isError(req) {
		return "error"
	}
	if isAuthEntry(req) {
		return "auth_entry"
	}
	if isTokenIssuance(req) {
		return "token_issuance"
	}
	if isStaticResource(req) {
		return "static_resource"
	}
	if isConfigFetch(req) {
		return "config_fetch"
	}
	if isSignedRequest(req) {
		return "signed_request"
	}
	if isAuthRequest(req) {
		return "auth_request"
	}
	if isDataQuery(req) {
		return "data_query"
	}
	return "other"
}

func isAuthEntry(req *storage.Request) bool {
	path := strings.ToLower(req.Path)
	if !strings.Contains(path, "login") && !strings.Contains(path, "signin") &&
		!strings.Contains(path, "register") && !strings.Contains(path, "auth") {
		return false
	}
	bodyPreview := previewBody(req.Body, req.ContentType, req.Headers)
	lowerBody := strings.ToLower(bodyPreview)
	if strings.Contains(lowerBody, "password") || strings.Contains(lowerBody, "credential") {
		return true
	}
	// response sets cookie/token
	for key := range req.RespHeaders {
		if strings.EqualFold(key, "Set-Cookie") {
			return true
		}
	}
	respPreview := previewBody(req.RespBody, req.RespContentType, req.RespHeaders)
	lowerResp := strings.ToLower(respPreview)
	if strings.Contains(lowerResp, "accesstoken") || strings.Contains(lowerResp, "token") || strings.Contains(lowerResp, "session") {
		return true
	}
	return false
}

func isTokenIssuance(req *storage.Request) bool {
	for key := range req.RespHeaders {
		if strings.EqualFold(key, "Set-Cookie") {
			return true
		}
	}
	respPreview := previewBody(req.RespBody, req.RespContentType, req.RespHeaders)
	lowerResp := strings.ToLower(respPreview)
	if strings.Contains(lowerResp, "accesstoken") || strings.Contains(lowerResp, "token") || strings.Contains(lowerResp, "session") {
		return true
	}
	return false
}

func isSignedRequest(req *storage.Request) bool {
	return hasSignedParams(req)
}

func isDataQuery(req *storage.Request) bool {
	ct := strings.ToLower(req.ContentType)
	if strings.Contains(ct, "json") || strings.Contains(ct, "xml") {
		return true
	}
	return false
}

func isAuthRequest(req *storage.Request) bool {
	for key, vals := range req.Headers {
		lower := strings.ToLower(key)
		if lower == "authorization" || lower == "x-auth-token" {
			return true
		}
		if lower == "cookie" {
			for _, v := range vals {
				lowerV := strings.ToLower(v)
				if strings.Contains(lowerV, "session") || strings.Contains(lowerV, "token") || strings.Contains(lowerV, "aid") {
					return true
				}
			}
		}
	}
	return false
}

func isConfigFetch(req *storage.Request) bool {
	path := strings.ToLower(req.Path)
	return strings.Contains(path, "config") || strings.Contains(path, "bootstrap") || strings.Contains(path, "init")
}

func isStaticResource(req *storage.Request) bool {
	path := strings.ToLower(req.Path)
	staticExts := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".mp4", ".webp", ".html"}
	for _, ext := range staticExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	ct := strings.ToLower(req.ContentType)
	if strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "font/") || strings.Contains(ct, "css") || ct == "text/javascript" || ct == "application/javascript" {
		return true
	}
	return false
}

func isRedirect(req *storage.Request) bool {
	return req.StatusCode >= 300 && req.StatusCode < 400
}

func isError(req *storage.Request) bool {
	return req.StatusCode >= 400
}

func buildIndicators(req *storage.Request, cat string) []string {
	indicators := make([]string, 0, 3)
	switch cat {
	case "auth_entry":
		bodyPreview := previewBody(req.Body, req.ContentType, req.Headers)
		if strings.Contains(strings.ToLower(bodyPreview), "password") {
			indicators = append(indicators, "request body contains 'password'")
		}
		for key := range req.RespHeaders {
			if strings.EqualFold(key, "Set-Cookie") {
				indicators = append(indicators, "response sets session cookie")
			}
		}
		respPreview := previewBody(req.RespBody, req.RespContentType, req.RespHeaders)
		if strings.Contains(strings.ToLower(respPreview), "accesstoken") || strings.Contains(strings.ToLower(respPreview), "token") {
			indicators = append(indicators, "response body has token")
		}
	case "signed_request":
		if req.QueryString != "" {
			values, _ := url.ParseQuery(req.QueryString)
			for key := range values {
				if isSignedParamName(key) {
					indicators = append(indicators, fmt.Sprintf("query param '%s' looks like signature", key))
				}
			}
		}
	case "auth_request":
		for key := range req.Headers {
			if strings.EqualFold(key, "Authorization") {
				indicators = append(indicators, "has Authorization header")
			}
		}
	case "static_resource":
		indicators = append(indicators, fmt.Sprintf("static asset: %s", req.Path))
	case "redirect":
		indicators = append(indicators, fmt.Sprintf("status %d redirect", req.StatusCode))
	case "error":
		indicators = append(indicators, fmt.Sprintf("status %d error", req.StatusCode))
	}
	if len(indicators) == 0 {
		indicators = append(indicators, fmt.Sprintf("%s %s", req.Method, req.Path))
	}
	return indicators
}

func generateAutoFocusSuggestion(cats *classifyCategories) string {
	parts := make([]string, 0, 4)
	if cats.AuthEntry.Count > 0 {
		parts = append(parts, fmt.Sprintf("auth entry found (%d requests)", cats.AuthEntry.Count))
	}
	if cats.TokenIssuance.Count > 0 {
		parts = append(parts, fmt.Sprintf("token issuance detected (%d requests)", cats.TokenIssuance.Count))
	}
	if cats.SignedRequest.Count > 0 {
		parts = append(parts, fmt.Sprintf("%d signed requests", cats.SignedRequest.Count))
	}
	if cats.AuthRequest.Count > 0 {
		parts = append(parts, fmt.Sprintf("%d authenticated requests", cats.AuthRequest.Count))
	}

	if len(parts) == 0 {
		return "No clear auth or signature patterns detected. Consider using search_all_fields to look for specific values."
	}

	suggestion := "The session has " + strings.Join(parts, ", ") + ". "
	if cats.AuthEntry.Count > 0 {
		suggestion += "Suggested next step: use get_request on the auth entry request to inspect login flow."
	} else if cats.SignedRequest.Count > 0 {
		suggestion += "Suggested next step: use trace_flow_sequence to understand how signatures are used."
	} else {
		suggestion += "Suggested next step: use summarize_session for a broader view."
	}
	return suggestion
}
