package discovery

import (
	"encoding/json"
	"sort"
	"strconv"

	"Postwoman/internal/model"
)

func openAPIBaseURL(doc map[string]any) string {
	if servers, ok := doc["servers"].([]any); ok && len(servers) > 0 {
		if server, ok := servers[0].(map[string]any); ok {
			return stringValue(server["url"])
		}
	}
	if host := stringValue(doc["host"]); host != "" {
		scheme := "https"
		if schemes, ok := doc["schemes"].([]any); ok && len(schemes) > 0 {
			scheme = stringValue(schemes[0])
		}
		return scheme + "://" + host + stringValue(doc["basePath"])
	}
	return ""
}

func openAPIAuthType(doc map[string]any) string {
	components, _ := doc["components"].(map[string]any)
	if components == nil {
		components, _ = doc["securityDefinitions"].(map[string]any)
	}
	schemes, _ := components["securitySchemes"].(map[string]any)
	if schemes == nil {
		schemes, _ = components["securityDefinitions"].(map[string]any)
	}
	for _, raw := range schemes {
		scheme, _ := raw.(map[string]any)
		if scheme == nil {
			continue
		}
		if kind := stringValue(scheme["type"]); kind != "" {
			return kind
		}
	}
	return ""
}

func applyOpenAPIParams(ep *model.DiscoveredEndpoint, params []any) {
	for _, raw := range params {
		param, _ := raw.(map[string]any)
		if param == nil {
			continue
		}
		name := stringValue(param["name"])
		if name == "" {
			continue
		}
		location := stringValue(param["in"])
		schema, _ := param["schema"].(map[string]any)
		discoveryParam := model.DiscoveryParam{
			Name: name, In: location, Required: boolValue(param["required"]),
			Description: stringValue(param["description"]),
			Type:        stringValue(schema["type"]),
			Example:     stringValue(param["example"]),
		}
		switch location {
		case "path":
			ep.PathParams = append(ep.PathParams, discoveryParam)
		case "query":
			ep.QueryParams = append(ep.QueryParams, discoveryParam)
		case "header":
			ep.Headers = append(ep.Headers, discoveryParam)
		}
	}
}

func applyOpenAPIRequestBody(ep *model.DiscoveredEndpoint, body map[string]any) {
	content, _ := body["content"].(map[string]any)
	if content == nil {
		return
	}
	keys := make([]string, 0, len(content))
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, contentType := range keys {
		media, _ := content[contentType].(map[string]any)
		schema, _ := media["schema"].(map[string]any)
		ep.ContentType = contentType
		ep.RequestSchema = schemaFields(schema, 0)
		if example := media["example"]; example != nil {
			if encoded, err := json.MarshalIndent(example, "", "  "); err == nil {
				ep.RequestBody = string(encoded)
			}
		}
		if ep.RequestBody == "" && len(ep.RequestSchema) > 0 {
			ep.RequestBody = schemaExampleJSON(ep.RequestSchema)
		}
		if ep.ContentType == "" {
			ep.ContentType = "application/json"
		}
		return
	}
}

func applyOpenAPIResponses(ep *model.DiscoveredEndpoint, responses map[string]any) {
	codes := make([]string, 0, len(responses))
	for code := range responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if numeric, err := strconv.Atoi(code); err == nil {
			ep.StatusCodes = append(ep.StatusCodes, numeric)
		}
		response, _ := responses[code].(map[string]any)
		if response == nil || len(ep.ResponseSchema) > 0 {
			continue
		}
		content, _ := response["content"].(map[string]any)
		for _, raw := range content {
			media, _ := raw.(map[string]any)
			if schema, ok := media["schema"].(map[string]any); ok {
				ep.ResponseSchema = schemaFields(schema, 0)
				break
			}
		}
	}
}

func schemaFields(schema map[string]any, depth int) []model.DiscoverySchemaField {
	if schema == nil || depth > 4 {
		return nil
	}
	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		if items, ok := schema["items"].(map[string]any); ok {
			return schemaFields(items, depth+1)
		}
		return nil
	}
	required := map[string]bool{}
	if names, ok := schema["required"].([]any); ok {
		for _, name := range names {
			required[stringValue(name)] = true
		}
	}
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]model.DiscoverySchemaField, 0, len(keys))
	for _, key := range keys {
		raw, _ := properties[key].(map[string]any)
		field := model.DiscoverySchemaField{
			Name: key, Required: required[key],
			Type:        stringValue(raw["type"]),
			Description: stringValue(raw["description"]),
			Example:     stringValue(raw["example"]),
		}
		if field.Type == "" {
			if _, ok := raw["properties"]; ok {
				field.Type = "object"
			} else if _, ok := raw["items"]; ok {
				field.Type = "array"
			}
		}
		if field.Type == "object" || field.Type == "array" {
			field.Children = schemaFields(raw, depth+1)
		}
		fields = append(fields, field)
	}
	return fields
}

func schemaExampleJSON(fields []model.DiscoverySchemaField) string {
	encoded, err := json.MarshalIndent(fieldsToValue(fields), "", "  ")
	if err != nil {
		return ""
	}
	return string(encoded)
}

func fieldsToValue(fields []model.DiscoverySchemaField) map[string]any {
	out := map[string]any{}
	for _, field := range fields {
		if len(field.Children) > 0 {
			out[field.Name] = fieldsToValue(field.Children)
			continue
		}
		switch field.Type {
		case "integer", "number":
			out[field.Name] = 0
		case "boolean":
			out[field.Name] = false
		case "array":
			out[field.Name] = []any{}
		default:
			if field.Example != "" {
				out[field.Name] = field.Example
			} else {
				out[field.Name] = ""
			}
		}
	}
	return out
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true"
	}
	return false
}
