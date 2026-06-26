package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type goAdapter struct {
	c *mcpclient.Client
}

func NewStdioClient(ctx context.Context, command string, args []string, env map[string]string) (Client, error) {
	envSlice := os.Environ()
	for k, v := range env {
		envSlice = append(envSlice, k+"="+v)
	}

	c, err := mcpclient.NewStdioMCPClient(command, envSlice, args...)
	if err != nil {
		return nil, fmt.Errorf("create stdio MCP client: %w", err)
	}

	_, err = c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "PacketMind",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("MCP initialize: %w", err)
	}

	return &goAdapter{c: c}, nil
}

func (a *goAdapter) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	result, err := a.c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}

	defs := make([]ToolDefinition, 0, len(result.Tools))
	for _, t := range result.Tools {
		def := ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
		}

		schemaBytes, err := json.Marshal(t.InputSchema)
		if err == nil {
			var schema map[string]interface{}
			if json.Unmarshal(schemaBytes, &schema) == nil && len(schema) > 0 {
				def.InputSchema = schema
			}
		}

		defs = append(defs, def)
	}
	return defs, nil
}

func (a *goAdapter) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	result, err := a.c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	})
	if err != nil {
		return nil, err
	}

	blocks := make([]ContentBlock, 0, len(result.Content))
	for _, raw := range result.Content {
		if tc, ok := mcp.AsTextContent(raw); ok {
			blocks = append(blocks, ContentBlock{Type: "text", Text: tc.Text})
			continue
		}
		if ic, ok := mcp.AsImageContent(raw); ok {
			blocks = append(blocks, ContentBlock{Type: "image", Data: ic.Data, MimeType: ic.MIMEType})
		}
	}

	return &ToolResult{
		Content: blocks,
		IsError: result.IsError,
	}, nil
}
