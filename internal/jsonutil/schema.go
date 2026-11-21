package jsonutil

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// InferSchema builds a permissive JSON Schema (draft-07 style) that describes
// the shape of a decoded JSON document. It is intended as a starting point for
// contract tests, not as a canonical schema.
func InferSchema(value any) map[string]any {
	switch v := value.(type) {
	case nil:
		return map[string]any{"type": "null"}
	case bool:
		return map[string]any{"type": "boolean"}
	case string:
		return map[string]any{"type": "string"}
	case json.Number:
		if isIntegral(v.String()) {
			return map[string]any{"type": "integer"}
		}
		return map[string]any{"type": "number"}
	case float64:
		if v == math.Trunc(v) {
			return map[string]any{"type": "integer"}
		}
		return map[string]any{"type": "number"}
	case []any:
		out := map[string]any{"type": "array"}
		if len(v) > 0 {
			first := InferSchema(v[0])
			same := true
			for _, item := range v[1:] {
				if TypeName(item) != TypeName(v[0]) {
					same = false
					break
				}
			}
			if same {
				out["items"] = first
			}
		}
		return out
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		props := map[string]any{}
		for _, k := range keys {
			props[k] = InferSchema(v[k])
		}
		return map[string]any{"type": "object", "properties": props, "required": keys}
	default:
		return map[string]any{}
	}
}

func isIntegral(s string) bool {
	if strings.ContainsAny(s, ".eE") {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

// MarshalSchema renders an inferred schema with stable indentation.
func MarshalSchema(schema map[string]any) (string, error) {
	schema["$schema"] = "http://json-schema.org/draft-07/schema#"
	raw, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Validate checks a JSON instance against the subset of JSON Schema the app
// supports. It returns human-readable errors (empty when valid).
func Validate(instanceRaw, schemaRaw string) ([]string, error) {
	instance, err := Decode(instanceRaw)
	if err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	schema, err := Decode(schemaRaw)
	if err != nil {
		return nil, fmt.Errorf("schema is not valid JSON: %w", err)
	}
	schemaObj, ok := schema.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema must be a JSON object")
	}
	errors := []string{}
	validateNode(instance, schemaObj, "$", &errors)
	return errors, nil
}

func validateNode(v any, schema map[string]any, path string, errors *[]string) {
	if len(*errors) > 200 {
		return
	}
	if ref, ok := schema["$ref"].(string); ok && strings.HasPrefix(ref, "#/") {
		// Local references are not resolved without the whole document; skip
		// gracefully instead of reporting a false error.
		_ = ref
	}
	if expected, ok := schema["type"].(string); ok {
		if !typeMatches(v, expected) {
			*errors = append(*errors, fmt.Sprintf("%s: expected %s, got %s", path, expected, TypeName(v)))
			return
		}
	}
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		found := false
		candidate := fmt.Sprintf("%v", v)
		for _, allowed := range enum {
			if fmt.Sprintf("%v", allowed) == candidate {
				found = true
				break
			}
		}
		if !found {
			*errors = append(*errors, fmt.Sprintf("%s: value is not one of the allowed enum values", path))
		}
	}
	if obj, ok := v.(map[string]any); ok {
		if required, ok := schema["required"].([]any); ok {
			for _, r := range required {
				key, _ := r.(string)
				if _, exists := obj[key]; !exists {
					*errors = append(*errors, fmt.Sprintf("%s: missing required property %q", path, key))
				}
			}
		}
		if props, ok := schema["properties"].(map[string]any); ok {
			for key, sub := range props {
				if child, exists := obj[key]; exists {
					if subSchema, ok := sub.(map[string]any); ok {
						validateNode(child, subSchema, path+"."+key, errors)
					}
				}
			}
		}
	}
	if arr, ok := v.([]any); ok {
		if items, ok := schema["items"].(map[string]any); ok {
			for i, item := range arr {
				validateNode(item, items, fmt.Sprintf("%s[%d]", path, i), errors)
			}
		}
	}
	if number, ok := asFloat(v); ok {
		if minimum, ok := asFloat(schema["minimum"]); ok && number < minimum {
			*errors = append(*errors, fmt.Sprintf("%s: %v is less than the minimum %v", path, number, minimum))
		}
		if maximum, ok := asFloat(schema["maximum"]); ok && number > maximum {
			*errors = append(*errors, fmt.Sprintf("%s: %v is greater than the maximum %v", path, number, maximum))
		}
	}
	if s, ok := v.(string); ok {
		if minLength, ok := asFloat(schema["minLength"]); ok && float64(len(s)) < minLength {
			*errors = append(*errors, fmt.Sprintf("%s: string is shorter than minLength %v", path, minLength))
		}
		if maxLength, ok := asFloat(schema["maxLength"]); ok && float64(len(s)) > maxLength {
			*errors = append(*errors, fmt.Sprintf("%s: string is longer than maxLength %v", path, maxLength))
		}
	}
}

func typeMatches(v any, expected string) bool {
	switch expected {
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "null":
		return v == nil
	case "number":
		_, ok := asFloat(v)
		return ok
	case "integer":
		n, ok := asFloat(v)
		return ok && n == math.Trunc(n)
	default:
		return true
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
