package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/packetmind/packetmind/internal/agent/llmcore"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestClient_Constructors(t *testing.T) {
	t.Run("custom base URL with /v1 append", func(t *testing.T) {
		client := NewClient("key", "http://localhost:9000", Config{AppendV1ToBaseURL: true})
		if client.oc == nil {
			t.Fatalf("client should not be nil")
		}
	})

	t.Run("custom base URL without /v1 append", func(t *testing.T) {
		client := NewClient("key", "http://localhost:9000", Config{})
		if client.oc == nil {
			t.Fatalf("client should not be nil")
		}
	})
}

func TestClient_StreamCollect(t *testing.T) {
	t.Run("success with tool calls", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request failed: %v", err)
			}
			if req["model"] != "gpt-4o-mini" {
				t.Fatalf("unexpected model: %+v", req)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"id\":\"chat_1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-4o-mini\",\"choices\":[{\"index\":0,\"finish_reason\":\"\",\"delta\":{\"role\":\"assistant\",\"content\":\"hel\",\"tool_calls\":[{\"index\":0,\"id\":\"tool_1\",\"type\":\"function\",\"function\":{\"name\":\"x\",\"arguments\":\"{\"}}]}}]}\n\n")
			_, _ = io.WriteString(w, "data: {\"id\":\"chat_1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-4o-mini\",\"choices\":[{\"index\":0,\"finish_reason\":\"stop\",\"delta\":{\"role\":\"assistant\",\"content\":\"lo\",\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"}\"}}]}}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n")
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
		defer server.Close()

		client := NewClient("key", server.URL, Config{})
		reader, err := client.Stream(
			context.Background(),
			[]*llmtypes.LLMMessage{{Role: llmtypes.RoleSystem, Content: "s"}, {Role: llmtypes.RoleUser, Content: "u"}},
			llmtypes.WithModel("gpt-4o-mini"),
		)
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}
		resp, err := llmtypes.CollectStream(reader)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if resp.Content != "hello" || len(resp.ToolCalls) != 1 {
			t.Fatalf("unexpected response: %+v", resp)
		}
		if resp.ToolCalls[0].Function.Name != "x" || resp.ToolCalls[0].Function.Arguments != "{}" {
			t.Fatalf("unexpected tool call merge: %+v", resp.ToolCalls[0])
		}
	})

	t.Run("stream fails fast on non-retriable 403", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error":{"message":"forbidden","type":"invalid_request_error","code":"forbidden"}}`)
		}))
		defer server.Close()

		client := NewClient("key", server.URL, Config{})
		_, err := client.Stream(context.Background(), nil, llmtypes.WithModel("gpt-4o-mini"))
		if err == nil {
			t.Fatalf("expected stream error")
		}
		if attempts != 1 {
			t.Fatalf("expected 1 attempt, got %d", attempts)
		}
		ae := llmcore.AsAgentError(err)
		if ae == nil || !ae.IsFatal() {
			t.Fatalf("expected fatal AgentError, got %v", err)
		}
	})
}

func TestClient_StreamWithReasoning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"glm-4\",\"choices\":[{\"index\":0,\"finish_reason\":\"\",\"delta\":{\"role\":\"assistant\",\"content\":\"hel\",\"reasoning_content\":\"think\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"glm-4\",\"choices\":[{\"index\":0,\"finish_reason\":\"stop\",\"delta\":{\"role\":\"assistant\",\"content\":\"lo\"}}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewClient("k", server.URL, Config{ExtractReasoningInStream: true})
	sr, err := client.Stream(
		context.Background(),
		[]*llmtypes.LLMMessage{{Role: llmtypes.RoleSystem, Content: "sys"}, {Role: llmtypes.RoleUser, Content: "usr"}},
		llmtypes.WithModel("glm-4"),
	)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	defer sr.Close()

	var chunks []*llmtypes.LLMMessage
	for {
		msg, recvErr := sr.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			t.Fatalf("Recv() error = %v", recvErr)
		}
		chunks = append(chunks, msg)
	}
	if len(chunks) != 2 || chunks[0].Content != "hel" || chunks[0].ReasoningContent != "think" {
		t.Fatalf("unexpected chunks: %+v", chunks)
	}
	if chunks[1].ResponseMeta == nil || chunks[1].ResponseMeta.FinishReason != "stop" {
		t.Fatalf("unexpected finish metadata: %+v", chunks[1])
	}
	if chunks[1].ResponseMeta.Usage == nil || chunks[1].ResponseMeta.Usage.PromptTokens != 3 || chunks[1].ResponseMeta.Usage.CompletionTokens != 2 {
		t.Fatalf("unexpected usage chunk: %+v", chunks[1])
	}
}

func TestClient_StreamCollectWithToolCalls(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"id\":\"chat_1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"glm-4\",\"choices\":[{\"index\":0,\"finish_reason\":\"\",\"delta\":{\"role\":\"assistant\",\"content\":\"o\",\"tool_calls\":[{\"index\":0,\"id\":\"tool_1\",\"type\":\"function\",\"function\":{\"name\":\"x\",\"arguments\":\"{\"}}]}}]}\n\n")
			_, _ = io.WriteString(w, "data: {\"id\":\"chat_1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"glm-4\",\"choices\":[{\"index\":0,\"finish_reason\":\"stop\",\"delta\":{\"role\":\"assistant\",\"content\":\"k\",\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"}\"}}]}}]}\n\n")
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
		}))
		defer server.Close()

		client := NewClient("key", server.URL, Config{})
		reader, err := client.Stream(
			context.Background(),
			[]*llmtypes.LLMMessage{{Role: llmtypes.RoleSystem, Content: "s"}, {Role: llmtypes.RoleUser, Content: "u"}},
			llmtypes.WithModel("glm-4"),
		)
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}
		resp, err := llmtypes.CollectStream(reader)
		if err != nil {
			t.Fatalf("CollectStream() error = %v", err)
		}
		if resp.Content != "ok" || len(resp.ToolCalls) != 1 {
			t.Fatalf("unexpected result: resp=%+v err=%v", resp, err)
		}
	})

	t.Run("network error", func(t *testing.T) {
		client := NewClient("key", "http://example.invalid", Config{})
		client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("request failed")
		})}
		client.rebuildClient()
		_, err := client.Stream(context.Background(), nil, llmtypes.WithModel("glm-4"))
		if err == nil || !strings.Contains(err.Error(), "request failed") {
			t.Fatalf("expected request failure, got %v", err)
		}
	})
}

