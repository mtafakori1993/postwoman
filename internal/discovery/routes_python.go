package discovery

import (
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

var (
	pyFlaskRoute  = regexp.MustCompile(`(?m)@\s*([A-Za-z_][\w.]*)\s*\.\s*route\s*\(\s*(['"])([^'"]*)['"]([^)]*)\)`)
	pyFastRoute   = regexp.MustCompile(`(?m)@\s*([A-Za-z_][\w.]*)\s*\.\s*(get|post|put|patch|delete|head|options)\s*\(\s*(['"])([^'"]*)['"]([^)]*)\)`)
	pyPrefix      = regexp.MustCompile(`(?m)([A-Za-z_]\w*)\s*=\s*(?:Blueprint|APIRouter|Flask)\s*\([^)]*?(?:url_prefix|prefix)\s*=\s*(['"])([^'"]*)['"]`)
	pyDjangoPath  = regexp.MustCompile(`(?m)\b(?:path|re_path|url)\s*\(\s*(?:r?(['"])([^'"]*)['"])\s*,\s*([A-Za-z_][\w.]*)`)
	pyRegister    = regexp.MustCompile(`(?m)\bregister\s*\(\s*(?:r?(['"])([^'"]*)['"])\s*,`)
	pyMethodsList = regexp.MustCompile(`(?i)methods\s*=\s*\[([^\]]*)\]`)
	pyStatus      = regexp.MustCompile(`(?i)\bstatus_code\s*=\s*(\d{3})`)
	pyAuthMarkers = []string{"login_required", "jwt_required", "requires_auth", "permission_classes",
		"get_current_user", "auth_required", "requires_permission", "Security(", "authenticate"}
)

// extractPythonRoutes understands Flask, FastAPI, Django and DRF.
func extractPythonRoutes(a *analyzer, f *fileEntry) {
	prefixes := map[string]string{}
	for _, match := range pyPrefix.FindAllStringSubmatch(f.Content, -1) {
		prefixes[match[1]] = normalizeRoutePath(match[3])
	}
	for _, match := range pyFlaskRoute.FindAllStringSubmatchIndex(f.Content, -1) {
		receiver := f.Content[match[2]:match[3]]
		routePath := f.Content[match[6]:match[7]]
		options := f.Content[match[8]:match[9]]
		line := lineNumberAt(f.Content, match[0])
		methods := []string{"GET"}
		if methodsMatch := pyMethodsList.FindStringSubmatch(options); methodsMatch != nil {
			parsed := []string{}
			for _, method := range strings.Split(methodsMatch[1], ",") {
				method = strings.ToUpper(strings.Trim(strings.TrimSpace(method), "'\""))
				if method != "" {
					parsed = append(parsed, method)
				}
			}
			if len(parsed) > 0 {
				methods = parsed
			}
		}
		handler := pythonHandlerAfter(f.Content, match[1])
		ep := model.DiscoveredEndpoint{
			Method: methods[0], Path: joinRoutePaths(prefixes[receiver], routePath),
			SourceFile: f.Rel, Line: line, Handler: handler, Framework: "Flask", Confidence: 82, Origin: "static",
		}
		applyPythonAuth(&ep, f.Content, match[0])
		applyPythonStatus(&ep, f.Content, match[0])
		a.addEndpoint(ep)
		for _, method := range methods[1:] {
			extra := ep
			extra.Method = method
			a.addEndpoint(extra)
		}
	}
	for _, match := range pyFastRoute.FindAllStringSubmatchIndex(f.Content, -1) {
		receiver := f.Content[match[2]:match[3]]
		method := strings.ToUpper(f.Content[match[4]:match[5]])
		routePath := f.Content[match[8]:match[9]]
		line := lineNumberAt(f.Content, match[0])
		handler := pythonHandlerAfter(f.Content, match[1])
		ep := model.DiscoveredEndpoint{
			Method: method, Path: joinRoutePaths(prefixes[receiver], routePath),
			SourceFile: f.Rel, Line: line, Handler: handler, Framework: "FastAPI", Confidence: 85, Origin: "static",
		}
		applyPythonAuth(&ep, f.Content, match[0])
		applyPythonStatus(&ep, f.Content, match[0])
		a.addEndpoint(ep)
	}
	for _, match := range pyDjangoPath.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[4]:match[5]]
		handler := f.Content[match[6]:match[7]]
		line := lineNumberAt(f.Content, match[0])
		a.addEndpoint(model.DiscoveredEndpoint{
			Method: "GET", Path: normalizeRoutePath(routePath), SourceFile: f.Rel, Line: line,
			Handler: handler, Framework: "Django", Confidence: 60, Origin: "static",
			Notes: []string{"method not specified in URL configuration"},
		})
	}
	for _, match := range pyRegister.FindAllStringSubmatchIndex(f.Content, -1) {
		routePath := f.Content[match[4]:match[5]]
		line := lineNumberAt(f.Content, match[0])
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			a.addEndpoint(model.DiscoveredEndpoint{
				Method: method, Path: normalizeRoutePath(routePath), SourceFile: f.Rel, Line: line,
				Framework: "Django REST", Confidence: 55, Origin: "static",
				Notes: []string{"ViewSet registration: actions inferred"},
			})
		}
	}
}

func pythonHandlerAfter(content string, offset int) string {
	rest := content[offset:]
	lines := strings.Split(rest, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") {
			trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "async "), "def ")
			if idx := strings.Index(trimmed, "("); idx > 0 {
				return strings.TrimSpace(trimmed[:idx])
			}
		}
	}
	return ""
}

func applyPythonAuth(ep *model.DiscoveredEndpoint, content string, offset int) {
	start := offset - 400
	if start < 0 {
		start = 0
	}
	end := offset + 600
	if end > len(content) {
		end = len(content)
	}
	window := content[start:end]
	for _, marker := range pyAuthMarkers {
		if strings.Contains(window, marker) {
			ep.AuthRequired = true
			ep.AuthType = "decorator"
			ep.AuthNotes = "auth decorator/dependency: " + marker
			ep.Confidence = minInt(ep.Confidence+5, 95)
			return
		}
	}
}

func applyPythonStatus(ep *model.DiscoveredEndpoint, content string, offset int) {
	end := offset + 2000
	if end > len(content) {
		end = len(content)
	}
	if match := pyStatus.FindStringSubmatch(content[offset:end]); match != nil {
		ep.StatusCodes = append(ep.StatusCodes, atoi(match[1]))
	}
}

func atoi(value string) int {
	result := 0
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < '0' || ch > '9' {
			return result
		}
		result = result*10 + int(ch-'0')
	}
	return result
}
