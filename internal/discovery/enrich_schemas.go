package discovery

import (
	"context"
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

// enrichSchemas fills in parameters, bodies, status codes and schemas that can
// be inferred from the handler implementation.
func (a *analyzer) enrichSchemas(ctx context.Context) {
	for _, key := range a.order {
		if ctx.Err() != nil {
			return
		}
		ep := a.endpoints[key]
		if ep == nil {
			continue
		}
		ensurePathParams(ep)
		if ep.Handler == "" {
			continue
		}
		body := a.handlerBody(ep.Handler)
		if body == "" {
			continue
		}
		ep.QueryParams = appendUniqueParams(ep.QueryParams, extractParams(body, queryParamPatterns, "query"))
		ep.Headers = appendUniqueParams(ep.Headers, extractParams(body, headerParamPatterns, "header"))
		ep.PathParams = appendUniqueParams(ep.PathParams, extractParams(body, pathVarPatterns, "path"))
		if jsonBodyPattern.MatchString(body) && ep.RequestBody == "" {
			ep.ContentType = "application/json"
			ep.RequestBody = "{}"
			ep.Notes = append(ep.Notes, "request body inferred from handler; example not resolved")
		}
		if codes := extractStatusCodes(body); len(codes) > 0 {
			ep.StatusCodes = appendUniqueInts(ep.StatusCodes, codes)
		}
		if !ep.AuthRequired && hasAuthKeyword(body) {
			ep.AuthRequired = true
			ep.AuthType = "handler"
			ep.AuthNotes = "authorization logic referenced in handler"
		}
		if ep.RequestBody != "" && ep.RequestBody != "{}" && len(ep.RequestSchema) == 0 {
			if schema := jsonSchemaFromBody(ep.RequestBody); len(schema) > 0 {
				ep.RequestSchema = schema
			}
		}
		if len(ep.RequestSchema) == 0 {
			if typeName := requestTypeName(body); typeName != "" {
				if fields := a.structFields(typeName); len(fields) > 0 {
					ep.RequestSchema = fields
					ep.RequestBody = schemaExampleJSON(fields)
					ep.ContentType = "application/json"
				}
			}
		}
	}
}

func ensurePathParams(ep *model.DiscoveredEndpoint) {
	existing := map[string]bool{}
	for _, param := range ep.PathParams {
		existing[param.Name] = true
	}
	for _, name := range pathParamNames(ep.Path) {
		if existing[name] {
			continue
		}
		ep.PathParams = append(ep.PathParams, model.DiscoveryParam{Name: name, In: "path", Required: true, Type: "string"})
	}
}

func extractParams(body string, patterns []*regexp.Regexp, location string) []model.DiscoveryParam {
	seen := map[string]bool{}
	params := []model.DiscoveryParam{}
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(body, -1) {
			if len(match) < 2 {
				continue
			}
			name := strings.TrimSpace(match[1])
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			params = append(params, model.DiscoveryParam{Name: name, In: location, Type: "string"})
		}
	}
	return params
}

func extractStatusCodes(body string) []int {
	codes := []int{}
	for _, match := range statusCodePattern.FindAllStringSubmatch(body, -1) {
		if len(match) >= 2 {
			if code := atoi(match[2]); code >= 100 && code <= 599 {
				codes = append(codes, code)
			}
		}
	}
	for _, match := range statusNamePattern.FindAllStringSubmatch(body, -1) {
		codes = append(codes, statusNameToCode(match[1]))
	}
	return appendUniqueInts(nil, codes)
}

func statusNameToCode(name string) int {
	switch name {
	case "OK":
		return 200
	case "Created":
		return 201
	case "Accepted":
		return 202
	case "NoContent":
		return 204
	case "BadRequest":
		return 400
	case "Unauthorized":
		return 401
	case "Forbidden":
		return 403
	case "NotFound":
		return 404
	case "Conflict":
		return 409
	case "UnprocessableEntity":
		return 422
	case "InternalServerError":
		return 500
	}
	return 0
}
