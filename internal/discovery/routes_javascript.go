package discovery

import (
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

var (
	jsMethodCall = regexp.MustCompile(`(?m)\b([A-Za-z_$][\w$]*)\s*\.\s*(get|post|put|patch|delete|head|options|all)\s*\(\s*(['"` + "`" + `])([^'"` + "`" + `]+)['"` + "`" + `]`)
	jsUsePrefix  = regexp.MustCompile(`(?m)\b([A-Za-z_$][\w$]*)\s*\.\s*use\s*\(\s*(['"` + "`" + `])([^'"` + "`" + `]+)['"` + "`" + `]`)
	jsRouteChain = regexp.MustCompile(`(?m)\.\s*route\s*\(\s*(['"` + "`" + `])([^'"` + "`" + `]+)['"` + "`" + `]\s*\)\s*\.\s*(get|post|put|patch|delete|head|options|all)\s*\(`)
	jsNestCtrl   = regexp.MustCompile(`(?m)@Controller\s*\(\s*(['"` + "`" + `])([^'"` + "`" + `]*)['"` + "`" + `]`)
	jsNestMethod = regexp.MustCompile(`(?m)@(Get|Post|Put|Patch|Delete|Head|Options|All)\s*\(\s*(['"` + "`" + `]?)([^'"` + "`" + `)]*)['"` + "`" + `]?`)
	jsFastifyObj = regexp.MustCompile(`(?m)\broute\s*\(\s*\{([^}]*)\}`)
	jsFastifyM   = regexp.MustCompile(`(?i)method\s*:\s*(['"` + "`" + `])([A-Za-z,]+)['"` + "`" + `]`)
	jsFastifyURL = regexp.MustCompile(`(?i)url\s*:\s*(['"` + "`" + `])([^'"` + "`" + `]+)['"` + "`" + `]`)
)

var jsMethods = map[string]string{
	"get": "GET", "post": "POST", "put": "PUT", "patch": "PATCH",
	"delete": "DELETE", "head": "HEAD", "options": "OPTIONS", "all": "ANY",
}

// extractJSRoutes understands Express, NestJS, Fastify, Koa, Hono and Next.js
// file-based routes.
func extractJSRoutes(a *analyzer, f *fileEntry) {
	lower := strings.ToLower(f.Rel)
	if strings.Contains(lower, "/pages/api/") || strings.Contains(lower, "/app/api/") || strings.Contains(lower, "/src/pages/api/") {
		extractNextJSRoutes(a, f)
		return
	}
	if strings.Contains(f.Content, "@Controller(") {
		extractNestJSRoutes(a, f)
		return
	}
	prefixes := map[string]string{}
	for _, match := range jsUsePrefix.FindAllStringSubmatch(f.Content, -1) {
		prefixes[match[1]] = normalizeRoutePath(match[3])
	}
	for _, match := range jsMethodCall.FindAllStringSubmatchIndex(f.Content, -1) {
		receiver := f.Content[match[2]:match[3]]
		method := jsMethods[strings.ToLower(f.Content[match[4]:match[5]])]
		routePath := f.Content[match[8]:match[9]]
		line := lineNumberAt(f.Content, match[0])
		args := callArgsAt(f.Content, match[0])
		handler := ""
		auth := false
		for _, arg := range args {
			trimmed := strings.TrimSpace(arg)
			if trimmed == "" || strings.HasPrefix(trimmed, "'") || strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "`") {
				continue
			}
			if hasAuthKeyword(trimmed) {
				auth = true
				continue
			}
			if handler == "" {
				handler = trimmed
			}
		}
		handlerName, handlerFile, _ := a.resolveHandler(handler, f.Rel)
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefixes[receiver], routePath),
			SourceFile: f.Rel, Line: line, Handler: handlerName, HandlerFile: handlerFile,
			Framework: "JavaScript", Confidence: 78, Origin: "static",
		}
		if auth {
			ep.AuthRequired = true
			ep.AuthType = "middleware"
			ep.AuthNotes = "middleware detected in route registration"
		}
		a.addEndpoint(ep)
	}
	for _, match := range jsRouteChain.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[4]:match[5]]
		method := jsMethods[strings.ToLower(f.Content[match[6]:match[7]])]
		line := lineNumberAt(f.Content, match[0])
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: method, Path: normalizeRoutePath(routePath), SourceFile: f.Rel, Line: line,
			Framework: "JavaScript", Confidence: 70, Origin: "static",
		})
	}
	for _, match := range jsFastifyObj.FindAllStringSubmatch(f.Content, -1) {
		body := match[1]
		methodMatch := jsFastifyM.FindStringSubmatch(body)
		urlMatch := jsFastifyURL.FindStringSubmatch(body)
		if methodMatch == nil || urlMatch == nil {
			continue
		}
		for _, method := range strings.Split(strings.ToUpper(methodMatch[2]), ",") {
			method = strings.TrimSpace(method)
			if method == "" {
				continue
			}
			a.addEndpoint(model.DiscoveredEndpoint{
				Method: method, Path: normalizeRoutePath(urlMatch[2]), SourceFile: f.Rel,
				Framework: "Fastify", Confidence: 80, Origin: "static",
			})
		}
	}
}
