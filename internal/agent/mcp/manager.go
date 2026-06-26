package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/packetmind/packetmind/internal/agent/llmcore"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	agenttools "github.com/packetmind/packetmind/internal/agent/tools"
)

const defaultToolTimeout = 30 * time.Second

type Manager struct {
	mu       sync.RWMutex
	adapters map[string]*ToolAdapter
	schemas  []*llmtypes.ToolDefinition
}

func NewManager() *Manager {
	return &Manager{adapters: make(map[string]*ToolAdapter), schemas: make([]*llmtypes.ToolDefinition, 0)}
}

func (m *Manager) Schemas() []*llmtypes.ToolDefinition {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.schemas) == 0 {
		return make([]*llmtypes.ToolDefinition, 0)
	}
	return agenttools.CloneToolDefinitions(m.schemas)
}

func (m *Manager) AddAdapter(name string, client Client) error {
	return m.AddAdapterWithPrefix(name, name, client)
}

func (m *Manager) AddAdapterWithPrefix(name, prefix string, client Client) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("adapter name cannot be empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.adapters[name]; exists {
		return fmt.Errorf("adapter %q already exists", name)
	}
	m.adapters[name] = NewToolAdapter(client, prefix)
	return nil
}

func (m *Manager) RemoveAdapter(name string) bool {
	if m == nil {
		return false
	}
	name = strings.ToLower(strings.TrimSpace(name))
	m.mu.Lock()
	defer m.mu.Unlock()
	adapter, exists := m.adapters[name]
	if !exists {
		return false
	}
	delete(m.adapters, name)
	if adapter.prefix != "" {
		prefix := adapter.prefix + "_"
		filtered := make([]*llmtypes.ToolDefinition, 0, len(m.schemas))
		for _, tool := range m.schemas {
			if tool == nil || !strings.HasPrefix(agenttools.ToolName(tool), prefix) {
				filtered = append(filtered, tool)
			}
		}
		m.schemas = filtered
	}
	return true
}

func (m *Manager) GetAdapter(name string) *ToolAdapter {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.adapters[name]
}

func (m *Manager) DiscoverAll(ctx context.Context) map[string]error {
	m.mu.RLock()
	adapters := make(map[string]*ToolAdapter, len(m.adapters))
	for k, v := range m.adapters {
		adapters[k] = v
	}
	m.mu.RUnlock()

	errors := make(map[string]error)
	for name, adapter := range adapters {
		if err := adapter.DiscoverTools(ctx); err != nil {
			errors[name] = err
		}
	}
	return errors
}

func (m *Manager) RegisterAll(ctx context.Context) (int, map[string]error) {
	m.mu.RLock()
	adapters := make(map[string]*ToolAdapter, len(m.adapters))
	for k, v := range m.adapters {
		adapters[k] = v
	}
	m.mu.RUnlock()

	totalRegistered := 0
	errors := make(map[string]error)
	collected := make([]*llmtypes.ToolDefinition, 0)
	for name, adapter := range adapters {
		defs, err := adapter.RegisterTools(ctx)
		if err != nil {
			errors[name] = err
			continue
		}
		totalRegistered += len(defs)
		collected = append(collected, defs...)
	}
	m.mu.Lock()
	m.schemas = agenttools.CloneToolDefinitions(collected)
	m.mu.Unlock()
	return totalRegistered, errors
}

func (m *Manager) executeTool(ctx context.Context, prefixedName string, arguments map[string]interface{}) (*ToolResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, adapter := range m.adapters {
		if adapter.prefix == "" {
			continue
		}
		prefix := adapter.prefix + "_"
		if strings.HasPrefix(prefixedName, prefix) {
			return adapter.ExecuteToolByPrefixedName(ctx, prefixedName, arguments)
		}
	}

	for _, adapter := range m.adapters {
		if adapter.prefix == "" {
			result, err := adapter.ExecuteTool(ctx, prefixedName, arguments)
			if err == nil {
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("no adapter found for tool %q", prefixedName)
}

func (m *Manager) Execute(ctx context.Context, name, arguments, defaultSessionID string) (*agentruntime.ToolExecutionResult, error) {
	args, err := agenttools.ParseToolArguments(arguments)
	if err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}

	sessionID := agenttools.GetStringArg(args, "session_id", defaultSessionID)
	if sessionID != "" {
		if _, has := args["session_id"]; !has {
			args["session_id"] = sessionID
		}
	}

	modifiedArgs := arguments
	if len(args) > 0 {
		if data, err := json.Marshal(args); err == nil {
			modifiedArgs = string(data)
		}
	}

	if tool, found := m.schemaMap()[strings.TrimSpace(name)]; found {
		if verr := agenttools.ValidateToolParams(name, args, agenttools.ToolJSONSchema(tool)); verr != nil {
			return &agentruntime.ToolExecutionResult{
				Content: mustMarshalJSON(map[string]any{
					"ok":    false,
					"error": verr.Error(),
					"hint":  "Please check the parameter types and required fields",
				}),
				Summary: fmt.Sprintf("Parameter validation failed for %s", name),
			}, nil
		}
	}

	content, err := m.executeToolWithJSON(ctx, name, modifiedArgs)
	if err == nil {
		return buildToolResult(content, name), nil
	}

	return nil, fmt.Errorf("unknown tool: %s", name)
}

func (m *Manager) SafeExecute(ctx context.Context, name, arguments, defaultSessionID string) *agentruntime.SafeToolResult {
	result := &agentruntime.SafeToolResult{}

	timeout := defaultToolTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				result.Err = llmcore.NewToolPanicError(name, r)
			}
		}()

		execResult, err := m.Execute(ctx, name, arguments, defaultSessionID)
		if err != nil {
			result.Err = llmcore.NewToolError(name, err.Error(), err)
			return
		}
		result.Result = execResult
	}()

	select {
	case <-done:
		return result
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			result.Err = llmcore.NewToolTimeoutError(name, timeout)
		} else {
			result.Err = llmcore.NewToolError(name, ctx.Err().Error(), ctx.Err())
		}
		return result
	}
}

func (m *Manager) ListAdapters() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.adapters))
	for name := range m.adapters {
		names = append(names, name)
	}
	return names
}

func (m *Manager) AdapterCount() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.adapters)
}

func (m *Manager) MergeFrom(other *Manager) {
	if m == nil || other == nil {
		return
	}
	other.mu.RLock()
	defer other.mu.RUnlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, adapter := range other.adapters {
		if _, exists := m.adapters[name]; !exists {
			m.adapters[name] = adapter
		}
	}
}

func (m *Manager) executeToolWithJSON(ctx context.Context, prefixedName, argumentsJSON string) (string, error) {
	var args map[string]interface{}
	if strings.TrimSpace(argumentsJSON) != "" && argumentsJSON != "{}" {
		if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
			return "", fmt.Errorf("failed to parse arguments: %w", err)
		}
	} else {
		args = make(map[string]interface{})
	}

	result, err := m.executeTool(ctx, prefixedName, args)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("tool execution error: %s", result.TextContent())
	}
	return result.TextContent(), nil
}

func (m *Manager) schemaMap() map[string]*llmtypes.ToolDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	schemas := make(map[string]*llmtypes.ToolDefinition, len(m.schemas))
	for _, s := range m.schemas {
		if s != nil {
			schemas[agenttools.ToolName(s)] = s
		}
	}
	return schemas
}

func buildToolResult(content, name string) *agentruntime.ToolExecutionResult {
	var structured struct {
		Content    string   `json:"content"`
		Summary    string   `json:"summary"`
		RequestIDs []string `json:"request_ids"`
	}
	if err := json.Unmarshal([]byte(content), &structured); err == nil && structured.Content != "" {
		return &agentruntime.ToolExecutionResult{
			Content:    structured.Content,
			Summary:    structured.Summary,
			RequestIDs: structured.RequestIDs,
		}
	}
	return &agentruntime.ToolExecutionResult{
		Content: content,
		Summary: fmt.Sprintf("Executed tool %s", name),
	}
}

func mustMarshalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `{"ok":false,"error":"failed to encode json"}`
	}
	return string(data)
}
