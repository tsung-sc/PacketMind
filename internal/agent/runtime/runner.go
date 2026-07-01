package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/packetmind/packetmind/internal/agent/llmcore"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

type ExecuteToolFunc func(ctx context.Context, name, arguments, sessionID string) *SafeToolResult

type Intervention struct {
	ID      string
	Content string
}

type InterventionFunc func() []Intervention

type AgentEvent struct {
	Depth          int       `json:"depth"`
	Type           string    `json:"type"`
	Content        string    `json:"content,omitempty"`
	ToolName       string    `json:"tool_name,omitempty"`
	ToolCallID     string    `json:"tool_call_id,omitempty"`
	Arguments      string    `json:"arguments,omitempty"`
	Result         string    `json:"result,omitempty"`
	RequestIDs     []string  `json:"request_ids,omitempty"`
	ToolCalls      int       `json:"tool_calls"`
	CreatedAt      time.Time `json:"created_at"`
	SpecialistName string    `json:"specialist_name,omitempty"`

	ErrorCategory  string `json:"error_category,omitempty"`
	ErrorToolName  string `json:"error_tool_name,omitempty"`
	ErrorTimeout   string `json:"error_timeout,omitempty"`
	ErrorRecovered bool   `json:"error_recovered,omitempty"`

	RetryAttempt int `json:"retry_attempt,omitempty"`
	RetryMax     int `json:"retry_max,omitempty"`

	InterventionID string `json:"intervention_id,omitempty"`
}

type AgentEventHandler func(event AgentEvent)

type RuntimeResult struct {
	FinalAnswer  string
	Messages     []*llmtypes.LLMMessage
	Iterations   int
	ToolCalls    int
	StoppedEarly bool
	StopReason   string
	TokenUsage   *llmtypes.TokenUsage
}

type ToolExecutionResult struct {
	Content    string
	Summary    string
	RequestIDs []string
}

type SafeToolResult struct {
	Result *ToolExecutionResult
	Err    *llmcore.AgentError
}

type Runner struct {
	client       llmtypes.LLMClient
	tools        []*llmtypes.ToolDefinition
	systemPrompt string
	model        string
	sessionID    string
	intervention InterventionFunc
	executeTool  ExecuteToolFunc
}

type RunnerOption func(*Runner)

func NewRunner(client llmtypes.LLMClient, tools []*llmtypes.ToolDefinition, opts ...RunnerOption) *Runner {
	r := &Runner{
		client: client,
		tools:  cloneToolDefinitions(tools),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

func WithModel(model string) RunnerOption {
	return func(r *Runner) {
		if r != nil {
			r.model = strings.TrimSpace(model)
		}
	}
}

func WithSessionID(sessionID string) RunnerOption {
	return func(r *Runner) {
		if r != nil {
			r.sessionID = strings.TrimSpace(sessionID)
		}
	}
}

func WithExecuteTool(fn ExecuteToolFunc) RunnerOption {
	return func(r *Runner) {
		if r != nil {
			r.executeTool = fn
		}
	}
}

func WithIntervention(fn InterventionFunc) RunnerOption {
	return func(r *Runner) {
		if r != nil {
			r.intervention = fn
		}
	}
}

func WithSystemPrompt(prompt string) RunnerOption {
	return func(r *Runner) {
		if r != nil {
			r.systemPrompt = strings.TrimSpace(prompt)
		}
	}
}

func (r *Runner) SetSystemPrompt(prompt string) {
	r.systemPrompt = strings.TrimSpace(prompt)
}

func (r *Runner) takeInterventions(transcript *[]*llmtypes.LLMMessage, onEvent AgentEventHandler) bool {
	if r == nil || r.intervention == nil || transcript == nil {
		return false
	}
	applied := false
	for _, item := range r.intervention() {
		if trimmed := strings.TrimSpace(item.Content); trimmed != "" {
			*transcript = append(*transcript, &llmtypes.LLMMessage{Role: llmtypes.RoleUser, Content: "User intervention while you were working. Treat this as the latest steering instruction and adjust your next step:\n" + trimmed})
			applied = true
			if onEvent != nil && strings.TrimSpace(item.ID) != "" {
				onEvent(AgentEvent{Type: "intervention_applied", InterventionID: item.ID, CreatedAt: time.Now()})
			}
		}
	}
	return applied
}

func (r *Runner) Run(ctx context.Context, input []*llmtypes.LLMMessage, onEvent AgentEventHandler) (*RuntimeResult, error) {
	if r.client == nil {
		return nil, fmt.Errorf("runtime runner llm client is nil")
	}
	if r.executeTool == nil {
		return nil, fmt.Errorf("runtime runner execute tool func is nil")
	}

	ctx = llmcore.WithRetryNotifier(ctx, func(attempt, maxAttempts int, providerName, operation string) {
		if onEvent == nil {
			return
		}
		onEvent(AgentEvent{
			Type:         "provider_retry",
			Content:      strings.TrimSpace(providerName + " " + operation),
			RetryAttempt: attempt,
			RetryMax:     maxAttempts,
			CreatedAt:    time.Now(),
		})
	})

	transcript := make([]*llmtypes.LLMMessage, 0, len(input)+1)
	if prompt := strings.TrimSpace(r.systemPrompt); prompt != "" {
		transcript = append(transcript, &llmtypes.LLMMessage{Role: llmtypes.RoleSystem, Content: prompt})
	}
	transcript = append(transcript, llmtypes.CloneMessages(input)...)

	result := &RuntimeResult{}
	var accumulatedUsage *llmtypes.TokenUsage
	lastAssistantTerminalText := ""
	depth := 0
	toolCalls := 0

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		r.takeInterventions(&transcript, onEvent)

		if onEvent != nil {
			onEvent(AgentEvent{
				Depth:     depth,
				Type:      "thinking",
				Content:   "thinking...",
				ToolCalls: toolCalls,
				CreatedAt: time.Now(),
			})
		}

		reader, err := r.client.Stream(ctx, transcript, llmtypes.WithModel(r.model), llmtypes.WithTools(r.tools))
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return nil, err
		}
		msg, err := r.collectStreamWithDelta(ctx, reader, depth, toolCalls, onEvent)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return nil, err
		}
		if msg == nil {
			result.StoppedEarly = true
			result.StopReason = "empty_final_answer"
			break
		}

		assistant := msg
		transcript = append(transcript, assistant)
		result.Messages = append(result.Messages, assistant)
		accumulatedUsage = AddTokenUsage(accumulatedUsage, messageUsage(assistant))

		depth++
		result.Iterations = depth

		if text := assistantTerminalAnswerText(assistant); text != "" {
			lastAssistantTerminalText = text
		}

		if len(assistant.ToolCalls) == 0 {
			if r.takeInterventions(&transcript, onEvent) {
				continue
			}
			result.FinalAnswer = extractFinalAnswer(assistant, lastAssistantTerminalText)
			if strings.TrimSpace(result.FinalAnswer) == "" {
				result.StoppedEarly = true
				result.StopReason = "empty_final_answer"
			} else if onEvent != nil {
				onEvent(AgentEvent{
					Depth:     depth,
					Type:      "final",
					Content:   result.FinalAnswer,
					ToolCalls: toolCalls,
					CreatedAt: time.Now(),
				})
			}
			break
		}

		toolCalls = emitActionsFromMessage(onEvent, depth, toolCalls, assistant)

		for _, call := range assistant.ToolCalls {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			toolName := strings.TrimSpace(call.Function.Name)
			if toolName == "exit" {
				finalAnswer := exitToolCallFinalResult(call)
				if finalAnswer == "" {
					finalAnswer = extractFinalAnswer(assistant, lastAssistantTerminalText)
				}
				result.FinalAnswer = finalAnswer
				result.ToolCalls = toolCalls
				result.TokenUsage = accumulatedUsage
				if strings.TrimSpace(finalAnswer) == "" {
					result.StoppedEarly = true
					result.StopReason = "empty_final_answer"
				} else if onEvent != nil {
					onEvent(AgentEvent{
						Depth:     depth,
						Type:      "final",
						Content:   finalAnswer,
						ToolCalls: toolCalls,
						CreatedAt: time.Now(),
					})
				}
				return finalizeRuntimeResult(result, transcript, accumulatedUsage, toolCalls), nil
			}

			safeResult := r.executeTool(ctx, toolName, call.Function.Arguments, r.sessionID)
			toolMessage := toolResultMessage(call, safeResult)
			transcript = append(transcript, toolMessage)
			result.Messages = append(result.Messages, llmtypes.CloneMessage(toolMessage))
			emitObservationFromMessage(onEvent, depth, toolCalls, toolMessage)
		}
	}

	return finalizeRuntimeResult(result, transcript, accumulatedUsage, toolCalls), nil
}

func finalizeRuntimeResult(result *RuntimeResult, transcript []*llmtypes.LLMMessage, usage *llmtypes.TokenUsage, toolCalls int) *RuntimeResult {
	if result == nil {
		result = &RuntimeResult{}
	}
	result.Messages = llmtypes.CloneMessages(transcript)
	result.ToolCalls = toolCalls
	result.TokenUsage = CloneTokenUsage(usage)
	return result
}

func (r *Runner) collectStreamWithDelta(
	ctx context.Context,
	reader *llmtypes.LLMStreamReader,
	depth, toolCalls int,
	onEvent AgentEventHandler,
) (*llmtypes.LLMMessage, error) {
	var onTextDelta func(string)
	var onReasoningDelta func(string)
	if onEvent != nil {
		onTextDelta = func(delta string) {
			if ctx.Err() != nil {
				return
			}
			onEvent(AgentEvent{
				Depth:     depth,
				Type:      "text_delta",
				Content:   delta,
				ToolCalls: toolCalls,
				CreatedAt: time.Now(),
			})
		}
		onReasoningDelta = func(delta string) {
			if ctx.Err() != nil {
				return
			}
			onEvent(AgentEvent{
				Depth:     depth,
				Type:      "thought",
				Content:   delta,
				ToolCalls: toolCalls,
				CreatedAt: time.Now(),
			})
		}
	}
	return llmtypes.CollectStreamWithDeltas(reader, onTextDelta, onReasoningDelta)
}
