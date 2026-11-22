package discovery

import (
	"encoding/json"
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

// requestTypeName finds a DTO/struct type used to bind the request body.
func requestTypeName(body string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:var|let|const)\s+\w+\s+([A-Z]\w*)\s*\{`),
		regexp.MustCompile(`(?i)(?:var|let|const)\s+\w+\s+([A-Z]\w*)\b`),
		regexp.MustCompile(`(?i)&\s*([A-Z]\w*)\s*\{`),
		regexp.MustCompile(`(?i):\s*([A-Z]\w*)\s*=\s*req\.body`),
		regexp.MustCompile(`(?i)as\s+([A-Z]\w*)`),
	}
	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(body); match != nil {
			return match[1]
		}
	}
	return ""
}

func jsonSchemaFromBody(body string) []model.DiscoverySchemaField {
	var value any
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return fieldsFromJSON(object, 0)
}

func fieldsFromJSON(object map[string]any, depth int) []model.DiscoverySchemaField {
	if depth > 3 {
		return nil
	}
	fields := make([]model.DiscoverySchemaField, 0, len(object))
	for key, value := range object {
		field := model.DiscoverySchemaField{Name: key}
		switch typed := value.(type) {
		case string:
			field.Type = "string"
			field.Example = typed
		case bool:
			field.Type = "boolean"
		case float64:
			field.Type = "number"
		case map[string]any:
			field.Type = "object"
			field.Children = fieldsFromJSON(typed, depth+1)
		case []any:
			field.Type = "array"
			if len(typed) > 0 {
				if child, ok := typed[0].(map[string]any); ok {
					field.Children = fieldsFromJSON(child, depth+1)
				}
			}
		}
		fields = append(fields, field)
	}
	return fields
}

// structFields extracts the fields of a Go struct or TypeScript interface so a
// request body example can be generated without an AI call.
func (a *analyzer) structFields(typeName string) []model.DiscoverySchemaField {
	for _, f := range a.files {
		switch f.Language {
		case "go":
			if match := goStructPattern.FindStringSubmatchIndex(f.Content); match != nil {
				if f.Content[match[2]:match[3]] != typeName {
					continue
				}
				body, _ := splitBraceBody(f.Content, match[1]-1)
				return goStructFields(body)
			}
		case "typescript", "javascript":
			if match := tsInterfacePattern.FindStringSubmatchIndex(f.Content); match != nil {
				if f.Content[match[2]:match[3]] != typeName {
					continue
				}
				body, _ := splitBraceBody(f.Content, match[1]-1)
				return tsInterfaceFields(body)
			}
		}
	}
	return nil
}

func goStructFields(body string) []model.DiscoverySchemaField {
	fields := []model.DiscoverySchemaField{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if name == "" || strings.HasPrefix(name, "}") {
			continue
		}
		jsonName := name
		if tagIndex := strings.Index(line, "json:\""); tagIndex >= 0 {
			tag := line[tagIndex+6:]
			if end := strings.Index(tag, "\""); end > 0 {
				tag = tag[:end]
				if comma := strings.Index(tag, ","); comma >= 0 {
					tag = tag[:comma]
				}
				if tag != "" && tag != "-" {
					jsonName = tag
				}
			}
		}
		fieldType := parts[1]
		if strings.HasPrefix(fieldType, "*") {
			fieldType = strings.TrimPrefix(fieldType, "*")
		}
		fields = append(fields, model.DiscoverySchemaField{
			Name: jsonName, Type: goTypeToJSON(fieldType), Required: !strings.Contains(line, "omitempty"),
		})
	}
	return fields
}

func goTypeToJSON(goType string) string {
	switch {
	case strings.HasPrefix(goType, "[]"):
		return "array"
	case strings.HasPrefix(goType, "map["):
		return "object"
	case goType == "string":
		return "string"
	case goType == "bool":
		return "boolean"
	case strings.HasPrefix(goType, "int"), strings.HasPrefix(goType, "uint"), strings.HasPrefix(goType, "float"):
		return "number"
	case goType == "time.Time":
		return "string"
	}
	return "object"
}

func tsInterfaceFields(body string) []model.DiscoverySchemaField {
	fields := []model.DiscoverySchemaField{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "}") {
			continue
		}
		colon := strings.Index(line, ":")
		if colon <= 0 {
			continue
		}
		name := strings.TrimSpace(line[:colon])
		name = strings.TrimSuffix(name, "?")
		if name == "" || strings.ContainsAny(name, " ()") {
			continue
		}
		typeName := strings.TrimSpace(strings.TrimSuffix(line[colon+1:], ";"))
		fields = append(fields, model.DiscoverySchemaField{
			Name: name, Type: tsTypeToJSON(typeName), Required: !strings.Contains(line[:colon], "?"),
		})
	}
	return fields
}

func tsTypeToJSON(tsType string) string {
	lower := strings.ToLower(tsType)
	switch {
	case strings.HasPrefix(lower, "string"):
		return "string"
	case strings.HasPrefix(lower, "number"):
		return "number"
	case strings.HasPrefix(lower, "boolean"):
		return "boolean"
	case strings.HasSuffix(lower, "[]") || strings.HasPrefix(lower, "array"):
		return "array"
	case strings.HasPrefix(lower, "record") || strings.HasPrefix(lower, "{"):
		return "object"
	}
	return "object"
}
