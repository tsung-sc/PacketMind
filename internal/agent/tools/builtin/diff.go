package builtin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

type requestDiffField struct {
	Field    string `json:"field"`
	ValueA   any    `json:"value_a,omitempty"`
	ValueB   any    `json:"value_b,omitempty"`
	Same     bool   `json:"same"`
	Category string `json:"category,omitempty"`
}

var defaultDiffFieldsList = []string{"meta", "headers", "body", "cookies", "query", "timing"}

func newDiffRequestsHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		requestIDA, err := tools.GetRequiredStringArg(args, "request_id_a")
		if err != nil {
			return nil, err
		}
		requestIDB, err := tools.GetRequiredStringArg(args, "request_id_b")
		if err != nil {
			return nil, err
		}
		fields := tools.ParseDiffFieldsArg(args, "fields")
		if len(fields) == 0 {
			fields = []string{"all"}
		}

		reqA, err := store.GetRequest(sessionID, requestIDA)
		if err != nil {
			return nil, fmt.Errorf("failed to get request_id_a: %w", err)
		}
		reqB, err := store.GetRequest(sessionID, requestIDB)
		if err != nil {
			return nil, fmt.Errorf("failed to get request_id_b: %w", err)
		}

		selected := normalizeDiffFields(fields)
		diffs := make([]requestDiffField, 0, 16)

		appendScalarDiff := func(field, category string, va, vb any) {
			aJSON := mustMarshalJSON(va)
			bJSON := mustMarshalJSON(vb)
			diffs = append(diffs, requestDiffField{Field: field, ValueA: va, ValueB: vb, Same: aJSON == bJSON, Category: category})
		}

		for _, f := range selected {
			switch f {
			case "meta":
				appendScalarDiff("method", "meta", reqA.Method, reqB.Method)
				appendScalarDiff("scheme", "meta", reqA.Scheme, reqB.Scheme)
				appendScalarDiff("host", "meta", reqA.Host, reqB.Host)
				appendScalarDiff("port", "meta", reqA.Port, reqB.Port)
				appendScalarDiff("path", "meta", reqA.Path, reqB.Path)
				appendScalarDiff("url", "meta", reqA.URL, reqB.URL)
				appendScalarDiff("status_code", "meta", reqA.StatusCode, reqB.StatusCode)
				appendScalarDiff("status_reason", "meta", reqA.StatusReason, reqB.StatusReason)
			case "headers":
				appendScalarDiff("headers", "headers", normalizeHeaders(reqA.Headers), normalizeHeaders(reqB.Headers))
				appendScalarDiff("response_headers", "headers", normalizeHeaders(reqA.RespHeaders), normalizeHeaders(reqB.RespHeaders))
			case "body":
				appendScalarDiff("body_preview", "body", previewBody(reqA.Body, reqA.ContentType, reqA.Headers), previewBody(reqB.Body, reqB.ContentType, reqB.Headers))
				appendScalarDiff("response_body_preview", "body", previewBody(reqA.RespBody, reqA.RespContentType, reqA.RespHeaders), previewBody(reqB.RespBody, reqB.RespContentType, reqB.RespHeaders))
				appendScalarDiff("content_type", "body", reqA.ContentType, reqB.ContentType)
				appendScalarDiff("response_content_type", "body", reqA.RespContentType, reqB.RespContentType)
			case "cookies":
				appendScalarDiff("cookies", "cookies", reqA.Cookies, reqB.Cookies)
			case "query":
				appendScalarDiff("query_string", "query", reqA.QueryString, reqB.QueryString)
			case "timing":
				appendScalarDiff("duration", "timing", reqA.Duration, reqB.Duration)
				appendScalarDiff("dns_duration", "timing", reqA.DNSDuration, reqB.DNSDuration)
				appendScalarDiff("connect_duration", "timing", reqA.ConnectDuration, reqB.ConnectDuration)
				appendScalarDiff("tls_handshake_duration", "timing", reqA.TLSHandshakeDuration, reqB.TLSHandshakeDuration)
				appendScalarDiff("request_duration", "timing", reqA.RequestDuration, reqB.RequestDuration)
				appendScalarDiff("response_duration", "timing", reqA.ResponseDuration, reqB.ResponseDuration)
				appendScalarDiff("latency_duration", "timing", reqA.LatencyDuration, reqB.LatencyDuration)
			}
		}

		diffOnly := make([]requestDiffField, 0, len(diffs))
		for _, item := range diffs {
			if !item.Same {
				diffOnly = append(diffOnly, item)
			}
		}

		content := mustMarshalJSON(map[string]any{
			"ok":              true,
			"request_id_a":    requestIDA,
			"request_id_b":    requestIDB,
			"selected_fields": selected,
			"diff_count":      len(diffOnly),
			"same_count":      len(diffs) - len(diffOnly),
			"diffs":           diffOnly,
		})

		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Compared %s and %s, found %d differences", requestIDA, requestIDB, len(diffOnly)),
			RequestIDs: []string{requestIDA, requestIDB},
		}, nil
	}
}

func normalizeDiffFields(raw []string) []string {
	allowed := map[string]struct{}{"meta": {}, "headers": {}, "body": {}, "cookies": {}, "query": {}, "timing": {}, "all": {}}
	if len(raw) == 0 {
		return defaultDiffFields()
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		field := strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowed[field]; !ok {
			continue
		}
		if field == "all" {
			return defaultDiffFields()
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	if len(out) == 0 {
		return defaultDiffFields()
	}
	return out
}

func defaultDiffFields() []string {
	return append([]string(nil), defaultDiffFieldsList...)
}

func normalizeHeaders(headers storage.Headers) map[string][]string {
	if len(headers) == 0 {
		return nil
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make(map[string][]string, len(keys))
	for _, key := range keys {
		values := append([]string(nil), headers[key]...)
		sort.Strings(values)
		result[strings.ToLower(key)] = values
	}
	return result
}
