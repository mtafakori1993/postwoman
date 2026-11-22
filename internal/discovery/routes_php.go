package discovery

import (
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

var (
	phpRouteCall   = regexp.MustCompile(`(?m)Route::(get|post|put|patch|delete|options|any)\s*\(\s*(['"])([^'"]+)['"]\s*,\s*([^;]+)`)
	phpRouteMatch  = regexp.MustCompile(`(?m)Route::match\s*\(\s*\[([^\]]*)\]\s*,\s*(['"])([^'"]+)['"]\s*,\s*([^;]+)`)
	phpResource    = regexp.MustCompile(`(?m)Route::(?:apiResource|resource)\s*\(\s*(['"])([^'"]+)['"]\s*,\s*([A-Za-z_\\]+)`)
	phpPrefix      = regexp.MustCompile(`(?m)Route::(?:prefix|group)\s*\(\s*(?:\[\s*['"]prefix['"]\s*=>\s*)?(['"])([^'"]+)['"]`)
	phpSymfonyAttr = regexp.MustCompile(`(?m)#\[Route\s*\(\s*(['"])([^'"]+)['"]([^)]*)\)\]`)
	phpSymfonyMeth = regexp.MustCompile(`(?i)methods\s*:\s*\[([^\]]*)\]`)
	phpController  = regexp.MustCompile(`([A-Za-z_\\][\w\\]*)\s*(?:::|@)\s*['"]?(\w+)`)
)

// extractPHPRoutes understands Laravel and Symfony routing.
func extractPHPRoutes(a *analyzer, f *fileEntry) {
	prefix := ""
	if match := phpPrefix.FindStringSubmatch(f.Content); match != nil {
		prefix = normalizeRoutePath(match[2])
	}
	for _, match := range phpRouteCall.FindAllStringSubmatchIndex(f.Content, -1) {
		method := strings.ToUpper(f.Content[match[2]:match[3]])
		routePath := f.Content[match[6]:match[7]]
		handlerRaw := f.Content[match[8]:match[9]]
		line := lineNumberAt(f.Content, match[0])
		handler := ""
		if ctrl := phpController.FindStringSubmatch(handlerRaw); ctrl != nil {
			handler = ctrl[1] + "." + ctrl[2]
		}
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Handler: handler, Framework: "Laravel", Confidence: 82, Origin: "static",
		}
		applyPHPAuth(&ep, handlerRaw)
		a.addEndpoint(ep)
	}
	for _, match := range phpRouteMatch.FindAllStringSubmatchIndex(f.Content, -1) {
		methodsRaw := f.Content[match[2]:match[3]]
		routePath := f.Content[match[6]:match[7]]
		handlerRaw := f.Content[match[8]:match[9]]
		line := lineNumberAt(f.Content, match[0])
		handler := ""
		if ctrl := phpController.FindStringSubmatch(handlerRaw); ctrl != nil {
			handler = ctrl[1] + "." + ctrl[2]
		}
		for _, method := range strings.Split(methodsRaw, ",") {
			method = strings.ToUpper(strings.Trim(strings.TrimSpace(method), "'\""))
			if method == "" {
				continue
			}
			a.addEndpoint(model.DiscoveredEndpoint{
				Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
				Handler: handler, Framework: "Laravel", Confidence: 80, Origin: "static",
			})
		}
	}
	for _, match := range phpResource.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[4]:match[5]]
		controller := f.Content[match[6]:match[7]]
		line := lineNumberAt(f.Content, match[0])
		for _, pair := range []struct{ method, action string }{
			{"GET", "index"}, {"POST", "store"}, {"GET", "show"},
			{"PUT", "update"}, {"PATCH", "update"}, {"DELETE", "destroy"},
		} {
			path := routePath
			if pair.action == "show" || pair.action == "update" || pair.action == "destroy" {
				path = strings.TrimRight(routePath, "/") + "/:id"
			}
			a.addEndpoint(model.DiscoveredEndpoint{
				Method: pair.method, Path: normalizeRoutePath(path), SourceFile: f.Rel, Line: line,
				Handler: controller + "." + pair.action, Controller: controller,
				Framework: "Laravel", Confidence: 75, Origin: "static",
				Notes: []string{"resource route: " + pair.action},
			})
		}
	}
	for _, match := range phpSymfonyAttr.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[4]:match[5]]
		options := f.Content[match[6]:match[7]]
		line := lineNumberAt(f.Content, match[0])
		methods := []string{"GET"}
		if methodMatch := phpSymfonyMeth.FindStringSubmatch(options); methodMatch != nil {
			parsed := []string{}
			for _, method := range strings.Split(methodMatch[1], ",") {
				method = strings.ToUpper(strings.Trim(strings.TrimSpace(method), "'\""))
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
				Framework: "Symfony", Confidence: 80, Origin: "static",
			})
		}
	}
}

func applyPHPAuth(ep *model.DiscoveredEndpoint, handlerRaw string) {
	if hasAuthKeyword(handlerRaw) {
		ep.AuthRequired = true
		ep.AuthType = "middleware"
		ep.AuthNotes = "middleware/auth detected in route definition"
	}
}
