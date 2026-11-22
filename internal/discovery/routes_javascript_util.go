package discovery

import (
	"path"
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

func extractNestJSRoutes(a *analyzer, f *fileEntry) {
	prefix := ""
	if match := jsNestCtrl.FindStringSubmatch(f.Content); match != nil {
		prefix = match[2]
	}
	controller := ""
	if idx := strings.Index(f.Content, "export class "); idx >= 0 {
		rest := f.Content[idx+len("export class "):]
		if end := strings.IndexAny(rest, " {\n"); end > 0 {
			controller = rest[:end]
		}
	}
	for _, match := range jsNestMethod.FindAllStringSubmatchIndex(f.Content, -1) {
		method := strings.ToUpper(f.Content[match[2]:match[3]])
		routePath := strings.TrimSpace(f.Content[match[6]:match[7]])
		line := lineNumberAt(f.Content, match[0])
		handler := nearestIdentifier(f.Content, match[0])
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefix, routePath), SourceFile: f.Rel, Line: line,
			Handler: handler, Controller: controller, Framework: "NestJS", Confidence: 85, Origin: "static",
		}
		if hasAuthKeyword(nearestDecorators(f.Content, match[0])) {
			ep.AuthRequired = true
			ep.AuthType = "guard"
			ep.AuthNotes = "auth guard decorator detected"
		}
		a.addEndpoint(ep)
	}
}

// extractNextJSRoutes derives the route path from the file location and reads
// exported method handlers.
func extractNextJSRoutes(a *analyzer, f *fileEntry) {
	routePath := nextRoutePath(f.Rel)
	methods := []string{}
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
		if regexp.MustCompile(`(?m)export\s+(async\s+)?function\s+`+method+`\b`).MatchString(f.Content) ||
			regexp.MustCompile(`(?m)export\s+const\s+`+method+`\s*=`).MatchString(f.Content) {
			methods = append(methods, method)
		}
	}
	if len(methods) == 0 {
		methods = []string{"GET"}
	}
	for _, method := range methods {
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: method, Path: routePath, SourceFile: f.Rel, Line: 1,
			Framework: "Next.js", Confidence: 85, Origin: "static",
		})
	}
}

func nextRoutePath(rel string) string {
	normalized := strings.ReplaceAll(rel, "\\", "/")
	normalized = strings.TrimSuffix(normalized, path.Ext(normalized))
	for _, marker := range []string{"/pages/api/", "/src/pages/api/", "/app/api/", "/src/app/api/"} {
		if idx := strings.Index(normalized, marker); idx >= 0 {
			rest := normalized[idx+len(marker):]
			rest = strings.TrimSuffix(rest, "/route")
			rest = strings.TrimSuffix(rest, "/index")
			rest = strings.ReplaceAll(rest, "[", ":")
			rest = strings.ReplaceAll(rest, "]", "")
			return normalizeRoutePath("/" + rest)
		}
	}
	return "/" + normalizeRoutePath(normalized)
}

// callArgsAt returns the arguments of the call whose opening parenthesis is at or
// after start.
func callArgsAt(content string, start int) []string {
	open := strings.IndexByte(content[start:], '(')
	if open < 0 {
		return nil
	}
	open += start
	body, _ := splitBraceBody(content, open)
	if body == "" {
		return nil
	}
	return splitTopLevelArgs(body)
}

func nearestIdentifier(content string, offset int) string {
	lines := strings.Split(content[offset:], "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "async ") {
			trimmed = strings.TrimPrefix(trimmed, "async ")
		}
		if strings.HasPrefix(trimmed, "function ") {
			trimmed = strings.TrimPrefix(trimmed, "function ")
			if idx := strings.IndexAny(trimmed, "(<"); idx > 0 {
				return strings.TrimSpace(trimmed[:idx])
			}
		}
		if idx := strings.Index(trimmed, "("); idx > 0 {
			candidate := strings.TrimSpace(trimmed[:idx])
			if candidate != "" && !strings.ContainsAny(candidate, " :={}") {
				return candidate
			}
		}
	}
	return ""
}

func nearestDecorators(content string, offset int) string {
	start := offset - 400
	if start < 0 {
		start = 0
	}
	return content[start:offset]
}
