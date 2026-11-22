package discovery

import (
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

var (
	rbRoute     = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete|options)\s+['"]([^'"]+)['"](.*)$`)
	rbMatch     = regexp.MustCompile(`(?m)^\s*match\s+['"]([^'"]+)['"](.*)$`)
	rbResources = regexp.MustCompile(`(?m)^\s*(resources?|api_resources?)\s+:(\w+)`)
	rbNamespace = regexp.MustCompile(`(?m)^\s*namespace\s+:(\w+)`)
	rbScope     = regexp.MustCompile(`(?m)^\s*scope\s+['"]([^'"]+)['"]`)
	rbRoot      = regexp.MustCompile(`(?m)^\s*root\s+(?:to:\s*)?['"]([^'"]+)['"]`)
	rbVia       = regexp.MustCompile(`(?i)via:\s*\[([^\]]*)\]`)
)

// extractRubyRoutes understands Rails and Sinatra routing.
func extractRubyRoutes(a *analyzer, f *fileEntry) {
	prefix := ""
	if match := rbNamespace.FindStringSubmatch(f.Content); match != nil {
		prefix = "/" + match[1]
	} else if match := rbScope.FindStringSubmatch(f.Content); match != nil {
		prefix = normalizeRoutePath(match[1])
	}
	for _, match := range rbRoute.FindAllStringSubmatchIndex(f.Content, -1) {
		method := strings.ToUpper(f.Content[match[2]:match[3]])
		routePath := f.Content[match[4]:match[5]]
		rest := f.Content[match[6]:match[7]]
		line := lineNumberAt(f.Content, match[0])
		handler := firstQuoted(rest)
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Handler: handler, Framework: "Rails", Confidence: 82, Origin: "static",
		}
		if hasAuthKeyword(rest) {
			ep.AuthRequired = true
			ep.AuthType = "before_action"
			ep.AuthNotes = "authentication filter referenced in route"
		}
		a.addEndpoint(ep)
	}
	for _, match := range rbMatch.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[4]:match[5]]
		rest := f.Content[match[6]:match[7]]
		line := lineNumberAt(f.Content, match[0])
		methods := []string{"GET"}
		if viaMatch := rbVia.FindStringSubmatch(rest); viaMatch != nil {
			parsed := []string{}
			for _, method := range strings.Split(viaMatch[1], ",") {
				method = strings.ToUpper(strings.Trim(strings.TrimSpace(method), ": '\""))
				if method != "" {
					parsed = append(parsed, method)
				}
			}
			if len(parsed) > 0 {
				methods = parsed
			}
		}
		for _, method := range methods {
			a.addEndpoint(model.DiscoveredEndpoint{
				Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
				Framework: "Rails", Confidence: 70, Origin: "static",
			})
		}
	}
	for _, match := range rbResources.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[4]:match[5]]
		line := lineNumberAt(f.Content, match[0])
		for _, pair := range []struct{ method, suffix string }{
			{"GET", ""}, {"POST", ""}, {"GET", "/:id"},
			{"PUT", "/:id"}, {"PATCH", "/:id"}, {"DELETE", "/:id"},
		} {
			a.addEndpoint(model.DiscoveredEndpoint{
				Method: pair.method, Path: joinRoutePaths(prefix, routePath+pair.suffix), SourceFile: f.Rel, Line: line,
				Framework: "Rails", Confidence: 70, Origin: "static",
				Notes: []string{"resource route"},
			})
		}
	}
	for _, match := range rbRoot.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[2]:match[3]]
		line := lineNumberAt(f.Content, match[0])
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: "GET", Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Framework: "Rails", Confidence: 65, Origin: "static", Notes: []string{"root route"},
		})
	}
}
