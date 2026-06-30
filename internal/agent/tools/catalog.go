package tools

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

const (
	ToolGetRequest               = "get_request"
	ToolSearchRequestsByHeader   = "search_requests_by_header"
	ToolSearchRequestsByBody     = "search_requests_by_body"
	ToolSearchRequestsByResponse = "search_requests_by_response"
	ToolAnalyzeEncoding          = "analyze_encoding"
	ToolSearchAllFields          = "search_all_fields"
	ToolFindPriorResponseSources = "find_prior_response_sources"
	ToolFindLaterRequestUsages   = "find_later_request_usages"
	ToolTraceValueFlow           = "trace_value_flow"
	ToolDiffRequests             = "diff_requests"
	ToolBatchExecute             = "batch_execute"
	ToolBash                     = "bash"
	ToolSummarizeSession         = "summarize_session"
	ToolClassifyRequests         = "classify_requests"
	ToolTraceFlowSequence        = "trace_flow_sequence"
	ToolTestHypothesis           = "test_hypothesis"
)

const toolJSONSchemaExtraKey = "json_schema"

func BuiltInSchemas() []*llmtypes.ToolDefinition {
	return []*llmtypes.ToolDefinition{
		NewFunctionTool(ToolGetRequest, "Fetch the complete details of a specific HTTP request by its ID.\nReturns full request/response metadata including method, URL, headers, body previews, status code, timing, TLS info, and error details.\nUse this when you have a request ID (from search results or user input) and need to inspect its full content.\nSet include_full_body=true to get complete request/response body text without truncation — use this when the default body preview is insufficient for analysis.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"request_id":        map[string]any{"type": "string", "description": "The request ID to fetch details for"},
				"include_full_body": map[string]any{"type": "boolean", "description": "When true, returns complete request/response body text without truncation. Use when body preview is insufficient. Default false."},
			},
			"required": []string{"request_id"},
		}),
		NewFunctionTool(ToolSearchRequestsByHeader, "Search for requests containing a specific header name and value pattern.\nReturns matching requests with context about where the header was found.\nUse this to find requests with particular cookies, authorization headers, content-type, or custom headers. Case-insensitive header name matching.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id":   map[string]any{"type": "string", "description": "Optional session ID filter"},
				"header_name":  map[string]any{"type": "string", "description": "Request header name to match (case-insensitive)"},
				"header_value": map[string]any{"type": "string", "description": "Request header value fragment to match"},
				"limit":        map[string]any{"type": "integer", "description": "Maximum number of results to return (1-50)", "minimum": 1, "maximum": 50},
			},
			"required": []string{"header_name", "header_value"},
		}),
		NewFunctionTool(ToolSearchRequestsByBody, "Search for requests whose request body contains a given text fragment.\nReturns matching requests with body excerpts highlighting the match.\nUse this to find API calls sending specific JSON fields, form values, or payload patterns.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Optional session ID filter"},
				"value":      map[string]any{"type": "string", "description": "Content fragment to search in request bodies"},
				"limit":      map[string]any{"type": "integer", "description": "Maximum number of results to return (1-50)", "minimum": 1, "maximum": 50},
			},
			"required": []string{"value"},
		}),
		NewFunctionTool(ToolSearchRequestsByResponse, "Search for requests whose response body contains a given text fragment.\nReturns matching requests with response body excerpts.\nUse this to find API responses containing specific values like tokens, error messages, or data patterns.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Optional session ID filter"},
				"value":      map[string]any{"type": "string", "description": "Content fragment to search in response bodies"},
				"limit":      map[string]any{"type": "integer", "description": "Maximum number of results to return (1-50)", "minimum": 1, "maximum": 50},
			},
			"required": []string{"value"},
		}),
		NewFunctionTool(ToolAnalyzeEncoding, "Detect and decode nested encoding layers in a value (Base64, URL-encoding, HTML entities, hex, etc.).\nReturns the decoded chain with intermediate results.\nUse this when you encounter suspicious or encoded values in headers, cookies, query parameters, or body fields.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"value": map[string]any{"type": "string", "description": "Input value to inspect"},
			},
			"required": []string{"value"},
		}),
		NewFunctionTool(ToolSearchAllFields, "Search across ALL request fields simultaneously — headers, request body, response body, cookies, and query parameters — for a given value.\nReturns consolidated results showing where the value appears.\nUse this as a broad-first search when you're not sure which field contains the target value.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Optional session ID filter"},
				"value":      map[string]any{"type": "string", "description": "Content fragment to search across all fields"},
				"limit":      map[string]any{"type": "integer", "description": "Maximum number of matching requests to return (1-50)", "minimum": 1, "maximum": 50},
			},
			"required": []string{"value"},
		}),
		NewFunctionTool(ToolFindPriorResponseSources, "Trace backward in time: find earlier responses in the session whose response data contains a target value.\nReturns candidate source responses with timestamps and confidence indicators.\nUse this to answer 'where did this value come from?' — finding the upstream response that originally provided a token, cookie, or parameter.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id":        map[string]any{"type": "string", "description": "Optional session ID filter"},
				"before_request_id": map[string]any{"type": "string", "description": "Target request ID; only searches responses before this request"},
				"value":             map[string]any{"type": "string", "description": "Target value to locate in prior responses"},
				"limit":             map[string]any{"type": "integer", "description": "Maximum number of results to return (1-50)", "minimum": 1, "maximum": 50},
			},
			"required": []string{"before_request_id", "value"},
		}),
		NewFunctionTool(ToolFindLaterRequestUsages, "Trace forward in time: find later requests that reuse a target value in their request fields.\nReturns candidate requests showing where and how the value was used.\nUse this to answer 'where is this value consumed?' — tracking how a server-provided value propagates to subsequent client requests.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id":       map[string]any{"type": "string", "description": "Optional session ID filter"},
				"after_request_id": map[string]any{"type": "string", "description": "Source request ID; only searches requests after this one"},
				"value":            map[string]any{"type": "string", "description": "Target value to locate in later requests"},
				"limit":            map[string]any{"type": "integer", "description": "Maximum number of results to return (1-50)", "minimum": 1, "maximum": 50},
			},
			"required": []string{"after_request_id", "value"},
		}),
		NewFunctionTool(ToolTraceValueFlow, "End-to-end provenance tracing with automatic confidence scoring.\nGiven a target field in a request, traces its likely origin through earlier responses and scores each candidate by temporal, semantic, and field-name similarity.\nReturns a ranked provenance chain with confidence levels.\nUse this for comprehensive 'where did X come from?' investigations.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Optional session ID filter"},
				"request_id": map[string]any{"type": "string", "description": "Target request ID"},
				"field_name": map[string]any{"type": "string", "description": "Target field name"},
				"location": map[string]any{
					"type":        "string",
					"description": "Where the target field is located",
					"enum":        []string{"query", "header", "cookie", "form_body", "json_body"},
				},
				"limit": map[string]any{"type": "integer", "description": "Maximum number of candidate provenance links (1-50)", "minimum": 1, "maximum": 50},
			},
			"required": []string{"request_id", "field_name", "location"},
		}),
		NewFunctionTool(ToolDiffRequests, "Compare two captured requests field-by-field to identify differences.\nReturns a structured diff showing matching and differing fields across metadata, headers, body, cookies, query parameters, and timing.\nUse this to compare two similar requests (e.g., successful vs failed, or same endpoint different parameters).", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"request_id_a": map[string]any{"type": "string", "description": "First request ID to compare"},
				"request_id_b": map[string]any{"type": "string", "description": "Second request ID to compare"},
				"fields":       map[string]any{"description": "Optional list: meta, headers, body, cookies, query, timing, all"},
			},
			"required": []string{"request_id_a", "request_id_b"},
		}),
		NewFunctionTool(ToolBatchExecute, "Execute multiple tool calls in parallel and aggregate their results.\nAccepts an array of tool name + arguments pairs.\nUse this when you need to run several independent searches or lookups simultaneously — for example, searching multiple values or checking both headers and body at once.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Optional session ID fallback for nested calls"},
				"calls":      map[string]any{"description": "Array of tool call objects: [{\"tool\":\"name\",\"args\":\"{...}\"}]"},
			},
			"required": []string{"calls"},
		}),
		NewFunctionTool(ToolBash, "Execute a shell command in a sandboxed workspace directory and return the output.\nRuns in data/agent-workspace by default; use workdir to specify a different relative path.\nUse this for data transformation, file inspection, scripting, or any task that benefits from shell execution.\nOutput is truncated at 50000 bytes.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "Shell command to execute"},
				"workdir": map[string]any{"type": "string", "description": "Working directory for the command (default: data/agent-workspace)"},
				"timeout": map[string]any{"type": "integer", "description": "Timeout in seconds (1-300, default 60)", "minimum": 1, "maximum": 300},
			},
			"required": []string{"command"},
		}),
		NewFunctionTool(ToolSummarizeSession, "Get a high-level overview of a capture session. Returns host breakdowns, request counts, status distributions, content-type distributions, auth detection, and a timeline overview. Use this as the FIRST step in any analysis.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Optional session ID"},
			},
			"required": []string{},
		}),
		NewFunctionTool(ToolClassifyRequests, "Automatically classify requests in a session by their likely role: auth_entry, token_issuance, signed_request, data_query, auth_request, config_fetch, static_resource, redirect, error, other. Returns a categorized list with key findings.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string", "description": "Optional session ID filter"},
				"host":       map[string]any{"type": "string", "description": "Optional host filter"},
				"limit":      map[string]any{"type": "integer", "description": "Max results per category (1-50, default 10)", "minimum": 1, "maximum": 50},
			},
			"required": []string{},
		}),
		NewFunctionTool(ToolTraceFlowSequence, "List requests in chronological order and detect field flow relationships — cookie set/use, token issuance/consumption, redirect chains, value reuse. Annotates flow edges showing how data moves through the session.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id":   map[string]any{"type": "string", "description": "Optional session ID filter"},
				"host":         map[string]any{"type": "string", "description": "Optional host filter"},
				"focus_fields": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional field names to track"},
				"max_requests": map[string]any{"type": "integer", "description": "Max requests (10-100, default 50)", "minimum": 10, "maximum": 100},
			},
			"required": []string{},
		}),
		NewFunctionTool(ToolTestHypothesis, "Test a hypothesis about how a field value is generated. Given request IDs and a hypothesis composed of operations (field extraction, concatenation, hash, HMAC, base64, etc.), this tool tests against actual values and returns match rate.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"request_ids":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Request IDs to test (minimum 3)"},
				"target_field":    map[string]any{"type": "string", "description": "Field to verify, e.g. 'sign', 'Authorization', 'token'"},
				"target_location": map[string]any{"type": "string", "description": "'body', 'header', 'query', 'response_body', 'response_header', 'cookie'"},
				"hypothesis":      map[string]any{"type": "string", "description": "Structured generation rule. E.g. 'MD5(CONCAT(timestamp, body_raw))', 'HMAC_SHA256(CONCAT(method,path,timestamp), secret=EXTRACT(response_body,$.data.secret))'"},
			},
			"required": []string{"request_ids", "target_field", "target_location", "hypothesis"},
		}),
	}
}

func NewFunctionTool(name, description string, parameters map[string]any) *llmtypes.ToolDefinition {
	tool := &llmtypes.ToolDefinition{
		Name: name,
		Desc: description,
		Extra: map[string]any{
			toolJSONSchemaExtraKey: CloneJSONObject(parameters),
		},
	}
	if params := jsonSchemaParams(parameters); params != nil {
		tool.ParamsOneOf = llmtypes.NewToolParams(params)
	}
	return tool
}

func ToolJSONSchema(tool *llmtypes.ToolDefinition) map[string]any {
	if tool == nil || len(tool.Extra) == 0 {
		return nil
	}
	raw, ok := tool.Extra[toolJSONSchemaExtraKey].(map[string]any)
	if !ok {
		return nil
	}
	return CloneJSONObject(raw)
}

func CloneJSONObject(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var clone map[string]any
	if err := json.Unmarshal(raw, &clone); err != nil {
		return nil
	}
	return clone
}

func CloneToolDefinitions(toolDefs []*llmtypes.ToolDefinition) []*llmtypes.ToolDefinition {
	cloned := make([]*llmtypes.ToolDefinition, 0, len(toolDefs))
	for _, tool := range toolDefs {
		if tool == nil {
			continue
		}
		copyTool := *tool
		if tool.ParamsOneOf != nil {
			params, _ := tool.ParamsOneOf.ToJSONSchema()
			copyTool.Extra = CloneJSONObject(copyTool.Extra)
			if params != nil {
				if copyTool.Extra == nil {
					copyTool.Extra = map[string]any{}
				}
				if _, exists := copyTool.Extra[toolJSONSchemaExtraKey]; !exists {
					copyTool.Extra[toolJSONSchemaExtraKey] = params
				}
			}
			copyTool.ParamsOneOf = llmtypes.NewToolParams(tool.ParamsOneOf.Params)
		} else if len(tool.Extra) > 0 {
			copyTool.Extra = CloneJSONObject(tool.Extra)
		}
		cloned = append(cloned, &copyTool)
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func CloneToolDefinition(info *llmtypes.ToolDefinition) *llmtypes.ToolDefinition {
	cloned := CloneToolDefinitions([]*llmtypes.ToolDefinition{info})
	if len(cloned) == 0 {
		return nil
	}
	return cloned[0]
}

func ToolName(tool *llmtypes.ToolDefinition) string {
	if tool == nil {
		return ""
	}
	return tool.Name
}

func jsonSchemaParams(parameters map[string]any) map[string]*llmtypes.ToolParameter {
	if len(parameters) == 0 {
		return nil
	}
	props, _ := parameters["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	required := requiredFields(parameters["required"])
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[name] = true
	}
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]*llmtypes.ToolParameter, len(props))
	for _, key := range keys {
		raw, _ := props[key].(map[string]any)
		out[key] = jsonSchemaParamInfo(raw, requiredSet[key])
	}
	return out
}

func jsonSchemaParamInfo(raw map[string]any, required bool) *llmtypes.ToolParameter {
	if raw == nil {
		return &llmtypes.ToolParameter{Type: llmtypes.TypeString, Required: required}
	}
	info := &llmtypes.ToolParameter{
		Type:     jsonSchemaDataType(raw),
		Desc:     stringValue(raw["description"]),
		Enum:     enumStrings(raw["enum"]),
		Required: required,
	}
	if items, ok := raw["items"].(map[string]any); ok {
		info.ElemInfo = jsonSchemaParamInfo(items, false)
	}
	if props, ok := raw["properties"].(map[string]any); ok {
		childRequired := requiredFields(raw["required"])
		childRequiredSet := make(map[string]bool, len(childRequired))
		for _, name := range childRequired {
			childRequiredSet[name] = true
		}
		keys := make([]string, 0, len(props))
		for key := range props {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		info.SubParams = make(map[string]*llmtypes.ToolParameter, len(keys))
		for _, key := range keys {
			child, _ := props[key].(map[string]any)
			info.SubParams[key] = jsonSchemaParamInfo(child, childRequiredSet[key])
		}
	}
	return info
}

func jsonSchemaDataType(raw map[string]any) llmtypes.DataType {
	if raw == nil {
		return llmtypes.TypeString
	}
	if kind := strings.TrimSpace(stringValue(raw["type"])); kind != "" {
		return llmtypes.DataType(kind)
	}
	if _, ok := raw["properties"]; ok {
		return llmtypes.TypeObject
	}
	if _, ok := raw["items"]; ok {
		return llmtypes.TypeArray
	}
	return llmtypes.TypeString
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func requiredFields(raw any) []string {
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func enumStrings(raw any) []string {
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return append([]string(nil), v...)
	default:
		return nil
	}
}
