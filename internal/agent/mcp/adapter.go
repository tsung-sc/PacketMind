package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
	agenttools "github.com/packetmind/packetmind/internal/agent/tools"
)

type ToolAdapter struct {
	client     Client
	prefix     string
	mu         sync.RWMutex
	tools      []ToolDefinition
	discovered bool
}

func NewToolAdapter(client Client, prefix string) *ToolAdapter {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	prefix = strings.TrimSuffix(prefix, "_")
	return &ToolAdapter{client: client, prefix: prefix}
}

func (a *ToolAdapter) Prefix() string {
	if a == nil {
		return ""
	}
	return a.prefix
}

func (a *ToolAdapter) DiscoverTools(ctx context.Context) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("adapter or client is nil")
	}

	tools, err := a.client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list MCP tools: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = tools
	a.discovered = true
	return nil
}

func (a *ToolAdapter) GetToolDefinitions() ([]ToolDefinition, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.discovered {
		return nil, fmt.Errorf("tools not discovered; call DiscoverTools first")
	}
	result := make([]ToolDefinition, len(a.tools))
	copy(result, a.tools)
	return result, nil
}

func (a *ToolAdapter) PrefixedName(originalName string) string {
	if a == nil || a.prefix == "" {
		return originalName
	}
	return a.prefix + "_" + originalName
}

func (a *ToolAdapter) ToTool(def ToolDefinition) *llmtypes.ToolDefinition {
	return agenttools.NewFunctionTool(a.PrefixedName(def.Name), def.Description, def.InputSchema)
}

func (a *ToolAdapter) RegisterTools(ctx context.Context) ([]*llmtypes.ToolDefinition, error) {
	if a == nil {
		return nil, fmt.Errorf("adapter is nil")
	}
	if !a.discovered {
		if err := a.DiscoverTools(ctx); err != nil {
			return nil, err
		}
	}

	a.mu.RLock()
	defs := append([]ToolDefinition(nil), a.tools...)
	a.mu.RUnlock()

	registered := make([]*llmtypes.ToolDefinition, 0, len(defs))
	for _, def := range defs {
		registered = append(registered, a.ToTool(def))
	}
	return registered, nil
}

func (a *ToolAdapter) ExecuteTool(ctx context.Context, originalName string, arguments map[string]interface{}) (*ToolResult, error) {
	if a == nil || a.client == nil {
		return nil, fmt.Errorf("adapter or client is nil")
	}
	return a.client.CallTool(ctx, originalName, arguments)
}

func (a *ToolAdapter) ExecuteToolByPrefixedName(ctx context.Context, prefixedName string, arguments map[string]interface{}) (*ToolResult, error) {
	originalName := a.stripPrefix(prefixedName)
	if originalName == "" {
		return nil, fmt.Errorf("invalid tool name: %s", prefixedName)
	}
	return a.ExecuteTool(ctx, originalName, arguments)
}

func (a *ToolAdapter) stripPrefix(prefixedName string) string {
	if a.prefix == "" {
		return prefixedName
	}
	expectedPrefix := a.prefix + "_"
	if !strings.HasPrefix(prefixedName, expectedPrefix) {
		return ""
	}
	return strings.TrimPrefix(prefixedName, expectedPrefix)
}
