package discovery

import (
	"encoding/json"
	"sort"
	"strings"

	"Postwoman/internal/model"

	"gopkg.in/yaml.v3"
)

var httpMethods = map[string]bool{
	"get": true, "post": true, "put": true, "patch": true,
	"delete": true, "head": true, "options": true, "trace": true,
}

// extractOpenAPI parses OpenAPI 2/3, Swagger and AsyncAPI documents. These are
// the highest-confidence sources, so they are merged first and win conflicts.
func extractOpenAPI(a *analyzer, f *fileEntry) {
	lower := strings.ToLower(f.Rel)
	if !strings.Contains(lower, "openapi") && !strings.Contains(lower, "swagger") &&
		!strings.Contains(lower, "api.json") && !strings.Contains(lower, "api.yaml") &&
		!strings.Contains(lower, "api.yml") && !strings.Contains(lower, "asyncapi") {
		return
	}
	doc := decodeDocument(f)
	if doc == nil {
		return
	}
	if doc["openapi"] == nil && doc["swagger"] == nil && doc["asyncapi"] == nil {
		return
	}
	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		return
	}
	baseURL := openAPIBaseURL(doc)
	authType := openAPIAuthType(doc)
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, routePath := range keys {
		item, _ := paths[routePath].(map[string]any)
		if item == nil {
			continue
		}
		for method, raw := range item {
			if !httpMethods[strings.ToLower(method)] {
				continue
			}
			op, _ := raw.(map[string]any)
			if op == nil {
				continue
			}
			ep := model.DiscoveredEndpoint{
				Method: strings.ToUpper(method), Path: normalizeRoutePath(routePath),
				SourceFile: f.Rel, Origin: "openapi", Confidence: 95,
				Framework: "OpenAPI", Description: stringValue(op["description"]),
			}
			if ep.Description == "" {
				ep.Description = stringValue(op["summary"])
			}
			if tags, ok := op["tags"].([]any); ok {
				for _, tag := range tags {
					if name := stringValue(tag); name != "" {
						ep.Tags = append(ep.Tags, name)
					}
				}
			}
			if operationID := stringValue(op["operationId"]); operationID != "" {
				ep.Handler = operationID
				ep.Name = humanizeIdentifier(operationID)
			}
			if params, ok := op["parameters"].([]any); ok {
				applyOpenAPIParams(&ep, params)
			}
			if body, ok := op["requestBody"].(map[string]any); ok {
				applyOpenAPIRequestBody(&ep, body)
			}
			if responses, ok := op["responses"].(map[string]any); ok {
				applyOpenAPIResponses(&ep, responses)
			}
			if security, ok := op["security"].([]any); ok && len(security) > 0 {
				ep.AuthRequired = true
				ep.AuthType = "security"
				ep.AuthNotes = "security requirement declared in specification"
			} else if authType != "" {
				ep.AuthRequired = true
				ep.AuthType = authType
				ep.AuthNotes = "global security scheme in specification"
			}
			if baseURL != "" {
				ep.Notes = append(ep.Notes, "base URL: "+baseURL)
			}
			ep.Selected = true
			a.addEndpoint(ep)
		}
	}
}

func decodeDocument(f *fileEntry) map[string]any {
	if strings.HasSuffix(strings.ToLower(f.Rel), ".json") {
		var doc map[string]any
		if err := json.Unmarshal([]byte(f.Content), &doc); err == nil {
			return doc
		}
		return nil
	}
	var raw any
	if err := yaml.Unmarshal([]byte(f.Content), &raw); err != nil {
		return nil
	}
	return normalizeYAMLMap(raw)
}

func normalizeYAMLMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = normalizeYAMLValue(child)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[stringValue(key)] = normalizeYAMLValue(child)
		}
		return out
	}
	return nil
}

func normalizeYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any, map[any]any:
		return normalizeYAMLMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = normalizeYAMLValue(child)
		}
		return out
	}
	return value
}
