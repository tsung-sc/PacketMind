package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	agenttools "github.com/packetmind/packetmind/internal/agent/tools"
	"github.com/packetmind/packetmind/internal/agent/tools/builtin"
	"github.com/packetmind/packetmind/internal/storage"
)

func NewBuiltinServer(store *storage.Storage, exec agenttools.ToolExecutor) *server.MCPServer {
	s := server.NewMCPServer("PacketMind Builtin", "1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(builtinGetRequestTool(), builtinWrappedHandler(store, builtin.NewGetRequestHandler(store)))
	s.AddTool(builtinSearchByHeaderTool(), builtinWrappedHandler(store, builtin.NewSearchByHeaderHandler(store)))
	s.AddTool(builtinSearchByBodyTool(), builtinWrappedHandler(store, builtin.NewSearchByBodyHandler(store)))
	s.AddTool(builtinSearchByResponseTool(), builtinWrappedHandler(store, builtin.NewSearchByResponseHandler(store)))
	s.AddTool(builtinAnalyzeEncodingTool(), builtinWrappedHandler(store, builtin.NewAnalyzeEncodingHandler()))
	s.AddTool(builtinSearchAllFieldsTool(), builtinWrappedHandler(store, builtin.NewSearchAllFieldsHandler(store)))
	s.AddTool(builtinFindPriorResponseSourcesTool(), builtinWrappedHandler(store, builtin.NewFindPriorResponseSourcesHandler(store)))
	s.AddTool(builtinFindLaterRequestUsagesTool(), builtinWrappedHandler(store, builtin.NewFindLaterRequestUsagesHandler(store)))
	s.AddTool(builtinTraceValueFlowTool(), builtinWrappedHandler(store, builtin.NewTraceValueFlowHandler(store)))
	s.AddTool(builtinDiffRequestsTool(), builtinWrappedHandler(store, builtin.NewDiffRequestsHandler(store)))
	s.AddTool(builtinBatchExecuteTool(), builtinBatchHandler(exec))
	s.AddTool(builtinBashTool(), builtinWrappedHandler(store, builtin.NewBashHandler()))
	s.AddTool(builtinSummarizeSessionTool(), builtinWrappedHandler(store, builtin.NewSummarizeSessionHandler(store)))
	s.AddTool(builtinClassifyRequestsTool(), builtinWrappedHandler(store, builtin.NewClassifyRequestsHandler(store)))
	s.AddTool(builtinTraceFlowSequenceTool(), builtinWrappedHandler(store, builtin.NewTraceFlowSequenceHandler(store)))
	s.AddTool(builtinTestHypothesisTool(), builtinWrappedHandler(store, builtin.NewTestHypothesisHandler(store)))
	s.AddTool(builtinListSessionsTool(), builtinWrappedHandler(store, builtin.NewListSessionsHandler(store)))

	return s
}

func NewInProcessClient(ctx context.Context, s *server.MCPServer) (Client, error) {
	c, err := mcpclient.NewInProcessClient(s)
	if err != nil {
		return nil, fmt.Errorf("create in-process MCP client: %w", err)
	}

	if _, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "PacketMind",
				Version: "1.0.0",
			},
		},
	}); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("MCP initialize: %w", err)
	}

	return &goAdapter{c: c}, nil
}

func builtinEncodeHandlerResult(result *agentruntime.ToolExecutionResult) string {
	encoded := map[string]interface{}{
		"content":     result.Content,
		"summary":     result.Summary,
		"request_ids": result.RequestIDs,
	}
	data, _ := json.Marshal(encoded)
	return string(data)
}

func builtinWrappedHandler(store *storage.Storage, handler builtin.BuiltinHandlerFunc) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_ = store
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		if args == nil {
			args = map[string]any{}
		}
		sessionID, _ := args["session_id"].(string)
		result, err := handler(args, sessionID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(builtinEncodeHandlerResult(result)), nil
	}
}

func builtinBatchHandler(exec agenttools.ToolExecutor) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	handler := builtin.NewBatchExecuteHandler(exec)
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		if args == nil {
			args = map[string]any{}
		}
		sessionID, _ := args["session_id"].(string)
		result, err := handler(args, sessionID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(builtinEncodeHandlerResult(result)), nil
	}
}

func builtinGetRequestTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolGetRequest,
		mcp.WithDescription("Fetch the complete details of a specific HTTP request by its ID."),
		mcp.WithString("request_id", mcp.Required(), mcp.Description("The request ID to fetch details for")),
		mcp.WithBoolean("include_full_body", mcp.Description("When true, returns complete request/response body text without truncation")),
	)
}

func builtinSearchByHeaderTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolSearchRequestsByHeader,
		mcp.WithDescription("Search for requests containing a specific header name and value pattern."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("header_name", mcp.Required(), mcp.Description("Request header name to match (case-insensitive)")),
		mcp.WithString("header_value", mcp.Required(), mcp.Description("Request header value fragment to match")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (1-50)")),
	)
}

func builtinSearchByBodyTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolSearchRequestsByBody,
		mcp.WithDescription("Search for requests whose request body contains a given text fragment."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Content fragment to search in request bodies")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (1-50)")),
	)
}

func builtinSearchByResponseTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolSearchRequestsByResponse,
		mcp.WithDescription("Search for requests whose response body contains a given text fragment."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Content fragment to search in response bodies")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (1-50)")),
	)
}

func builtinAnalyzeEncodingTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolAnalyzeEncoding,
		mcp.WithDescription("Detect and decode nested encoding layers in a value (Base64, URL-encoding, HTML entities, hex, etc.)."),
		mcp.WithString("value", mcp.Required(), mcp.Description("Input value to inspect")),
	)
}

func builtinSearchAllFieldsTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolSearchAllFields,
		mcp.WithDescription("Search across ALL request fields simultaneously for a given value."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Content fragment to search across all fields")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of matching requests to return (1-50)")),
	)
}

func builtinFindPriorResponseSourcesTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolFindPriorResponseSources,
		mcp.WithDescription("Trace backward in time: find earlier responses whose response data contains a target value."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("before_request_id", mcp.Required(), mcp.Description("Target request ID; only searches responses before this request")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Target value to locate in prior responses")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (1-50)")),
	)
}

func builtinFindLaterRequestUsagesTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolFindLaterRequestUsages,
		mcp.WithDescription("Trace forward in time: find later requests that reuse a target value."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("after_request_id", mcp.Required(), mcp.Description("Source request ID; only searches requests after this one")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Target value to locate in later requests")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (1-50)")),
	)
}

func builtinTraceValueFlowTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolTraceValueFlow,
		mcp.WithDescription("End-to-end provenance tracing with automatic confidence scoring."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("request_id", mcp.Required(), mcp.Description("Target request ID")),
		mcp.WithString("field_name", mcp.Required(), mcp.Description("Target field name")),
		mcp.WithString("location", mcp.Required(), mcp.Description("Where the target field is located: query, header, cookie, form_body, json_body")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of candidate provenance links (1-50)")),
	)
}

func builtinDiffRequestsTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolDiffRequests,
		mcp.WithDescription("Compare two captured requests field-by-field to identify differences."),
		mcp.WithString("request_id_a", mcp.Required(), mcp.Description("First request ID to compare")),
		mcp.WithString("request_id_b", mcp.Required(), mcp.Description("Second request ID to compare")),
	)
}

func builtinBatchExecuteTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolBatchExecute,
		mcp.WithDescription("Execute multiple tool calls in parallel and aggregate their results."),
		mcp.WithString("session_id", mcp.Description("Optional session ID fallback for nested calls")),
		mcp.WithString("calls", mcp.Required(), mcp.Description("JSON array of tool call objects: [{\"tool\":\"name\",\"args\":\"{...}\"}]")),
	)
}

func builtinBashTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolBash,
		mcp.WithDescription("Execute a shell command in a sandboxed workspace directory and return the output."),
		mcp.WithString("command", mcp.Required(), mcp.Description("Shell command to execute")),
		mcp.WithString("workdir", mcp.Description("Working directory for the command (default: data/agent-workspace)")),
		mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (1-300, default 60)")),
	)
}

func builtinSummarizeSessionTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolSummarizeSession,
		mcp.WithDescription("Get a high-level overview of a capture session. Returns host breakdowns, request counts, status distributions, content-type distributions, auth detection, and a timeline overview. Use this as the FIRST step in any analysis."),
		mcp.WithString("session_id", mcp.Description("Optional session ID")),
	)
}

func builtinClassifyRequestsTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolClassifyRequests,
		mcp.WithDescription("Automatically classify requests in a session by their likely role: auth_entry, token_issuance, signed_request, data_query, auth_request, config_fetch, static_resource, redirect, error, other. Returns a categorized list with key findings."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("host", mcp.Description("Optional host filter")),
		mcp.WithNumber("limit", mcp.Description("Max results per category (1-50, default 10)")),
	)
}

func builtinTraceFlowSequenceTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolTraceFlowSequence,
		mcp.WithDescription("List requests in chronological order and detect field flow relationships — cookie set/use, token issuance/consumption, redirect chains, value reuse. Annotates flow edges showing how data moves through the session."),
		mcp.WithString("session_id", mcp.Description("Optional session ID filter")),
		mcp.WithString("host", mcp.Description("Optional host filter")),
		mcp.WithArray("focus_fields", mcp.Description("Optional field names to track")),
		mcp.WithNumber("max_requests", mcp.Description("Max requests (10-100, default 50)")),
	)
}

func builtinTestHypothesisTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolTestHypothesis,
		mcp.WithDescription("Test a hypothesis about how a field value is generated. Given request IDs and a hypothesis composed of operations (field extraction, concatenation, hash, HMAC, base64, etc.), this tool tests against actual values and returns match rate."),
		mcp.WithString("request_ids", mcp.Required(), mcp.Description("Request IDs to test (minimum 3)")),
		mcp.WithString("target_field", mcp.Required(), mcp.Description("Field to verify, e.g. 'sign', 'Authorization', 'token'")),
		mcp.WithString("target_location", mcp.Required(), mcp.Description("'body', 'header', 'query', 'response_body', 'response_header', 'cookie'")),
		mcp.WithString("hypothesis", mcp.Required(), mcp.Description("Structured generation rule. E.g. 'MD5(CONCAT(timestamp, body_raw))', 'HMAC_SHA256(CONCAT(method,path,timestamp), secret=EXTRACT(response_body,$.data.secret))'")),
	)
}

func builtinListSessionsTool() mcp.Tool {
	return mcp.NewTool(agenttools.ToolListSessions,
		mcp.WithDescription("List all capture sessions with their IDs, names, and active status."),
	)
}
