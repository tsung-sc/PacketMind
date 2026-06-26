package llmcore

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestAgentError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AgentError
		contains string
	}{
		{
			name:     "fatal error with cause",
			err:      NewFatalError("something failed", errors.New("underlying")),
			contains: "[fatal]",
		},
		{
			name:     "retryable error without cause",
			err:      NewRetryableError("rate limited", nil),
			contains: "[retryable]",
		},
		{
			name:     "recoverable error",
			err:      NewRecoverableError("fallback needed", nil),
			contains: "[recoverable]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if !containsStr(msg, tt.contains) {
				t.Errorf("Error() = %q, want to contain %q", msg, tt.contains)
			}
		})
	}
}

func TestAgentError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewFatalError("wrapped", underlying)

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}
}

func TestAgentError_Classification(t *testing.T) {
	t.Run("IsRetryable", func(t *testing.T) {
		if !NewRetryableError("", nil).IsRetryable() {
			t.Error("retryable error should be retryable")
		}
		if NewFatalError("", nil).IsRetryable() {
			t.Error("fatal error should not be retryable")
		}
	})

	t.Run("IsRecoverable", func(t *testing.T) {
		if !NewRecoverableError("", nil).IsRecoverable() {
			t.Error("recoverable error should be recoverable")
		}
		if !NewRetryableError("", nil).IsRecoverable() {
			t.Error("retryable error should be recoverable")
		}
		if NewFatalError("", nil).IsRecoverable() {
			t.Error("fatal error should not be recoverable")
		}
	})

	t.Run("IsFatal", func(t *testing.T) {
		if !NewFatalError("", nil).IsFatal() {
			t.Error("fatal error should be fatal")
		}
		if NewRetryableError("", nil).IsFatal() {
			t.Error("retryable error should not be fatal")
		}
	})
}

func TestAgentError_Timeout(t *testing.T) {
	err := NewToolTimeoutError("test_tool", 30*time.Second)

	if !err.IsTimeout() {
		t.Error("timeout error should report IsTimeout=true")
	}
	if err.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", err.Timeout)
	}
	if err.ToolName != "test_tool" {
		t.Errorf("ToolName = %q, want %q", err.ToolName, "test_tool")
	}
}

func TestAgentError_Panic(t *testing.T) {
	err := NewToolPanicError("panic_tool", "something went wrong")

	if !err.IsPanic() {
		t.Error("panic error should report IsPanic=true")
	}
	if !err.Recovered {
		t.Error("panic error should have Recovered=true")
	}
	if err.ToolName != "panic_tool" {
		t.Errorf("ToolName = %q, want %q", err.ToolName, "panic_tool")
	}
}

func TestNewToolError(t *testing.T) {
	underlying := errors.New("tool failed")
	err := NewToolError("get_request", "failed to fetch", underlying)

	if err.Category != CategoryRecoverable {
		t.Errorf("Category = %q, want %q", err.Category, CategoryRecoverable)
	}
	if err.ToolName != "get_request" {
		t.Errorf("ToolName = %q, want %q", err.ToolName, "get_request")
	}
}

func TestAsAgentError(t *testing.T) {
	t.Run("returns AgentError for AgentError", func(t *testing.T) {
		original := NewFatalError("test", nil)
		result := AsAgentError(original)
		if result != original {
			t.Error("should return same AgentError")
		}
	})

	t.Run("returns nil for non-AgentError", func(t *testing.T) {
		result := AsAgentError(errors.New("standard error"))
		if result != nil {
			t.Error("should return nil for non-AgentError")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		result := AsAgentError(nil)
		if result != nil {
			t.Error("should return nil for nil error")
		}
	})

	t.Run("extracts wrapped AgentError", func(t *testing.T) {
		original := NewRetryableError("transient", errors.New("429"))
		wrapped := fmt.Errorf("outer: %w", original)
		result := AsAgentError(wrapped)
		if result != original {
			t.Fatalf("should extract wrapped AgentError, got %v", result)
		}
	})
}

func TestWrapError(t *testing.T) {
	t.Run("wraps standard error", func(t *testing.T) {
		original := errors.New("standard error")
		wrapped := WrapError(original, CategoryFatal, "wrapped message")

		if wrapped.Category != CategoryFatal {
			t.Errorf("Category = %q, want %q", wrapped.Category, CategoryFatal)
		}
		if wrapped.Message != "wrapped message" {
			t.Errorf("Message = %q, want %q", wrapped.Message, "wrapped message")
		}
		if wrapped.Cause != original {
			t.Error("Cause should be original error")
		}
	})

	t.Run("returns unchanged AgentError", func(t *testing.T) {
		original := NewFatalError("original", nil)
		wrapped := WrapError(original, CategoryRetryable, "new message")

		if wrapped != original {
			t.Error("should return original AgentError unchanged")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		wrapped := WrapError(nil, CategoryFatal, "message")
		if wrapped != nil {
			t.Error("should return nil for nil error")
		}
	})
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStrHelper(s, substr))
}

func containsStrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
