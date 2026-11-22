package discovery

import "strings"

// firstQuoted returns the first quoted string literal in a code fragment.
func firstQuoted(fragment string) string {
	for i := 0; i < len(fragment); i++ {
		ch := fragment[i]
		if ch != '"' && ch != '\'' && ch != '`' {
			continue
		}
		quote := ch
		end := i + 1
		for end < len(fragment) {
			if fragment[end] == '\\' {
				end += 2
				continue
			}
			if fragment[end] == quote {
				return fragment[i+1 : end]
			}
			end++
		}
	}
	return ""
}

// splitTopLevelArgs splits a call argument list on commas that are not nested in
// brackets, braces or strings.
func splitTopLevelArgs(raw string) []string {
	args := []string{}
	depth := 0
	inString := byte(0)
	escaped := false
	start := 0
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if inString != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inString {
				inString = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inString = ch
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(raw[start:i]))
				start = i + 1
			}
		}
	}
	if tail := strings.TrimSpace(raw[start:]); tail != "" {
		args = append(args, tail)
	}
	return args
}

// splitBraceBody returns the body between the first "{" and its matching "}".
func splitBraceBody(content string, openIdx int) (string, int) {
	if openIdx < 0 || openIdx >= len(content) || content[openIdx] != '{' {
		return "", -1
	}
	depth := 0
	inString := byte(0)
	escaped := false
	for i := openIdx; i < len(content); i++ {
		ch := content[i]
		if inString != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inString {
				inString = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inString = ch
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[openIdx+1 : i], i
			}
		}
	}
	return content[openIdx+1:], len(content) - 1
}

// lineNumberAt returns the 1-based line number of a byte offset.
func lineNumberAt(content string, offset int) int {
	if offset <= 0 {
		return 1
	}
	if offset > len(content) {
		offset = len(content)
	}
	return strings.Count(content[:offset], "\n") + 1
}

// hasAuthKeyword reports whether text suggests authentication requirements.
func hasAuthKeyword(text string) bool {
	lower := strings.ToLower(text)
	for _, keyword := range []string{
		"auth", "jwt", "bearer", "token", "login_required", "requiresauth",
		"authenticate", "authorize", "permission", "guard", "middleware",
		"apikey", "api_key", "session", "oauth", "cognito", "passport",
	} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}
