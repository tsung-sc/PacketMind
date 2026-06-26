package llmtypes

import (
	"errors"
	"io"
	"testing"
)

func TestCollectStream_MergesChunkContentAndToolCalls(t *testing.T) {
	reader, writer := NewLLMStreamReader(4)
	go func() {
		defer writer.Close()
		writer.Send(&LLMMessage{
			Role:             RoleAssistant,
			Content:          "hel",
			ReasoningContent: "think-",
			ToolCalls: []ToolCall{{
				Index: 0,
				ID:    "call_1",
				Type:  "function",
				Function: FunctionCall{
					Name:      "sea",
					Arguments: `{"q":`,
				},
			}},
		})
		writer.Send(&LLMMessage{
			Role:             RoleAssistant,
			Content:          "lo",
			ReasoningContent: "plan",
			ToolCalls: []ToolCall{{
				Index: 0,
				Function: FunctionCall{
					Name:      "rch",
					Arguments: `"x"}`,
				},
			}},
			ResponseMeta: &ResponseMeta{
				FinishReason: "tool_calls",
				Usage: &TokenUsage{
					PromptTokens:     3,
					CompletionTokens: 2,
					TotalTokens:      5,
				},
			},
		})
	}()

	msg, err := CollectStream(reader)
	if err != nil {
		t.Fatalf("CollectStream() error = %v", err)
	}
	if msg == nil {
		t.Fatal("CollectStream() returned nil message")
	}
	if msg.Content != "hello" {
		t.Fatalf("content = %q, want %q", msg.Content, "hello")
	}
	if msg.ReasoningContent != "think-plan" {
		t.Fatalf("reasoning = %q, want %q", msg.ReasoningContent, "think-plan")
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("tool call count = %d, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Function.Name != "search" {
		t.Fatalf("tool name = %q, want %q", msg.ToolCalls[0].Function.Name, "search")
	}
	if msg.ToolCalls[0].Function.Arguments != `{"q":"x"}` {
		t.Fatalf("tool arguments = %q, want %q", msg.ToolCalls[0].Function.Arguments, `{"q":"x"}`)
	}
	if msg.ResponseMeta == nil || msg.ResponseMeta.FinishReason != "tool_calls" {
		t.Fatalf("unexpected response meta: %+v", msg.ResponseMeta)
	}
	if msg.ResponseMeta.Usage == nil || msg.ResponseMeta.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", msg.ResponseMeta)
	}
}

func TestCollectStream_PropagatesStreamError(t *testing.T) {
	reader, writer := NewLLMStreamReader(1)
	streamErr := errors.New("stream failed")
	go func() {
		defer writer.Close()
		writer.SendError(streamErr)
	}()

	_, err := CollectStream(reader)
	if !errors.Is(err, streamErr) {
		t.Fatalf("CollectStream() error = %v, want %v", err, streamErr)
	}
}

func TestCollectStream_EmptyReaderReturnsNilMessage(t *testing.T) {
	reader, writer := NewLLMStreamReader(1)
	writer.Close()

	msg, err := CollectStream(reader)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("CollectStream() unexpected error = %v", err)
	}
	if msg != nil {
		t.Fatalf("CollectStream() message = %+v, want nil", msg)
	}
}
