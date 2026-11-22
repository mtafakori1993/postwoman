package discovery

import (
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

var (
	jClassMapping = regexp.MustCompile(`(?m)@RequestMapping\s*\(\s*(?:value\s*=\s*)?['"]([^'"]*)['"][^)]*\)`)
	jMethodMap    = regexp.MustCompile(`(?m)@(Get|Post|Put|Patch|Delete)Mapping\s*\(\s*(?:value\s*=\s*)?['"]?([^'")\s]*)['"]?[^)]*\)`)
	jRequestMap   = regexp.MustCompile(`(?m)@RequestMapping\s*\(\s*(?:value\s*=\s*)?['"]([^'"]*)['"][^)]*?method\s*=\s*RequestMethod\.(\w+)`)
	jClassDecl    = regexp.MustCompile(`(?m)\bclass\s+(\w+)`)
	jAuthAnnot    = []string{"@PreAuthorize", "@Secured", "@RolesAllowed", "@RequiresAuthentication", "@Authenticated"}
)

var javaMethodNames = map[string]string{
	"Get": "GET", "Post": "POST", "Put": "PUT", "Patch": "PATCH", "Delete": "DELETE",
}

// extractJavaRoutes understands Spring Boot (and Kotlin) annotation routing.
func extractJavaRoutes(a *analyzer, f *fileEntry) {
	prefix := ""
	if match := jClassMapping.FindStringSubmatch(f.Content); match != nil {
		prefix = match[1]
	}
	controller := ""
	if match := jClassDecl.FindStringSubmatch(f.Content); match != nil {
		controller = match[1]
	}
	for _, match := range jMethodMap.FindAllStringSubmatchIndex(f.Content, -1) {
		method := javaMethodNames[f.Content[match[2]:match[3]]]
		routePath := f.Content[match[4]:match[5]]
		line := lineNumberAt(f.Content, match[0])
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Controller: controller, Framework: "Spring Boot", Confidence: 85, Origin: "static",
		}
		applyJavaAuth(&ep, f.Content, match[0])
		a.addEndpoint(ep)
	}
	for _, match := range jRequestMap.FindAllStringSubmatchIndex(f.Content, -1) {
		method := strings.ToUpper(f.Content[match[4]:match[5]])
		routePath := f.Content[match[2]:match[3]]
		line := lineNumberAt(f.Content, match[0])
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Controller: controller, Framework: "Spring Boot", Confidence: 80, Origin: "static",
		}
		applyJavaAuth(&ep, f.Content, match[0])
		a.addEndpoint(ep)
	}
}

func applyJavaAuth(ep *model.DiscoveredEndpoint, content string, offset int) {
	start := offset - 300
	if start < 0 {
		start = 0
	}
	window := content[start:offset]
	for _, annotation := range jAuthAnnot {
		if strings.Contains(window, annotation) {
			ep.AuthRequired = true
			ep.AuthType = "annotation"
			ep.AuthNotes = "authorization annotation: " + annotation
			return
		}
	}
}
