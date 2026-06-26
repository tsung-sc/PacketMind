package tools

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

func ValidateToolParams(toolName string, params map[string]any, schema map[string]any) error {
	schemaType, _ := schema["type"].(string)
	if schemaType != "object" {
		return nil
	}

	properties, _ := schema["properties"].(map[string]any)
	required := requiredFields(schema["required"])

	var errs []string

	for _, fieldName := range required {
		val, exists := params[fieldName]
		if !exists || val == nil {
			errs = append(errs, fmt.Sprintf("missing required parameter '%s'", fieldName))
			continue
		}

		if propSchema, ok := properties[fieldName].(map[string]any); ok {
			if typeErr := checkType(fieldName, val, propSchema); typeErr != "" {
				errs = append(errs, typeErr)
			}
		}
	}

	for key, val := range params {
		if isRequired(key, required) {
			continue
		}
		if propSchema, ok := properties[key].(map[string]any); ok {
			if typeErr := checkType(key, val, propSchema); typeErr != "" {
				errs = append(errs, typeErr)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("tool '%s' parameter validation failed: %s", toolName, strings.Join(errs, "; "))
	}

	return nil
}

func checkType(fieldName string, value any, schema map[string]any) string {
	expectedType, _ := schema["type"].(string)
	if expectedType == "" {
		return ""
	}

	switch expectedType {
	case "string":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Sprintf("parameter '%s' must be a string, got %T", fieldName, value)
		}

		if enumErr := checkEnum(fieldName, strVal, schema["enum"]); enumErr != "" {
			return enumErr
		}

	case "integer":
		num, ok := toNumber(value)
		if !ok {
			return fmt.Sprintf("parameter '%s' must be an integer, got %T", fieldName, value)
		}
		if math.Mod(num, 1) != 0 {
			return fmt.Sprintf("parameter '%s' must be an integer, got %v", fieldName, value)
		}

		if min, ok := toFloat(schema["minimum"]); ok && num < min {
			return fmt.Sprintf("parameter '%s' must be >= %v, got %v", fieldName, trimFloat(min), trimFloat(num))
		}
		if max, ok := toFloat(schema["maximum"]); ok && num > max {
			return fmt.Sprintf("parameter '%s' must be <= %v, got %v", fieldName, trimFloat(max), trimFloat(num))
		}
	}

	return ""
}

func checkEnum(fieldName, value string, enumRaw any) string {
	enumVals := enumStrings(enumRaw)
	if len(enumVals) == 0 {
		return ""
	}

	for _, allowed := range enumVals {
		if allowed == value {
			return ""
		}
	}

	return fmt.Sprintf("parameter '%s' value '%s' is not one of allowed values: [%s]", fieldName, value, strings.Join(enumVals, ", "))
}

func isRequired(field string, required []string) bool {
	for _, r := range required {
		if r == field {
			return true
		}
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func toNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func trimFloat(v float64) any {
	if math.Mod(v, 1) == 0 {
		return int64(v)
	}
	return v
}
