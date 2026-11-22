package discovery

import (
	"path"
	"strings"

	"Postwoman/internal/model"
)

func declaredSymbols(f *fileEntry) []string {
	symbols := []string{}
	switch f.Language {
	case "go":
		for _, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "func ") {
				continue
			}
			rest := strings.TrimPrefix(trimmed, "func ")
			if strings.HasPrefix(rest, "(") {
				if close := strings.Index(rest, ")"); close >= 0 {
					rest = strings.TrimSpace(rest[close+1:])
				}
			}
			if idx := strings.Index(rest, "("); idx > 0 {
				symbols = append(symbols, strings.TrimSpace(rest[:idx]))
			}
		}
	case "javascript", "typescript":
		for _, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			for _, prefix := range []string{"export default function ", "export async function ", "export function ", "async function ", "function "} {
				if strings.HasPrefix(trimmed, prefix) {
					rest := strings.TrimPrefix(trimmed, prefix)
					if idx := strings.Index(rest, "("); idx > 0 {
						symbols = append(symbols, strings.TrimSpace(rest[:idx]))
					}
				}
			}
			if strings.Contains(trimmed, "=>") {
				left := strings.SplitN(trimmed, "=", 2)[0]
				left = strings.TrimSpace(strings.TrimPrefix(left, "export "))
				left = strings.TrimPrefix(left, "const ")
				left = strings.TrimPrefix(left, "let ")
				left = strings.TrimPrefix(left, "var ")
				if left != "" && !strings.ContainsAny(left, " ()[]{}") {
					symbols = append(symbols, left)
				}
			}
		}
	case "python":
		for _, line := range f.Lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "def ") && !strings.HasPrefix(trimmed, "async def ") {
				continue
			}
			rest := strings.TrimPrefix(strings.TrimPrefix(trimmed, "async "), "def ")
			if idx := strings.Index(rest, "("); idx > 0 {
				symbols = append(symbols, strings.TrimSpace(rest[:idx]))
			}
		}
	}
	return symbols
}

func mergeEndpoint(existing *model.DiscoveredEndpoint, incoming model.DiscoveredEndpoint) {
	if incoming.Confidence > existing.Confidence {
		existing.Confidence = incoming.Confidence
	}
	if incoming.Origin == "openapi" {
		existing.Origin = "openapi"
	}
	if existing.Description == "" {
		existing.Description = incoming.Description
	}
	if existing.Handler == "" {
		existing.Handler = incoming.Handler
	}
	if existing.HandlerFile == "" {
		existing.HandlerFile = incoming.HandlerFile
	}
	if existing.Controller == "" {
		existing.Controller = incoming.Controller
	}
	if existing.SourceFile == "" {
		existing.SourceFile = incoming.SourceFile
	}
	if existing.Line == 0 {
		existing.Line = incoming.Line
	}
	if existing.ContentType == "" {
		existing.ContentType = incoming.ContentType
	}
	if existing.RequestBody == "" {
		existing.RequestBody = incoming.RequestBody
	}
	if !existing.AuthRequired && incoming.AuthRequired {
		existing.AuthRequired = true
		existing.AuthType = incoming.AuthType
		existing.AuthNotes = incoming.AuthNotes
	}
	existing.Headers = appendUniqueParams(existing.Headers, incoming.Headers)
	existing.PathParams = appendUniqueParams(existing.PathParams, incoming.PathParams)
	existing.QueryParams = appendUniqueParams(existing.QueryParams, incoming.QueryParams)
	existing.Tags = appendUniqueStrings(existing.Tags, incoming.Tags)
	existing.Notes = appendUniqueStrings(existing.Notes, incoming.Notes)
	existing.Warnings = appendUniqueStrings(existing.Warnings, incoming.Warnings)
	existing.Related = appendUniqueStrings(existing.Related, incoming.Related)
	existing.StatusCodes = appendUniqueInts(existing.StatusCodes, incoming.StatusCodes)
	if len(existing.RequestSchema) == 0 {
		existing.RequestSchema = incoming.RequestSchema
	}
	if len(existing.ResponseSchema) == 0 {
		existing.ResponseSchema = incoming.ResponseSchema
	}
}

func appendUniqueParams(base []model.DiscoveryParam, extra []model.DiscoveryParam) []model.DiscoveryParam {
	seen := map[string]bool{}
	for _, p := range base {
		seen[p.In+"|"+p.Name] = true
	}
	for _, p := range extra {
		if p.Name == "" || seen[p.In+"|"+p.Name] {
			continue
		}
		seen[p.In+"|"+p.Name] = true
		base = append(base, p)
	}
	return base
}

func appendUniqueStrings(base []string, extra []string) []string {
	seen := map[string]bool{}
	for _, s := range base {
		seen[s] = true
	}
	for _, s := range extra {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		base = append(base, s)
	}
	return base
}

func appendUniqueInts(base []int, extra []int) []int {
	seen := map[int]bool{}
	for _, n := range base {
		seen[n] = true
	}
	for _, n := range extra {
		if seen[n] {
			continue
		}
		seen[n] = true
		base = append(base, n)
	}
	return base
}

// collectSnippets captures a small, local source excerpt per endpoint so AI
// enrichment can reason about the handler without sending the whole file.
func (a *analyzer) collectSnippets() {
	for _, ep := range a.endpoints {
		if ep.HandlerSnippet != "" {
			continue
		}
		snippet := ""
		if ep.Handler != "" {
			snippet = a.handlerBody(ep.Handler)
		}
		if snippet == "" && ep.SourceFile != "" {
			if file := a.byRel[ep.SourceFile]; file != nil && ep.Line > 0 {
				start := ep.Line - 25
				if start < 1 {
					start = 1
				}
				end := ep.Line + 40
				if end > len(file.Lines) {
					end = len(file.Lines)
				}
				snippet = strings.Join(file.Lines[start-1:end], "\n")
			}
		}
		if len(snippet) > 3500 {
			snippet = snippet[:3500]
		}
		ep.HandlerSnippet = snippet
	}
}

// folderForEndpoint derives a logical folder from tags, controller or path.
func folderForEndpoint(ep model.DiscoveredEndpoint) string {
	if len(ep.Tags) > 0 {
		return humanizeIdentifier(ep.Tags[0])
	}
	if ep.Controller != "" {
		name := ep.Controller
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		name = strings.TrimSuffix(name, "Controller")
		name = strings.TrimSuffix(name, "Handler")
		name = strings.TrimSuffix(name, "Routes")
		name = strings.TrimSuffix(name, "Router")
		if name != "" {
			return humanizeIdentifier(name)
		}
	}
	for _, segment := range strings.Split(strings.Trim(ep.Path, "/"), "/") {
		if segment == "" || strings.HasPrefix(segment, ":") || segment == "api" || segment == "v1" || segment == "v2" {
			continue
		}
		return humanizeIdentifier(segment)
	}
	if ep.SourceFile != "" {
		dir := path.Dir(ep.SourceFile)
		if dir != "." && dir != "/" {
			parts := strings.Split(dir, "/")
			return humanizeIdentifier(parts[len(parts)-1])
		}
	}
	return "General"
}

func sanitizeIdentifier(value string) string {
	value = strings.ToLower(strings.Trim(value, "/"))
	value = strings.NewReplacer("/", "-", ":", "", "{", "", "}", "", " ", "-", ".", "-", "_", "-").Replace(value)
	if value == "" {
		return "endpoint"
	}
	return value
}
