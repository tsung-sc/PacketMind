package agent

import (
	"context"
	"testing"

	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	agenttools "github.com/packetmind/packetmind/internal/agent/tools"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
	"github.com/packetmind/packetmind/internal/storage"
)

type scriptedLLMClient struct {
	messages []*llmtypes.LLMMessage
	index    int
}

func (c *scriptedLLMClient) Stream(ctx context.Context, input []*llmtypes.LLMMessage, opts ...llmtypes.LLMOption) (*llmtypes.LLMStreamReader, error) {
	sr, sw := llmtypes.NewLLMStreamReader(1)
	var msg *llmtypes.LLMMessage
	if c.index >= len(c.messages) {
		msg = &llmtypes.LLMMessage{Role: llmtypes.RoleAssistant}
	} else {
		msg = c.messages[c.index]
		c.index++
	}
	go func() {
		defer sw.Close()
		sw.Send(msg)
	}()
	return sr, nil
}

func newRuntimeTestAgent() *Agent {
	store, _ := storage.NewStorage("")
	storage.Default = store
	agent := NewAgent(&scriptedLLMClient{})
	agent.SetStore(store)
	return agent
}

func TestReactRuntime_UsesReasoningOnlyAssistantAsFinalAnswer(t *testing.T) {
	testAgent := newRuntimeTestAgent()
	runtime := agentruntime.NewRunner(&scriptedLLMClient{messages: []*llmtypes.LLMMessage{{Role: llmtypes.RoleAssistant, ReasoningContent: "Reasoning-only final answer"}}}, nil, agentruntime.WithExecuteTool(testAgent.mcpMgr.SafeExecute))
	result, err := runtime.Run(context.Background(), []*llmtypes.LLMMessage{{Role: llmtypes.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.FinalAnswer != "Reasoning-only final answer" {
		t.Fatalf("final answer = %q, want %q", result.FinalAnswer, "Reasoning-only final answer")
	}
	if result.StoppedEarly {
		t.Fatalf("expected normal completion, got stopped early with reason %q", result.StopReason)
	}
}

func TestReactRuntime_UsesAssistantMultiContentAsFinalAnswer(t *testing.T) {
	testAgent := newRuntimeTestAgent()
	runtime := agentruntime.NewRunner(&scriptedLLMClient{messages: []*llmtypes.LLMMessage{{Role: llmtypes.RoleAssistant, MultiContent: []llmtypes.ChatMessagePart{{Type: "text", Text: "Final answer from multi content"}}}}}, nil, agentruntime.WithExecuteTool(testAgent.mcpMgr.SafeExecute))
	result, err := runtime.Run(context.Background(), []*llmtypes.LLMMessage{{Role: llmtypes.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.FinalAnswer != "Final answer from multi content" {
		t.Fatalf("final answer = %q, want %q", result.FinalAnswer, "Final answer from multi content")
	}
	if result.StoppedEarly {
		t.Fatalf("expected normal completion, got stopped early with reason %q", result.StopReason)
	}
}

func TestReactRuntime_FallsBackToAssistantTextWhenExitResultEmpty(t *testing.T) {
	testAgent := newRuntimeTestAgent()
	runtime := agentruntime.NewRunner(&scriptedLLMClient{messages: []*llmtypes.LLMMessage{{
		Role:    llmtypes.RoleAssistant,
		Content: "Answer before exit tool",
		ToolCalls: []llmtypes.ToolCall{{
			ID:   "call-exit",
			Type: "function",
			Function: llmtypes.FunctionCall{
				Name:      "exit",
				Arguments: `{"final_result":""}`,
			},
		}},
	}}}, nil, agentruntime.WithExecuteTool(testAgent.mcpMgr.SafeExecute))
	result, err := runtime.Run(context.Background(), []*llmtypes.LLMMessage{{Role: llmtypes.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.FinalAnswer != "Answer before exit tool" {
		t.Fatalf("final answer = %q, want %q", result.FinalAnswer, "Answer before exit tool")
	}
	if result.StoppedEarly {
		t.Fatalf("expected normal completion, got stopped early with reason %q", result.StopReason)
	}
}

func TestReactRuntime_UsesExitToolArgumentsAsFinalAnswer(t *testing.T) {
	testAgent := newRuntimeTestAgent()
	runtime := agentruntime.NewRunner(&scriptedLLMClient{messages: []*llmtypes.LLMMessage{{
		Role: llmtypes.RoleAssistant,
		ToolCalls: []llmtypes.ToolCall{{
			ID:   "call-exit",
			Type: "function",
			Function: llmtypes.FunctionCall{
				Name:      "exit",
				Arguments: `{"final_result":"Answer from exit arguments"}`,
			},
		}},
	}}}, nil, agentruntime.WithExecuteTool(testAgent.mcpMgr.SafeExecute))
	result, err := runtime.Run(context.Background(), []*llmtypes.LLMMessage{{Role: llmtypes.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.FinalAnswer != "Answer from exit arguments" {
		t.Fatalf("final answer = %q, want %q", result.FinalAnswer, "Answer from exit arguments")
	}
	if result.StoppedEarly {
		t.Fatalf("expected normal completion, got stopped early with reason %q", result.StopReason)
	}
}

func TestReactRuntime_DoesNotOverwriteLastAssistantTextWithEmptyToolCallRound(t *testing.T) {
	testAgent := newRuntimeTestAgent()
	runtime := agentruntime.NewRunner(&scriptedLLMClient{messages: []*llmtypes.LLMMessage{{
		Role:             llmtypes.RoleAssistant,
		ReasoningContent: "Preserved answer from earlier round",
		ToolCalls: []llmtypes.ToolCall{{
			ID:   "call-search",
			Type: "function",
			Function: llmtypes.FunctionCall{
				Name:      "search_all_fields",
				Arguments: `{"query":"美国"}`,
			},
		}},
	}, {
		Role: llmtypes.RoleAssistant,
		ToolCalls: []llmtypes.ToolCall{{
			ID:   "call-exit",
			Type: "function",
			Function: llmtypes.FunctionCall{
				Name:      "exit",
				Arguments: `{"final_result":""}`,
			},
		}},
	}}}, agenttools.BuiltInSchemas(), agentruntime.WithExecuteTool(testAgent.mcpMgr.SafeExecute))
	result, err := runtime.Run(context.Background(), []*llmtypes.LLMMessage{{Role: llmtypes.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.FinalAnswer != "Preserved answer from earlier round" {
		t.Fatalf("final answer = %q, want %q", result.FinalAnswer, "Preserved answer from earlier round")
	}
	if result.StoppedEarly {
		t.Fatalf("expected normal completion, got stopped early with reason %q", result.StopReason)
	}
}
