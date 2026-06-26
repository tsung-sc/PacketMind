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
