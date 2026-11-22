package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"Postwoman/internal/model"
)

func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// newFixture builds a small mixed-language project so every static extractor
// path is exercised without touching the network.
func newFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "package.json", `{
  "name": "fixture-api",
  "dependencies": { "express": "^4.18.0" }
}`)
	writeFixture(t, root, "src/routes/users.js", `const express = require('express')
const router = express.Router()

router.get('/users', listUsers)
router.post('/users', requireAuth, createUser)
router.get('/users/:id', requireAuth, getUser)

module.exports = router
`)
	writeFixture(t, root, "server.js", `const express = require('express')
const app = express()

app.get('/health', health)

app.listen(3000)
`)
	writeFixture(t, root, "api/main.py", `from fastapi import FastAPI, APIRouter

router = APIRouter(prefix="/api")

@router.get("/health")
async def health():
    return {"status": "ok"}

@router.post("/users")
async def create_user():
    return {"id": 1}
`)
	writeFixture(t, root, "cmd/server/main.go", `package main

import "net/http"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)
	http.HandleFunc("/metrics", metricsHandler)
}
`)
	writeFixture(t, root, "openapi.yaml", `openapi: 3.0.0
info:
  title: Fixture API
  version: 1.0.0
paths:
  /widgets:
    get:
      summary: List widgets
      operationId: listWidgets
`)
	writeFixture(t, root, ".gitignore", "ignored.js\n")
	writeFixture(t, root, "ignored.js", `const app = require('express')()
app.get('/gitignored', handler)
`)
	writeFixture(t, root, "node_modules/pkg/index.js", `const app = require('express')()
app.get('/from-node-modules', handler)
`)
	return root
}

func runDiscovery(t *testing.T, root string) (*Manager, *model.DiscoverySession, []model.DiscoveryProgress) {
	t.Helper()
	manager := NewManager(nil)
	var mu sync.Mutex
	events := []model.DiscoveryProgress{}
	id, err := manager.Start(model.DiscoveryOptions{
		Directory:        root,
		CollectionName:   "Fixture API",
		BaseURL:          "https://api.example.com",
		RespectGitignore: true,
		UseAI:            false,
		MaxFiles:         200,
		MaxFileSizeKB:    512,
	}, model.AppSettings{}, func(progress model.DiscoveryProgress) {
		mu.Lock()
		events = append(events, progress)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("start discovery: %v", err)
	}
	session := waitForSession(t, manager, id)
	mu.Lock()
	defer mu.Unlock()
	return manager, session, append([]model.DiscoveryProgress{}, events...)
}

func waitForSession(t *testing.T, manager *Manager, id string) *model.DiscoverySession {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		session, err := manager.Get(id)
		if err != nil {
			t.Fatalf("get session: %v", err)
		}
		if session.Status != "running" && session.Status != "cancelling" {
			return session
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("discovery session %s did not finish in time", id)
	return nil
}

func endpointByRoute(endpoints []model.DiscoveredEndpoint, method, path string) *model.DiscoveredEndpoint {
	for i := range endpoints {
		if strings.EqualFold(endpoints[i].Method, method) && endpoints[i].Path == path {
			return &endpoints[i]
		}
	}
	return nil
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}

func routePairs(endpoints []model.DiscoveredEndpoint) []string {
	out := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		out = append(out, endpoint.Method+" "+endpoint.Path)
	}
	return out
}

func TestManagerDiscoversMixedProject(t *testing.T) {
	manager, session, events := runDiscovery(t, newFixture(t))

	if session.Status != "completed" {
		t.Fatalf("expected completed status, got %q (error: %s)", session.Status, session.Error)
	}
	if session.Progress.Percent != 100 || !session.Progress.Done {
		t.Fatalf("expected finished progress, got %+v", session.Progress)
	}
	if len(events) == 0 {
		t.Fatal("expected progress events")
	}
	if session.Stats.FilesSeen == 0 || session.Stats.Endpoints == 0 {
		t.Fatalf("expected scanned files and endpoints, got %+v", session.Stats)
	}
	if session.Collection == "" {
		t.Fatal("expected a generated collection preview")
	}
	if !containsFold(session.Frameworks, "Express") {
		t.Fatalf("expected Express in frameworks, got %v", session.Frameworks)
	}
	if !containsFold(session.Frameworks, "FastAPI") {
		t.Fatalf("expected FastAPI in frameworks, got %v", session.Frameworks)
	}

	for _, want := range []struct{ method, path string }{
		{"GET", "/health"},
		{"POST", "/users"},
		{"GET", "/users/:id"},
		{"GET", "/api/health"},
		{"POST", "/api/users"},
		{"GET", "/status"},
		{"GET", "/metrics"},
		{"GET", "/widgets"},
	} {
		if endpointByRoute(session.Endpoints, want.method, want.path) == nil {
			t.Errorf("missing %s %s in %v", want.method, want.path, routePairs(session.Endpoints))
		}
	}
	if endpointByRoute(session.Endpoints, "GET", "/gitignored") != nil {
		t.Error("gitignored file was analysed")
	}
	if endpointByRoute(session.Endpoints, "GET", "/from-node-modules") != nil {
		t.Error("node_modules was analysed")
	}
	if widgets := endpointByRoute(session.Endpoints, "GET", "/widgets"); widgets != nil {
		if widgets.Origin != "openapi" || widgets.Confidence < 90 {
			t.Errorf("expected OpenAPI endpoint to be merged with high confidence, got %+v", widgets)
		}
	}

	manager.Remove(session.ID)
	if _, err := manager.Get(session.ID); err == nil {
		t.Fatal("expected removed session to be unavailable")
	}
}

func buildSelection(endpoints []model.DiscoveredEndpoint, keepID string) []model.DiscoveryEndpointEdit {
	edits := make([]model.DiscoveryEndpointEdit, 0, len(endpoints))
	for _, endpoint := range endpoints {
		edit := model.DiscoveryEndpointEdit{ID: endpoint.ID, Selected: endpoint.ID == keepID}
		if endpoint.ID == keepID {
			edit.Name = "Health probe"
			edit.Folder = "Ops"
		}
		edits = append(edits, edit)
	}
	return edits
}

func TestGenerateCollectionHonoursReviewEdits(t *testing.T) {
	manager, session, _ := runDiscovery(t, newFixture(t))
	get := endpointByRoute(session.Endpoints, "GET", "/health")
	if get == nil {
		t.Fatalf("fixture did not produce GET /health: %v", routePairs(session.Endpoints))
	}
	users := endpointByRoute(session.Endpoints, "POST", "/users")
	if users == nil {
		t.Fatalf("fixture did not produce POST /users: %v", routePairs(session.Endpoints))
	}

	content, err := manager.Generate(session.ID, model.DiscoveryGenerateRequest{
		SessionID:      session.ID,
		CollectionName: "Reviewed Fixture",
		BaseURL:        "https://staging.example.com",
		Endpoints:      buildSelection(session.Endpoints, get.ID),
	})
	if err != nil {
		t.Fatalf("generate collection: %v", err)
	}
	if warnings, err := ValidateCollection(content); err != nil {
		t.Fatalf("generated collection is invalid: %v", err)
	} else if len(warnings) > 0 {
		t.Logf("validation warnings: %v", warnings)
	}

	var doc struct {
		Info struct {
			Name   string `json:"name"`
			Schema string `json:"schema"`
		} `json:"info"`
		Item []struct {
			Name string `json:"name"`
			Item []struct {
				Name    string `json:"name"`
				Request struct {
					Method string `json:"method"`
					URL    struct {
						Raw string `json:"raw"`
					} `json:"url"`
				} `json:"request"`
			} `json:"item"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("collection is not valid JSON: %v", err)
	}
	if doc.Info.Name != "Reviewed Fixture" {
		t.Errorf("expected reviewed name, got %q", doc.Info.Name)
	}
	if !strings.Contains(doc.Info.Schema, "v2.1.0") {
		t.Errorf("expected Postman v2.1 schema, got %q", doc.Info.Schema)
	}
	if len(doc.Item) != 1 || doc.Item[0].Name != "Ops" {
		t.Fatalf("expected a single Ops folder, got %+v", doc.Item)
	}
	if len(doc.Item[0].Item) != 1 {
		t.Fatalf("expected exactly the selected endpoint, got %+v", doc.Item[0].Item)
	}
	request := doc.Item[0].Item[0]
	if request.Name != "Health probe" || request.Request.Method != "GET" {
		t.Errorf("unexpected request: %+v", request)
	}
	if request.Request.URL.Raw != "{{baseUrl}}/health" {
		t.Errorf("expected variable-based URL, got %q", request.Request.URL.Raw)
	}
}

func TestGenerateRejectsEmptySelection(t *testing.T) {
	manager, session, _ := runDiscovery(t, newFixture(t))
	edits := make([]model.DiscoveryEndpointEdit, 0, len(session.Endpoints))
	for _, endpoint := range session.Endpoints {
		edits = append(edits, model.DiscoveryEndpointEdit{ID: endpoint.ID, Selected: false})
	}
	if _, err := manager.Generate(session.ID, model.DiscoveryGenerateRequest{
		SessionID: session.ID, Endpoints: edits,
	}); err == nil {
		t.Fatal("expected an error when nothing is selected")
	}
}
func TestListDirectoryFlagsExcludedEntries(t *testing.T) {
	root := newFixture(t)
	manager := NewManager(nil)
	listing, err := manager.ListDirectory(root, model.DiscoveryOptions{})
	if err != nil {
		t.Fatalf("list directory: %v", err)
	}
	byName := map[string]model.DiscoveryDirEntry{}
	for _, entry := range listing.Entries {
		byName[entry.Name] = entry
	}
	if entry, ok := byName["node_modules"]; !ok || !entry.Excluded {
		t.Errorf("expected node_modules to be excluded, got %+v", entry)
	}
	if entry, ok := byName[".gitignore"]; !ok || !entry.Excluded {
		t.Errorf("expected hidden files to be flagged, got %+v", entry)
	}
	if entry, ok := byName["openapi.yaml"]; !ok || entry.Excluded {
		t.Errorf("expected openapi.yaml to be listed normally, got %+v", entry)
	}
}

func TestSourceFileWindow(t *testing.T) {
	manager, session, _ := runDiscovery(t, newFixture(t))
	source, err := manager.SourceFile(session.ID, "src/routes/users.js", 1, 3)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if source.Language != "javascript" || source.StartLine != 1 || source.EndLine != 3 {
		t.Errorf("unexpected window metadata: %+v", source)
	}
	if !strings.Contains(source.Content, "require('express')") {
		t.Errorf("unexpected source content: %q", source.Content)
	}
	if source.TotalLines < 3 {
		t.Errorf("expected total line count, got %d", source.TotalLines)
	}
	if _, err := manager.SourceFile(session.ID, "node_modules/pkg/index.js", 1, 2); err == nil {
		t.Error("expected excluded files to be unavailable")
	}
}

func TestNormalizeRoutePath(t *testing.T) {
	cases := map[string]string{
		"":                   "/",
		"users":              "/users",
		"/users/":            "/users",
		"/users/{id}":        "/users/:id",
		"/users/{id}/posts/": "/users/:id/posts",
		"/search?q=1":        "/search",
		"//double//slash":    "/double/slash",
	}
	for input, want := range cases {
		if got := normalizeRoutePath(input); got != want {
			t.Errorf("normalizeRoutePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestClassifyFileCategories(t *testing.T) {
	cases := []struct {
		name     string
		language string
		category string
		found    bool
		skip     bool
	}{
		{"src/routes/users.js", "javascript", CategoryRoute, true, false},
		{"openapi.yaml", "yaml", CategorySchema, true, false},
		{"package.json", "json", CategoryConfig, true, false},
		{"README.md", "markdown", CategoryDoc, true, false},
		{"package-lock.json", "json", CategoryOther, false, true},
		{"src/types.generated.ts", "typescript", CategoryOther, false, true},
	}
	for _, tc := range cases {
		language, category, candidate, reason := classifyFile(tc.name, filepath.Base(tc.name))
		if candidate != tc.found {
			t.Errorf("%s: candidate = %v, want %v (%s)", tc.name, candidate, tc.found, reason)
		}
		if tc.skip && reason == "" {
			t.Errorf("%s: expected a skip reason", tc.name)
		}
		if !tc.skip && language != tc.language {
			t.Errorf("%s: language = %q, want %q", tc.name, language, tc.language)
		}
		if !tc.skip && category != tc.category {
			t.Errorf("%s: category = %q, want %q", tc.name, category, tc.category)
		}
	}
}

func TestValidateCollectionRejectsGarbage(t *testing.T) {
	if _, err := ValidateCollection("not json"); err == nil {
		t.Fatal("expected invalid JSON to be rejected")
	}
	valid := `{"info":{"name":"ok","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[{"name":"Ping","request":{"method":"GET","url":"{{baseUrl}}/ping"}}]}`
	if _, err := ValidateCollection(valid); err != nil {
		t.Fatalf("expected a valid collection to pass: %v", err)
	}
}
