package mcp

import (
	"context"
	"strings"
)

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func (r *ToolResult) TextContent() string {
	if r == nil || len(r.Content) == 0 {
		return ""
	}
	var texts []string
	for _, block := range r.Content {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n")
}

type Client interface {
	ListTools(ctx context.Context) ([]ToolDefinition, error)
	CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error)
}
