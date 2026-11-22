package discovery

import (
	"net/url"
	"strings"

	"Postwoman/internal/model"
)

func requestItem(ep model.DiscoveredEndpoint, baseURL string) map[string]any {
	pathSegments := splitPath(ep.Path)
	query := make([]map[string]any, 0, len(ep.QueryParams))
	for _, param := range ep.QueryParams {
		query = append(query, map[string]any{
			"key": param.Name, "value": param.Example, "description": param.Description,
			"disabled": !param.Required,
		})
	}
	variables := make([]map[string]any, 0, len(ep.PathParams))
	for _, param := range ep.PathParams {
		variables = append(variables, map[string]any{
			"key": param.Name, "value": param.Example, "description": param.Description,
		})
	}
	raw := "{{baseUrl}}" + ep.Path
	if len(query) > 0 {
		parts := make([]string, 0, len(query))
		for _, q := range query {
			parts = append(parts, url.QueryEscape(q["key"].(string))+"="+url.QueryEscape(stringValue(q["value"])))
		}
		raw += "?" + strings.Join(parts, "&")
	}
	headers := make([]map[string]any, 0, len(ep.Headers)+2)
	headers = append(headers, map[string]any{"key": "Accept", "value": "application/json"})
	seenHeader := map[string]bool{"accept": true}
	for _, header := range ep.Headers {
		key := strings.ToLower(header.Name)
		if key == "" || seenHeader[key] {
			continue
		}
		seenHeader[key] = true
		headers = append(headers, map[string]any{
			"key": header.Name, "value": header.Example, "description": header.Description, "disabled": false,
		})
	}
	request := map[string]any{
		"method": ep.Method,
		"header": headers,
		"url": map[string]any{
			"raw":      raw,
			"host":     []string{"{{baseUrl}}"},
			"path":     pathSegments,
			"query":    query,
			"variable": variables,
		},
	}
	if strings.TrimSpace(ep.Description) != "" {
		request["description"] = ep.Description
	}
	if ep.RequestBody != "" {
		if !seenHeader["content-type"] {
			contentType := ep.ContentType
			if contentType == "" {
				contentType = "application/json"
			}
			headers = append(headers, map[string]any{"key": "Content-Type", "value": contentType})
			request["header"] = headers
		}
		request["body"] = map[string]any{
			"mode": "raw",
			"raw":  ep.RequestBody,
			"options": map[string]any{
				"raw": map[string]any{"language": "json"},
			},
		}
	}
	if ep.AuthRequired {
		request["auth"] = authForEndpoint(ep)
	}
	item := map[string]any{
		"name":     fallbackName(ep),
		"request":  request,
		"response": []any{},
	}
	notes := append([]string{}, ep.Notes...)
	notes = append(notes, ep.Warnings...)
	if len(notes) > 0 {
		item["description"] = strings.TrimSpace(fallbackName(ep) + "\n\n" + strings.Join(notes, "\n"))
	}
	_ = baseURL
	return item
}

func authForEndpoint(ep model.DiscoveredEndpoint) map[string]any {
	kind := strings.ToLower(ep.AuthType)
	switch kind {
	case "basic":
		return map[string]any{"type": "basic", "basic": []map[string]any{
			{"key": "username", "value": "{{username}}", "type": "string"},
			{"key": "password", "value": "{{password}}", "type": "string"},
		}}
	case "apikey":
		return map[string]any{"type": "apikey", "apikey": []map[string]any{
			{"key": "key", "value": "X-API-Key", "type": "string"},
			{"key": "value", "value": "{{apiKey}}", "type": "string"},
			{"key": "in", "value": "header", "type": "string"},
		}}
	case "oauth2":
		return map[string]any{"type": "oauth2", "oauth2": []map[string]any{
			{"key": "accessToken", "value": "{{authToken}}", "type": "string"},
			{"key": "grant_type", "value": "authorization_code", "type": "string"},
		}}
	default:
		return map[string]any{"type": "bearer", "bearer": []map[string]any{
			{"key": "token", "value": "{{authToken}}", "type": "string"},
		}}
	}
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{""}
	}
	segments := strings.Split(trimmed, "/")
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" {
			out = append(out, segment)
		}
	}
	return out
}

func fallbackName(ep model.DiscoveredEndpoint) string {
	if strings.TrimSpace(ep.Name) != "" {
		return ep.Name
	}
	return routeDisplayName(ep.Method, ep.Path, ep.Handler)
}
