package tools

import agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"

type BuiltinHandler func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error)