package tools

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
)

type ToolExecutor interface {
	Execute(ctx context.Context, name, arguments, defaultSessionID string) (*agentruntime.ToolExecutionResult, error)
	SafeExecute(ctx context.Context, name, arguments, defaultSessionID string) *agentruntime.SafeToolResult
	Schemas() []*llmtypes.ToolDefinition
}

type Executor struct {
	schemas map[string]*llmtypes.ToolDefinition
}

func NewExecutor(schemas []*llmtypes.ToolDefinition) *Executor {
	exec := &Executor{
		schemas: make(map[string]*llmtypes.ToolDefinition),
	}
	for _, schema := range schemas {
		exec.registerSchema(schema)
	}
	return exec
}

func (e *Executor) Schemas() []*llmtypes.ToolDefinition {
	keys := make([]string, 0, len(e.schemas))
	for name := range e.schemas {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	list := make([]*llmtypes.ToolDefinition, 0, len(keys))
	for _, name := range keys {
		list = append(list, CloneToolDefinition(e.schemas[name]))
	}
	return list
}

func (e *Executor) registerSchema(schema *llmtypes.ToolDefinition) {
	if e == nil || schema == nil {
		return
	}
	name := strings.TrimSpace(ToolName(schema))
	if name == "" {
		return
	}
	e.schemas[name] = CloneToolDefinition(schema)
}

func mustMarshalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `{"ok":false,"error":"failed to encode json"}`
	}
	return string(data)
}
