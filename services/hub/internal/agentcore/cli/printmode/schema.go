package printmode

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

func ParseJSONSchema(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("Invalid --json-schema: %v", err)
	}

	obj, ok := parsed.(map[string]any)
	if !ok {
		return nil, errors.New("Invalid --json-schema: Schema must be a JSON object")
	}
	return obj, nil
}

func ParseJSONObject(raw string) (map[string]any, error) {
	unfenced := strings.TrimSpace(removeCodeFence(raw))
	if unfenced == "" {
		return nil, errors.New("output is empty")
	}
	var parsed any
	if err := json.Unmarshal([]byte(unfenced), &parsed); err != nil {
		return nil, err
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return nil, errors.New("Structured output must be a JSON object")
	}
	return obj, nil
}

func ValidateOutputAgainstSchema(raw string, schema map[string]any) (map[string]any, error) {
	output, err := ParseJSONObject(raw)
	if err != nil {
		return nil, err
	}
	if len(schema) == 0 {
		return output, nil
	}
	if validateErr := validateSchemaValue("$", output, schema); validateErr != nil {
		return nil, fmt.Errorf("Structured output failed JSON schema validation: %v", validateErr)
	}
	return output, nil
}

func validateSchemaValue(path string, value any, schema map[string]any) error {
	expectedType, _ := schema["type"].(string)
	expectedType = strings.TrimSpace(expectedType)
	if expectedType != "" {
		if typeErr := validateValueType(path, value, expectedType); typeErr != nil {
			return typeErr
		}
	}

	if expectedType != "object" {
		return nil
	}

	obj, _ := value.(map[string]any)
	required, _ := schema["required"].([]any)
	for _, field := range required {
		key, ok := field.(string)
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		if _, exists := obj[key]; !exists {
			return fmt.Errorf("%s.%s is required", path, key)
		}
	}

	properties, _ := schema["properties"].(map[string]any)
	for key, propertyDef := range properties {
		subSchema, ok := propertyDef.(map[string]any)
		if !ok {
			continue
		}
		subValue, exists := obj[key]
		if !exists {
			continue
		}
		if err := validateSchemaValue(path+"."+key, subValue, subSchema); err != nil {
			return err
		}
	}

	return nil
}

func validateValueType(path string, value any, expected string) error {
	switch expected {
	case "string":
		if _, ok := value.(string); ok {
			return nil
		}
	case "number":
		switch value.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return nil
		}
	case "integer":
		switch typed := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return nil
		case float64:
			if math.Trunc(typed) == typed {
				return nil
			}
		case float32:
			if math.Trunc(float64(typed)) == float64(typed) {
				return nil
			}
		}
	case "boolean":
		if _, ok := value.(bool); ok {
			return nil
		}
	case "object":
		if _, ok := value.(map[string]any); ok {
			return nil
		}
	case "array":
		if _, ok := value.([]any); ok {
			return nil
		}
	case "null":
		if value == nil {
			return nil
		}
	default:
		return nil
	}
	return fmt.Errorf("%s should be %s", path, expected)
}

func removeCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) < 2 {
		return trimmed
	}
	if !strings.HasPrefix(lines[0], "```") {
		return trimmed
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		return trimmed
	}
	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
}
