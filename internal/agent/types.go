package agent

import (
	"time"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
)

type AgentRequest struct {
	RequestID            string                             `json:"request_id"`
	SessionID            string                             `json:"session_id,omitempty"`
	Query                string                             `json:"query"`
	Model                string                             `json:"model"`
	InterventionProvider func() []agentruntime.Intervention `json:"-"`
}

type AgentTraceStep struct {
	Depth         int       `json:"depth"`
	Type          string    `json:"type"`
	Content       string    `json:"content,omitempty"`
	ToolName      string    `json:"tool_name,omitempty"`
	ToolArguments string    `json:"tool_arguments,omitempty"`
	ToolResult    string    `json:"tool_result,omitempty"`
	RequestIDs    []string  `json:"request_ids,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AgentResult struct {
	RequestID    string               `json:"request_id"`
	SessionID    string               `json:"session_id,omitempty"`
	Model        string               `json:"model"`
	FinalAnswer  string               `json:"final_answer"`
	TraceChain   []AgentTraceStep     `json:"trace_chain"`
	DepthUsed    int                  `json:"depth_used"`
	ToolCalls    int                  `json:"tool_calls"`
	StoppedEarly bool                 `json:"stopped_early"`
	StopReason   string               `json:"stop_reason,omitempty"`
	TokenUsage   *llmtypes.TokenUsage `json:"token_usage,omitempty"`
}

type AgentEvent = agentruntime.AgentEvent
type AgentEventHandler = agentruntime.AgentEventHandler
