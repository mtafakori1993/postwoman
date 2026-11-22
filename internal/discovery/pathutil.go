package discovery

import (
	"regexp"
	"strings"
)

var (
	pathParamPatterns = []*regexp.Regexp{
		regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`),                  // express/gin :id
		regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`),               // chi/openapi {id}
		regexp.MustCompile(`<(?:(?:[a-z]+):)?([A-Za-z_][A-Za-z0-9_]*)>`), // flask <int:id>
		regexp.MustCompile(`\[([A-Za-z_][A-Za-z0-9_]*)\]`),               // next.js [id]
		regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`),             // php/shell ${id}
	}
	queryStringPattern = regexp.MustCompile(`\?[^#]*$`)
)

// normalizeRoutePath canonicalises a route path so paths from different
// frameworks can be merged: braces and angle brackets become :name, duplicate
// slashes collapse and trailing slashes are dropped.
func normalizeRoutePath(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "/"
	}
	path = queryStringPattern.ReplaceAllString(path, "")
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ReplaceAll(path, "//", "/")
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	path = strings.ReplaceAll(path, "<", ":")
	path = strings.ReplaceAll(path, ">", "")
	path = strings.ReplaceAll(path, "[", ":")
	path = strings.ReplaceAll(path, "]", "")
	path = strings.ReplaceAll(path, "${", ":")
	path = strings.ReplaceAll(path, "$", ":")
	path = strings.ReplaceAll(path, "::", ":")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// joinRoutePaths concatenates a group prefix and a route path without producing
// duplicate slashes.
func joinRoutePaths(prefix, path string) string {
	prefix = strings.TrimSpace(prefix)
	path = strings.TrimSpace(path)
	if prefix == "" {
		return normalizeRoutePath(path)
	}
	if path == "" || path == "/" {
		return normalizeRoutePath(prefix)
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return normalizeRoutePath(path)
	}
	return normalizeRoutePath(strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/"))
}

// pathParamNames extracts path parameter names in order.
func pathParamNames(path string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, re := range pathParamPatterns {
		for _, match := range re.FindAllStringSubmatch(path, -1) {
			if len(match) < 2 {
				continue
			}
			name := match[1]
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// routeDisplayName turns a handler name or path into a readable request name.
func routeDisplayName(method, path, handler string) string {
	handler = strings.TrimSpace(handler)
	handler = strings.TrimPrefix(handler, "func ")
	handler = strings.TrimPrefix(handler, "async ")
	if handler != "" {
		if idx := strings.LastIndex(handler, "."); idx >= 0 {
			handler = handler[idx+1:]
		}
		handler = strings.TrimSuffix(handler, "()")
		if handler != "" && handler != "func" {
			return humanizeIdentifier(handler)
		}
	}
	segments := strings.Split(strings.Trim(path, "/"), "/")
	meaningful := []string{}
	for _, segment := range segments {
		if segment == "" || strings.HasPrefix(segment, ":") {
			continue
		}
		meaningful = append(meaningful, humanizeIdentifier(segment))
	}
	if len(meaningful) == 0 {
		return strings.ToUpper(method) + " " + path
	}
	return strings.ToUpper(method) + " " + strings.Join(meaningful, " ")
}

var identifierSplit = regexp.MustCompile(`([a-z0-9])([A-Z])|[_\-]+`)

func humanizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = identifierSplit.ReplaceAllString(value, "$1 $2")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	words := strings.Fields(value)
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
