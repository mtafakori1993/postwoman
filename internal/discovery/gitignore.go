package discovery

import (
	"regexp"
	"strings"
)

// ignoreRule is one compiled .gitignore line.
type ignoreRule struct {
	base    string // directory the .gitignore lives in, relative to the project root
	re      *regexp.Regexp
	negate  bool
	dirOnly bool
	raw     string
}

// ignoreSet evaluates rules in order; the last matching rule wins, mirroring
// git's behaviour.
type ignoreSet struct {
	rules []ignoreRule
}

func (s *ignoreSet) addAll(content, base string) {
	s.rules = append(s.rules, parseGitignore(content, base)...)
}

func (s *ignoreSet) empty() bool { return len(s.rules) == 0 }

// ignored reports whether a project-relative path is ignored. isDir selects
// whether directory-only rules apply.
func (s *ignoreSet) ignored(rel string, isDir bool) bool {
	rel = strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "./")
	result := false
	for _, rule := range s.rules {
		subject := rel
		if rule.base != "" {
			if subject == rule.base {
				subject = ""
			} else if strings.HasPrefix(subject, rule.base+"/") {
				subject = strings.TrimPrefix(subject, rule.base+"/")
			} else {
				continue
			}
		}
		if subject == "" {
			continue
		}
		if rule.dirOnly && !isDir {
			continue
		}
		if rule.re.MatchString(subject) {
			result = !rule.negate
		}
	}
	return result
}

func parseGitignore(content, base string) []ignoreRule {
	rules := []ignoreRule{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		negate := false
		if strings.HasPrefix(trimmed, "!") {
			negate = true
			trimmed = strings.TrimPrefix(trimmed, "!")
		}
		trimmed = strings.TrimPrefix(trimmed, "\\")
		dirOnly := strings.HasSuffix(trimmed, "/")
		trimmed = strings.TrimSuffix(trimmed, "/")
		if trimmed == "" {
			continue
		}
		anchored := strings.HasPrefix(trimmed, "/") || strings.Contains(strings.TrimSuffix(trimmed, "/"), "/")
		trimmed = strings.TrimPrefix(trimmed, "/")
		expr := globToRegexp(trimmed)
		prefix := `^(?:.*/)?`
		if anchored {
			prefix = `^`
		}
		suffix := `(?:$|/.*$)`
		if dirOnly {
			suffix = `(?:$|/.*$)`
		}
		re, err := regexp.Compile(prefix + expr + suffix)
		if err != nil {
			continue
		}
		rules = append(rules, ignoreRule{base: base, re: re, negate: negate, dirOnly: dirOnly, raw: line})
	}
	return rules
}

func globToRegexp(pattern string) string {
	var out strings.Builder
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				// "**/" matches zero or more leading directories.
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					out.WriteString(`(?:.*/)?`)
				} else {
					out.WriteString(`.*`)
				}
				continue
			}
			out.WriteString(`[^/]*`)
		case '?':
			out.WriteString(`[^/]`)
		default:
			out.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	return out.String()
}
