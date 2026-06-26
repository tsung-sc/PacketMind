package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

func emitThoughtFromMessage(onEvent AgentEventHandler, depth, toolCallCount int, msg *llmtypes.LLMMessage) {
	content := strings.TrimSpace(msgReasoningOrContent(msg))
	if content == "" || onEvent == nil {
		return
	}

	onEvent(AgentEvent{
		Depth:     depth,
		Type:      "thought",
		Content:   content,
		ToolCalls: toolCallCount,
		CreatedAt: time.Now(),
	})
}

func emitActionsFromMessage(onEvent AgentEventHandler, depth, toolCallCount int, msg *llmtypes.LLMMessage) int {
	if onEvent == nil || msg == nil {
		return toolCallCount
	}

	for _, call := range msg.ToolCalls {
		toolCallCount++
		onEvent(AgentEvent{
			Depth:     depth,
			Type:      "action",
			ToolName:  call.Function.Name,
			Arguments: call.Function.Arguments,
			ToolCalls: toolCallCount,
			CreatedAt: time.Now(),
		})
	}

	return toolCallCount
}

func emitObservationFromMessage(onEvent AgentEventHandler, depth, toolCallCount int, msg *llmtypes.LLMMessage) {
	if onEvent == nil || msg == nil {
		return
	}

	result := strings.TrimSpace(msg.Content)
	metadata := extractObservationEventMetadata(msg.ToolName, result)

	summary := strings.TrimSpace(firstNonEmptyLine(result))
	if summary == "" {
		summary = fmt.Sprintf("Tool %s completed", strings.TrimSpace(msg.ToolName))
	}
	summary = truncateForModel(summary, 240)

	onEvent(AgentEvent{
		Depth:          depth,
		Type:           "observation",
		Content:        summary,
		ToolName:       msg.ToolName,
		Result:         result,
		RequestIDs:     metadata.RequestIDs,
		ToolCalls:      toolCallCount,
		CreatedAt:      time.Now(),
		ErrorCategory:  metadata.ErrorCategory,
		ErrorToolName:  metadata.ErrorToolName,
		ErrorTimeout:   metadata.ErrorTimeout,
		ErrorRecovered: metadata.ErrorRecovered,
	})
}

func extractFinalAnswer(msg *llmtypes.LLMMessage, fallback string) string {
	if text := exitToolFinalResult(msg); text != "" {
		return text
	}
	if text := terminalAnswerText(msg); text != "" {
		return text
	}
	return strings.TrimSpace(fallback)
}

type observationEventMetadata struct {
	RequestIDs     []string
	ErrorCategory  string
	ErrorToolName  string
	ErrorTimeout   string
	ErrorRecovered bool
}

func toolResultMessage(call llmtypes.ToolCall, safeResult *SafeToolResult) *llmtypes.LLMMessage {
	msg := &llmtypes.LLMMessage{
		Role:       llmtypes.RoleTool,
		ToolCallID: call.ID,
		ToolName:   strings.TrimSpace(call.Function.Name),
	}
	if safeResult == nil {
		msg.Content = mustMarshalJSON(map[string]any{"ok": false, "error": "tool execution failed"})
		return msg
	}
	if safeResult.Err != nil {
		msg.Content = encodeSafeToolError(safeResult.Err)
		return msg
	}
	content, _ := toolExecutionResultToString(safeResult.Result)
	msg.Content = content
	return msg
}

func msgReasoningOrContent(msg *llmtypes.LLMMessage) string {
	if msg == nil {
		return ""
	}
	if text := strings.TrimSpace(msg.ReasoningContent); text != "" {
		return text
	}
	return strings.TrimSpace(msg.Content)
}

func assistantTerminalAnswerText(msg *llmtypes.LLMMessage) string {
	if text := exitToolFinalResult(msg); text != "" {
		return text
	}
	return terminalAnswerText(msg)
}

func terminalAnswerText(msg *llmtypes.LLMMessage) string {
	if msg == nil {
		return ""
	}
	if text := strings.TrimSpace(msg.Content); text != "" {
		return text
	}
	if len(msg.MultiContent) > 0 {
		var parts []string
		for _, part := range msg.MultiContent {
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n\n")
		}
	}
	return strings.TrimSpace(msg.ReasoningContent)
}

func exitToolFinalResult(msg *llmtypes.LLMMessage) string {
	if msg == nil {
		return ""
	}
	for _, call := range msg.ToolCalls {
		if text := exitToolCallFinalResult(call); text != "" {
			return text
		}
	}
	return ""
}

func exitToolCallFinalResult(call llmtypes.ToolCall) string {
	if strings.TrimSpace(call.Function.Name) != "exit" {
		return ""
	}
	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return ""
	}
	return getStringArg(args, "final_result", "")
}

func firstNonEmptyLine(text string) string {
	for line := range strings.SplitSeq(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func extractObservationEventMetadata(toolName, raw string) observationEventMetadata {
	meta := observationEventMetadata{}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return meta
	}

	meta.RequestIDs = uniqueStrings(append(meta.RequestIDs, collectRequestIDsFromPayload(payload)...))
	meta.ErrorCategory = stringFromMap(payload, "category")
	meta.ErrorToolName = stringFromMap(payload, "tool_name")
	meta.ErrorTimeout = stringFromMap(payload, "timeout")
	meta.ErrorRecovered = boolFromMap(payload, "recovered")
	if meta.ErrorToolName == "" {
		meta.ErrorToolName = strings.TrimSpace(toolName)
	}

	return meta
}

func collectRequestIDsFromPayload(payload map[string]any) []string {
	ids := make([]string, 0, 8)
	ids = append(ids, stringFromMap(payload, "request_id"))
	ids = append(ids, stringsFromMap(payload, "request_ids")...)

	if request, ok := payload["request"].(map[string]any); ok {
		ids = append(ids, stringFromMap(request, "id"))
	}

	if results, ok := payload["results"].([]any); ok {
		for _, item := range results {
			result, ok := item.(map[string]any)
			if !ok {
				continue
			}
			ids = append(ids, stringFromMap(result, "request_id"))
			ids = append(ids, stringFromMap(result, "source_request_id"))
		}
	}

	if provenance, ok := payload["provenance"].(map[string]any); ok {
		if links, ok := provenance["links"].([]any); ok {
			for _, item := range links {
				link, ok := item.(map[string]any)
				if !ok {
					continue
				}
				ids = append(ids, stringFromMap(link, "source_request_id"))
			}
		}
	}

	return uniqueStrings(ids)
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func stringsFromMap(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	raw, ok := values[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if value, ok := item.(string); ok {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func boolFromMap(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	value, _ := values[key].(bool)
	return value
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
