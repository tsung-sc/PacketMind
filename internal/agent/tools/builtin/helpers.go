package builtin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

const (
	defaultSearchLimit    = 10
	maxBodyPreviewChars   = 8000
	maxSearchPreviewChars = 1000
	maxHeaderPreviewCount = 0
	maxHeaderValueChars   = 2000
)

type agentRequestSnapshot struct {
	ID                    string                 `json:"id"`
	SessionID             string                 `json:"session_id,omitempty"`
	CreatedAt             string                 `json:"created_at,omitempty"`
	Method                string                 `json:"method,omitempty"`
	Scheme                string                 `json:"scheme,omitempty"`
	Host                  string                 `json:"host,omitempty"`
	Port                  int                    `json:"port,omitempty"`
	Path                  string                 `json:"path,omitempty"`
	URL                   string                 `json:"url,omitempty"`
	QueryString           string                 `json:"query_string,omitempty"`
	HTTPVersion           string                 `json:"http_version,omitempty"`
	StatusCode            int                    `json:"status_code,omitempty"`
	StatusReason          string                 `json:"status_reason,omitempty"`
	ContentType           string                 `json:"content_type,omitempty"`
	ResponseContentType   string                 `json:"response_content_type,omitempty"`
	Headers               map[string][]string    `json:"headers,omitempty"`
	ResponseHeaders       map[string][]string    `json:"response_headers,omitempty"`
	BodyPreview           string                 `json:"body_preview,omitempty"`
	ResponseBodyPreview   string                 `json:"response_body_preview,omitempty"`
	Duration              int64                  `json:"duration,omitempty"`
	ClientAddr            string                 `json:"client_addr,omitempty"`
	ServerAddr            string                 `json:"server_addr,omitempty"`
	RemoteAddr            string                 `json:"remote_addr,omitempty"`
	TLSVersion            string                 `json:"tls_version,omitempty"`
	TLSCipherSuite        string                 `json:"tls_cipher_suite,omitempty"`
	TLSServerName         string                 `json:"tls_server_name,omitempty"`
	TLSALPN               string                 `json:"tls_alpn,omitempty"`
	Timing                map[string]int64       `json:"timing,omitempty"`
	TLSServerExtensions   []storage.TLSExtension `json:"tls_server_extensions,omitempty"`
	TLSServerCertificates int                    `json:"tls_server_certificates,omitempty"`
}

type agentRequestMatch struct {
	RequestID   string `json:"request_id"`
	SessionID   string `json:"session_id,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	Method      string `json:"method,omitempty"`
	Host        string `json:"host,omitempty"`
	Path        string `json:"path,omitempty"`
	URL         string `json:"url,omitempty"`
	StatusCode  int    `json:"status_code,omitempty"`
	MatchField  string `json:"match_field,omitempty"`
	MatchReason string `json:"match_reason,omitempty"`
	Preview     string `json:"preview,omitempty"`
}

func makeRequestSnapshot(req *storage.Request, sessionID string) agentRequestSnapshot {
	headers := limitHeaders(req.Headers, maxHeaderPreviewCount)
	respHeaders := limitHeaders(req.RespHeaders, maxHeaderPreviewCount)

	return agentRequestSnapshot{
		ID:                  req.ID,
		SessionID:           sessionID,
		CreatedAt:           req.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Method:              req.Method,
		Scheme:              req.Scheme,
		Host:                req.Host,
		Port:                req.Port,
		Path:                req.Path,
		URL:                 req.URL,
		QueryString:         req.QueryString,
		HTTPVersion:         req.HTTPVersion,
		StatusCode:          req.StatusCode,
		StatusReason:        req.StatusReason,
		ContentType:         req.ContentType,
		ResponseContentType: req.RespContentType,
		Headers:             headers,
		ResponseHeaders:     respHeaders,
		BodyPreview:         previewBody(req.Body, req.ContentType, req.Headers),
		ResponseBodyPreview: previewBody(req.RespBody, req.RespContentType, req.RespHeaders),
		Duration:            req.Duration,
		ClientAddr:          req.ClientAddr,
		ServerAddr:          req.ServerAddr,
		RemoteAddr:          req.RemoteAddr,
		TLSVersion:          req.TLSVersion,
		TLSCipherSuite:      req.TLSCipherSuite,
		TLSServerName:       req.TLSServerName,
		TLSALPN:             req.TLSALPN,
		Timing: map[string]int64{
			"dns_duration":           req.DNSDuration,
			"connect_duration":       req.ConnectDuration,
			"tls_handshake_duration": req.TLSHandshakeDuration,
			"request_duration":       req.RequestDuration,
			"response_duration":      req.ResponseDuration,
			"latency_duration":       req.LatencyDuration,
		},
		TLSServerExtensions:   append([]storage.TLSExtension(nil), req.TLSServerExtensions...),
		TLSServerCertificates: len(req.TLSServerCertificates),
	}
}

func makeRequestMatch(req *storage.Request, sessionID, field, reason, preview string) agentRequestMatch {
	return agentRequestMatch{
		RequestID:   req.ID,
		SessionID:   sessionID,
		CreatedAt:   req.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Method:      req.Method,
		Host:        req.Host,
		Path:        req.Path,
		URL:         req.URL,
		StatusCode:  req.StatusCode,
		MatchField:  field,
		MatchReason: reason,
		Preview:     preview,
	}
}

func formatSearchResult(requests []*storage.Request, label, field, value string, buildMatch func(req *storage.Request) agentRequestMatch) (*agentruntime.ToolExecutionResult, error) {
	matches := make([]agentRequestMatch, 0, len(requests))
	requestIDs := make([]string, 0, len(requests))
	for _, req := range requests {
		if req == nil {
			continue
		}
		matches = append(matches, buildMatch(req))
		requestIDs = append(requestIDs, req.ID)
	}

	content := mustMarshalJSON(map[string]any{
		"ok":      true,
		"field":   field,
		"value":   value,
		"count":   len(matches),
		"results": matches,
	})

	return &agentruntime.ToolExecutionResult{
		Content:    content,
		Summary:    fmt.Sprintf("Found %d %s (%s)", len(matches), label, value),
		RequestIDs: requestIDs,
	}, nil
}

func previewBody(body []byte, contentType string, headers storage.Headers) string {
	if len(body) == 0 {
		return ""
	}

	body = storage.DecodeBodyBytes(body, headers)

	if isTextBody(body, contentType) {
		text := string(body)
		if len(text) > maxBodyPreviewChars {
			return text[:maxBodyPreviewChars] +
				fmt.Sprintf("\n...[body truncated: %d total bytes. Call get_request with include_full_body=true to see full content]", len(text))
		}
		return text
	}

	encoded := base64.StdEncoding.EncodeToString(body)
	return fmt.Sprintf("[binary body, %d bytes, base64 preview=%s]", len(body), truncateForModel(encoded, 256))
}

func fullBodyText(body []byte, contentType string, headers storage.Headers) string {
	if len(body) == 0 {
		return ""
	}

	body = storage.DecodeBodyBytes(body, headers)

	if isTextBody(body, contentType) {
		return string(body)
	}

	encoded := base64.StdEncoding.EncodeToString(body)
	return fmt.Sprintf("[binary body, %d bytes, base64=%s]", len(body), encoded)
}

func isTextBody(body []byte, contentType string) bool {
	ct := strings.ToLower(contentType)
	if strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "xml") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "html") ||
		strings.Contains(ct, "x-www-form-urlencoded") ||
		strings.Contains(ct, "graphql") {
		return true
	}
	return utf8.Valid(body)
}

func limitHeaders(headers storage.Headers, limit int) map[string][]string {
	if len(headers) == 0 {
		return nil
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}

	out := make(map[string][]string, len(keys))
	for _, key := range keys {
		values := headers[key]
		copied := append([]string(nil), values...)
		for i := range copied {
			copied[i] = truncateForModel(copied[i], maxHeaderValueChars)
		}
		out[key] = copied
	}
	return out
}

func containsFold(haystack, needle string) bool {
	if strings.TrimSpace(needle) == "" {
		return false
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func truncateForModel(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "...(truncated)"
}

func MustMarshalJSON(value any) string {
	return mustMarshalJSON(value)
}

func mustMarshalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `{"ok":false,"error":"failed to encode json"}`
	}
	return string(data)
}

func MakeRequestSnapshot(req *storage.Request, sessionID string) any {
	return makeRequestSnapshot(req, sessionID)
}
