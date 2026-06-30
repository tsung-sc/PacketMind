package runtime

import (
	"testing"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

func TestExtractObservationEventMetadataSuccessDoesNotSetErrorToolName(t *testing.T) {
	meta := extractObservationEventMetadata("get_request", `{"ok":true,"request":{"id":"req-1"}}`)
	if meta.ErrorToolName != "" {
		t.Fatalf("ErrorToolName = %q, want empty for successful observation", meta.ErrorToolName)
	}
	if meta.ErrorCategory != "" || meta.ErrorTimeout != "" || meta.ErrorRecovered {
		t.Fatalf("unexpected error metadata for success: %+v", meta)
	}
	if len(meta.RequestIDs) != 1 || meta.RequestIDs[0] != "req-1" {
		t.Fatalf("RequestIDs = %#v, want [req-1]", meta.RequestIDs)
	}
}

func TestExtractObservationEventMetadataErrorSetsFallbackToolName(t *testing.T) {
	meta := extractObservationEventMetadata("get_request", `{"ok":false,"error":"boom","category":"recoverable"}`)
	if meta.ErrorToolName != "get_request" {
		t.Fatalf("ErrorToolName = %q, want fallback tool name", meta.ErrorToolName)
	}
	if meta.ErrorCategory != "recoverable" {
		t.Fatalf("ErrorCategory = %q, want recoverable", meta.ErrorCategory)
	}
}

func TestEmitThoughtFromMessageIgnoresPlainContent(t *testing.T) {
	called := false
	emitThoughtFromMessage(func(event AgentEvent) {
		called = true
	}, 1, 0, &llmtypes.LLMMessage{Role: llmtypes.RoleAssistant, Content: "final answer"})
	if called {
		t.Fatal("emitThoughtFromMessage emitted thought for plain content")
	}
}
