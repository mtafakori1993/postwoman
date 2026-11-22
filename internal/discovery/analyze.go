package discovery

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"Postwoman/internal/model"
)

// analyzer walks the scanned files and accumulates endpoints. Extraction is
// static and deterministic; AI is only used later to enrich unresolved details.
type analyzer struct {
	profile      *projectProfile
	files        []*fileEntry
	byRel        map[string]*fileEntry
	endpoints    map[string]*model.DiscoveredEndpoint
	order        []string
	warnings     []string
	handlerIndex map[string][]*fileEntry
	fileInfos    map[string]*model.DiscoveryFileInfo
}

func newAnalyzer(profile *projectProfile, files []*fileEntry) *analyzer {
	a := &analyzer{
		profile:      profile,
		files:        files,
		byRel:        map[string]*fileEntry{},
		endpoints:    map[string]*model.DiscoveredEndpoint{},
		handlerIndex: map[string][]*fileEntry{},
		fileInfos:    map[string]*model.DiscoveryFileInfo{},
	}
	for _, f := range files {
		a.byRel[f.Rel] = f
		a.indexHandlers(f)
	}
	return a
}

// indexHandlers records where each function/method is declared so routes can be
// traced to their handlers.
func (a *analyzer) indexHandlers(f *fileEntry) {
	for _, name := range declaredSymbols(f) {
		a.handlerIndex[name] = append(a.handlerIndex[name], f)
	}
}

// addEndpoint merges a newly discovered endpoint into the result set. Duplicates
// (same method + path) keep the highest-confidence information and combine
// notes so nothing is silently lost.
func (a *analyzer) addEndpoint(ep model.DiscoveredEndpoint) {
	ep.Method = strings.ToUpper(strings.TrimSpace(ep.Method))
	if ep.Method == "" {
		ep.Method = "GET"
	}
	ep.Path = normalizeRoutePath(ep.Path)
	if ep.Path == "" {
		return
	}
	key := ep.Method + " " + ep.Path
	if existing, ok := a.endpoints[key]; ok {
		mergeEndpoint(existing, ep)
		return
	}
	if ep.ID == "" {
		ep.ID = fmt.Sprintf("%s-%s", strings.ToLower(ep.Method), sanitizeIdentifier(ep.Path))
	}
	if ep.Name == "" {
		ep.Name = routeDisplayName(ep.Method, ep.Path, ep.Handler)
	}
	if ep.Folder == "" {
		ep.Folder = folderForEndpoint(ep)
	}
	if ep.Confidence == 0 {
		ep.Confidence = 60
	}
	if ep.Origin == "" {
		ep.Origin = "static"
	}
	ep.Selected = true
	a.endpoints[key] = &ep
	a.order = append(a.order, key)
}

// analyze runs every static strategy and returns the merged endpoint list.
func (a *analyzer) analyze(ctx context.Context, progress func(stage string, percent int, message, current string)) ([]model.DiscoveredEndpoint, []string) {
	progress("framework", 12, "Detecting framework and language", "")
	for _, f := range a.files {
		if ctx.Err() != nil {
			a.warnings = append(a.warnings, "analysis cancelled")
			break
		}
		percent := 15
		if len(a.files) > 0 {
			percent = 15 + (len(a.fileInfos)*55)/len(a.files)
		}
		progress("routes", percent, "Discovering routes and handlers", f.Rel)
		info := f.Info
		before := len(a.endpoints)
		a.extractFile(f)
		info.HasRoutes = len(a.endpoints) > before
		info.Analyzed = true
		a.fileInfos[f.Rel] = &info
	}
	progress("schemas", 72, "Extracting request schemas and parameters", "")
	a.enrichSchemas(ctx)
	progress("auth", 80, "Identifying authentication requirements", "")
	a.enrichAuth()
	progress("relationships", 88, "Resolving endpoint relationships", "")
	a.resolveRelationships()
	a.collectSnippets()
	return a.sortedEndpoints(), a.warnings
}

func (a *analyzer) extractFile(f *fileEntry) {
	switch f.Language {
	case "go":
		extractGoRoutes(a, f)
	case "javascript", "typescript":
		extractJSRoutes(a, f)
	case "python":
		extractPythonRoutes(a, f)
	case "php":
		extractPHPRoutes(a, f)
	case "ruby":
		extractRubyRoutes(a, f)
	case "java", "kotlin":
		extractJavaRoutes(a, f)
	case "csharp":
		extractCSharpRoutes(a, f)
	case "rust":
		extractRustRoutes(a, f)
	case "elixir":
		extractElixirRoutes(a, f)
	case "json", "yaml":
		extractOpenAPI(a, f)
	}
}

func (a *analyzer) sortedEndpoints() []model.DiscoveredEndpoint {
	out := make([]model.DiscoveredEndpoint, 0, len(a.order))
	for _, key := range a.order {
		if ep, ok := a.endpoints[key]; ok {
			out = append(out, *ep)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Folder != out[j].Folder {
			return out[i].Folder < out[j].Folder
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// resolveHandler traces a handler expression to its declaration.
func (a *analyzer) resolveHandler(expr string, sourceFile string) (name, file string, line int) {
	name = strings.TrimSpace(expr)
	name = strings.TrimSuffix(name, "()")
	if name == "" {
		return "", "", 0
	}
	simple := name
	if idx := strings.LastIndex(simple, "."); idx >= 0 {
		simple = simple[idx+1:]
	}
	if candidates, ok := a.handlerIndex[simple]; ok && len(candidates) > 0 {
		chosen := candidates[0]
		for _, candidate := range candidates {
			if candidate.Rel == sourceFile {
				chosen = candidate
				break
			}
		}
		for i, text := range chosen.Lines {
			if strings.Contains(text, simple) && (strings.Contains(text, "func ") || strings.Contains(text, "def ") || strings.Contains(text, "function ") || strings.Contains(text, "=>")) {
				return name, chosen.Rel, i + 1
			}
		}
		return name, chosen.Rel, 0
	}
	return name, "", 0
}

// handlerBody returns the source of a handler function when it can be located.
func (a *analyzer) handlerBody(name string) string {
	simple := name
	if idx := strings.LastIndex(simple, "."); idx >= 0 {
		simple = simple[idx+1:]
	}
	for _, candidate := range a.handlerIndex[simple] {
		for _, text := range candidate.Lines {
			if !strings.Contains(text, simple) {
				continue
			}
			start := strings.Index(candidate.Content, text)
			if start < 0 {
				continue
			}
			rest := candidate.Content[start:]
			open := strings.Index(rest, "{")
			if open < 0 {
				continue
			}
			body, _ := splitBraceBody(rest, open)
			if len(body) > 4000 {
				body = body[:4000]
			}
			return body
		}
	}
	return ""
}
