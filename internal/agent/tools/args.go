package tools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const defaultSearchLimit = 10

type BatchToolCall struct {
	Tool string `json:"tool"`
	Args string `json:"args"`
}

func ParseToolArguments(arguments string) (map[string]any, error) {
	if strings.TrimSpace(arguments) == "" {
		return map[string]any{}, nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}

func GetRequiredStringArg(args map[string]any, key string) (string, error) {
	value := GetStringArg(args, key, "")
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func GetStringArg(args map[string]any, key, fallback string) string {
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

func GetIntArg(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok || value == nil {
		return NormalizeLimit(fallback)
	}

	switch v := value.(type) {
	case float64:
		return NormalizeLimit(int(v))
	case float32:
		return NormalizeLimit(int(v))
	case int:
		return NormalizeLimit(v)
	case int64:
		return NormalizeLimit(int(v))
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return NormalizeLimit(int(n))
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return NormalizeLimit(n)
		}
	}

	return NormalizeLimit(fallback)
}

func GetBoolArg(args map[string]any, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(strings.TrimSpace(v)) == "true"
	default:
		return fallback
	}
}

func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func ParseBatchCallsArg(args map[string]any, key string) []BatchToolCall {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}

	list, ok := value.([]any)
	if !ok {
		return nil
	}

	calls := make([]BatchToolCall, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		calls = append(calls, BatchToolCall{
			Tool: GetStringArg(entry, "tool", ""),
			Args: GetStringArg(entry, "args", "{}"),
		})
	}

	return calls
}

func ParseDiffFieldsArg(args map[string]any, key string) []string {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}

	switch v := value.(type) {
	case []any:
		fields := make([]string, 0, len(v))
		for _, item := range v {
			switch itemValue := item.(type) {
			case string:
				fields = append(fields, itemValue)
			default:
				fields = append(fields, strings.TrimSpace(fmt.Sprintf("%v", itemValue)))
			}
		}
		return fields
	case string:
		return []string{v}
	default:
		return nil
	}
}
