package llmcore

import (
	"errors"
	"fmt"
	"time"
)

type ErrorCategory string

const (
	CategoryRetryable   ErrorCategory = "retryable"
	CategoryRecoverable ErrorCategory = "recoverable"
	CategoryFatal       ErrorCategory = "fatal"
)

type AgentError struct {
	Category  ErrorCategory `json:"category"`
	Message   string        `json:"message"`
	Cause     error         `json:"cause,omitempty"`
	ToolName  string        `json:"tool_name,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	Recovered bool          `json:"recovered,omitempty"`
}

func (e *AgentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Category, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

func (e *AgentError) Unwrap() error     { return e.Cause }
func (e *AgentError) IsRetryable() bool { return e.Category == CategoryRetryable }

func (e *AgentError) IsRecoverable() bool {
	return e.Category == CategoryRecoverable || e.Category == CategoryRetryable
}

func (e *AgentError) IsFatal() bool   { return e.Category == CategoryFatal }
func (e *AgentError) IsTimeout() bool { return e.Timeout > 0 }
func (e *AgentError) IsPanic() bool   { return e.Recovered }

func NewFatalError(message string, cause error) *AgentError {
	return &AgentError{Category: CategoryFatal, Message: message, Cause: cause}
}

func NewRetryableError(message string, cause error) *AgentError {
	return &AgentError{Category: CategoryRetryable, Message: message, Cause: cause}
}

func NewRecoverableError(message string, cause error) *AgentError {
	return &AgentError{Category: CategoryRecoverable, Message: message, Cause: cause}
}

func NewToolError(toolName, message string, cause error) *AgentError {
	return &AgentError{Category: CategoryRecoverable, Message: message, Cause: cause, ToolName: toolName}
}

func NewToolTimeoutError(toolName string, timeout time.Duration) *AgentError {
	return &AgentError{Category: CategoryRecoverable, Message: fmt.Sprintf("tool %q execution timed out after %v", toolName, timeout), ToolName: toolName, Timeout: timeout}
}

func NewToolPanicError(toolName string, recovered interface{}) *AgentError {
	return &AgentError{Category: CategoryRecoverable, Message: fmt.Sprintf("tool %q panicked", toolName), Cause: fmt.Errorf("panic: %v", recovered), ToolName: toolName, Recovered: true}
}

func AsAgentError(err error) *AgentError {
	if err == nil {
		return nil
	}
	var ae *AgentError
	if errors.As(err, &ae) {
		return ae
	}
	return nil
}

func WrapError(err error, category ErrorCategory, message string) *AgentError {
	if err == nil {
		return nil
	}
	if ae := AsAgentError(err); ae != nil {
		return ae
	}
	return &AgentError{Category: category, Message: message, Cause: err}
}
