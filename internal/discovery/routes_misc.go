package discovery

import (
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

var (
	csClassRoute = regexp.MustCompile(`(?m)\[Route\s*\(\s*"([^"]*)"\s*\)\]`)
	csHttpMethod = regexp.MustCompile(`(?m)\[Http(Get|Post|Put|Patch|Delete)\s*(?:\(\s*"([^"]*)"\s*\))?\]`)
	csClassDecl  = regexp.MustCompile(`(?m)\bclass\s+(\w+)`)
	csAuthorize  = regexp.MustCompile(`(?m)\[Authorize`)
	rsDecorator  = regexp.MustCompile(`(?m)#\[(get|post|put|patch|delete|head|options)\s*\(\s*"([^"]*)"\s*\)\]`)
	rsRouteAttr  = regexp.MustCompile(`(?m)#\[route\s*\(\s*"([^"]*)"\s*,\s*method\s*=\s*"(\w+)"`)
	rsAxumRoute  = regexp.MustCompile(`(?m)\.route\s*\(\s*"([^"]*)"\s*,\s*(get|post|put|patch|delete)\s*\(`)
	exRoute      = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete|options)\s+"([^"]+)"\s*,\s*([A-Za-z_]\w*)`)
	exResources  = regexp.MustCompile(`(?m)^\s*resources?\s+"([^"]+)"\s*,\s*([A-Za-z_]\w*)`)
	exScope      = regexp.MustCompile(`(?m)^\s*scope\s+"([^"]+)"`)
)

// extractCSharpRoutes understands ASP.NET Core attribute routing.
func extractCSharpRoutes(a *analyzer, f *fileEntry) {
	prefix := ""
	if match := csClassRoute.FindStringSubmatch(f.Content); match != nil {
		prefix = match[1]
	}
	controller := ""
	if match := csClassDecl.FindStringSubmatch(f.Content); match != nil {
		controller = strings.TrimSuffix(match[1], "Controller")
	}
	if controller != "" {
		prefix = strings.ReplaceAll(prefix, "[controller]", strings.ToLower(controller))
	}
	for _, match := range csHttpMethod.FindAllStringSubmatchIndex(f.Content, -1) {
		method := strings.ToUpper(f.Content[match[2]:match[3]])
		routePath := ""
		if match[4] >= 0 && match[5] > match[4] {
			routePath = f.Content[match[4]:match[5]]
		}
		line := lineNumberAt(f.Content, match[0])
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Controller: controller, Framework: "ASP.NET Core", Confidence: 85, Origin: "static",
		}
		if csAuthorize.MatchString(f.Content) {
			ep.AuthRequired = true
			ep.AuthType = "attribute"
			ep.AuthNotes = "[Authorize] attribute detected"
		}
		a.addEndpoint(ep)
	}
}

// extractRustRoutes understands Actix Web, Rocket and Axum.
func extractRustRoutes(a *analyzer, f *fileEntry) {
	for _, match := range rsDecorator.FindAllStringSubmatchIndex(f.Content, -1) {
		method := strings.ToUpper(f.Content[match[2]:match[3]])
		routePath := f.Content[match[4]:match[5]]
		line := lineNumberAt(f.Content, match[0])
		handler := nearestRustFn(f.Content, match[1])
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: method, Path: normalizeRoutePath(routePath), SourceFile: f.Rel, Line: line,
			Handler: handler, Framework: "Actix Web", Confidence: 82, Origin: "static",
		})
	}
	for _, match := range rsRouteAttr.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[2]:match[3]]
		method := strings.ToUpper(f.Content[match[4]:match[5]])
		line := lineNumberAt(f.Content, match[0])
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: method, Path: normalizeRoutePath(routePath), SourceFile: f.Rel, Line: line,
			Framework: "Actix Web", Confidence: 80, Origin: "static",
		})
	}
	for _, match := range rsAxumRoute.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[2]:match[3]]
		method := strings.ToUpper(f.Content[match[4]:match[5]])
		line := lineNumberAt(f.Content, match[0])
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: method, Path: normalizeRoutePath(routePath), SourceFile: f.Rel, Line: line,
			Framework: "Axum", Confidence: 80, Origin: "static",
		})
	}
}

func nearestRustFn(content string, offset int) string {
	lines := strings.Split(content[offset:], "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "pub async fn ") || strings.HasPrefix(trimmed, "async fn ") || strings.HasPrefix(trimmed, "pub fn ") || strings.HasPrefix(trimmed, "fn ") {
			trimmed = strings.TrimPrefix(trimmed, "pub ")
			trimmed = strings.TrimPrefix(trimmed, "async ")
			trimmed = strings.TrimPrefix(trimmed, "fn ")
			if idx := strings.Index(trimmed, "("); idx > 0 {
				return strings.TrimSpace(trimmed[:idx])
			}
		}
	}
	return ""
}

// extractElixirRoutes understands Phoenix routing.
func extractElixirRoutes(a *analyzer, f *fileEntry) {
	prefix := ""
	if match := exScope.FindStringSubmatch(f.Content); match != nil {
		prefix = normalizeRoutePath(match[1])
	}
	for _, match := range exRoute.FindAllStringSubmatchIndex(f.Content, -1) {
		method := strings.ToUpper(f.Content[match[2]:match[3]])
		routePath := f.Content[match[4]:match[5]]
		controller := f.Content[match[6]:match[7]]
		line := lineNumberAt(f.Content, match[0])
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Controller: controller, Framework: "Phoenix", Confidence: 82, Origin: "static",
		})
	}
	for _, match := range exResources.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[2]:match[3]]
		controller := f.Content[match[4]:match[5]]
		line := lineNumberAt(f.Content, match[0])
		for _, pair := range []struct{ method, suffix string }{
			{"GET", ""}, {"POST", ""}, {"GET", "/:id"},
			{"PUT", "/:id"}, {"PATCH", "/:id"}, {"DELETE", "/:id"},
		} {
			a.addEndpoint(model.DiscoveredEndpoint{
				Method: pair.method, Path: joinRoutePaths(prefix, routePath+pair.suffix), SourceFile: f.Rel, Line: line,
				Controller: controller, Framework: "Phoenix", Confidence: 70, Origin: "static",
				Notes: []string{"resource route"},
			})
		}
	}
}
