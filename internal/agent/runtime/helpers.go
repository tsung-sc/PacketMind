package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/llmcore"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

func parseToolArguments(arguments string) (map[string]any, error) {
	if strings.TrimSpace(arguments) == "" {
		return map[string]any{}, nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}

func getStringArg(args map[string]any, key, fallback string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		return strings.TrimSpace(v)
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", v))
		if text == "" {
			return fallback
		}
		return text
	}
}

func truncateForModel(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "...(truncated)"
}

func mustMarshalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `{"ok":false,"error":"failed to encode json"}`
	}
	return string(data)
}

func cloneToolDefinitions(tools []*llmtypes.ToolDefinition) []*llmtypes.ToolDefinition {
	cloned := make([]*llmtypes.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		copyTool := *tool
		if tool.ParamsOneOf != nil {
			params, _ := tool.ParamsOneOf.ToJSONSchema()
			copyTool.Extra = cloneJSONObject(copyTool.Extra)
			if params != nil {
				if copyTool.Extra == nil {
					copyTool.Extra = map[string]any{}
				}
				if _, exists := copyTool.Extra["json_schema"]; !exists {
					copyTool.Extra["json_schema"] = params
				}
			}
			copyTool.ParamsOneOf = llmtypes.NewToolParams(tool.ParamsOneOf.Params)
		} else if len(tool.Extra) > 0 {
			copyTool.Extra = cloneJSONObject(tool.Extra)
		}
		cloned = append(cloned, &copyTool)
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func cloneJSONObject(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var clone map[string]any
	if err := json.Unmarshal(raw, &clone); err != nil {
		return nil
	}
	return clone
}

func toolExecutionResultToString(result *ToolExecutionResult) (string, error) {
	if result == nil {
		return "", nil
	}
	if strings.TrimSpace(result.Content) != "" {
		return result.Content, nil
	}
	if strings.TrimSpace(result.Summary) != "" {
		return result.Summary, nil
	}
	return "", nil
}

func encodeSafeToolError(err *llmcore.AgentError) string {
	payload := map[string]any{
		"ok":        false,
		"error":     "tool execution failed",
		"category":  "recoverable",
		"recovered": false,
	}
	if err != nil {
		payload["error"] = err.Error()
		payload["category"] = string(err.Category)
		payload["tool_name"] = err.ToolName
		payload["recovered"] = err.Recovered
		if err.Timeout > 0 {
			payload["timeout"] = err.Timeout.String()
		}
	}
	return mustMarshalJSON(payload)
}
