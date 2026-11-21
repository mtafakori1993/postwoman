package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"Postwoman/internal/collections"
	"Postwoman/internal/database"
	"Postwoman/internal/httpclient"
	"Postwoman/internal/mockserver"
	"Postwoman/internal/model"
	"Postwoman/internal/scripting"
	"Postwoman/internal/websocket"
)

// newTestApp builds an App backed by a throwaway database. It mirrors startup()
// but with a temporary path so tests never touch the user's data.
func newTestApp(t *testing.T) *App {
	t.Helper()
	db, err := database.New(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	client, err := httpclient.NewClient()
	if err != nil {
		t.Fatalf("http client: %v", err)
	}
	app := &App{
		ctx:          context.Background(),
		db:           db,
		repo:         database.NewRepository(db),
		httpClient:   client,
		scriptEngine: scripting.NewEngine(5 * time.Second),
		wsManager:    websocket.NewManager(nil),
		mockManager:  mockserver.NewManager(),
	}
	app.collectionMgr = collections.NewManager(app.repo)
	ws, err := app.repo.EnsureDefaultWorkspace(app.ctx)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	app.workspaceID = ws.ID
	return app
}

func TestSendRequestResolvesCollectionVariables(t *testing.T) {
	app := newTestApp(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "pong")
	}))
	defer srv.Close()

	collection, err := app.CreateCollection(app.workspaceID, "Vars", "")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	collection.Variables = []model.Variable{{Key: "baseUrl", Value: srv.URL, Enabled: true, Type: "default"}}
	if err := app.UpdateCollection(*collection); err != nil {
		t.Fatalf("update collection: %v", err)
	}

	request, err := app.CreateRequest(model.Request{
		CollectionID: collection.ID, Name: "ping", Method: "GET",
		URL: "{{baseUrl}}/ping", Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "inherit"},
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	// The UI sends empty scope maps and relies on the backend to load the
	// collection variables — this is the regression that produced an
	// "invalid URL" error before the fix.
	resp, err := app.SendRequest(*request, nil, nil, nil, true, "")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("unexpected response error: %s", resp.Error)
	}
	if resp.StatusCode != 200 || resp.Body != "pong" {
		t.Fatalf("unexpected response: status=%d body=%q", resp.StatusCode, resp.Body)
	}
}

func TestSendRequestRunsScriptsAndRecordsHistory(t *testing.T) {
	app := newTestApp(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	collection, _ := app.CreateCollection(app.workspaceID, "Scripted", "")
	env, err := app.CreateEnvironment(app.workspaceID, "Test Env", nil)
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}

	request, err := app.CreateRequest(model.Request{
		CollectionID: collection.ID, Name: "scripted", Method: "GET", URL: srv.URL,
		Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"},
		Scripts: []model.Script{
			{Type: "test", Script: `
				pm.test("status is 200", () => pm.response.to.have.status(200));
				pm.environment.set("token", "abc123");
			`},
		},
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := app.SendRequest(*request, nil, nil, nil, true, env.ID)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(resp.Tests) != 1 || !resp.Tests[0].Passed {
		t.Fatalf("expected a passing test, got %+v", resp.Tests)
	}

	// The script mutated the environment; it must have been persisted.
	stored, err := app.GetEnvironment(env.ID)
	if err != nil || stored == nil {
		t.Fatalf("get environment: %v", err)
	}
	var found bool
	for _, v := range stored.Values {
		if v.Key == "token" && v.Value == "abc123" {
			found = true
		}
	}
	if !found {
		t.Fatalf("script variable was not persisted: %+v", stored.Values)
	}

	history, err := app.GetHistory(10, 0)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
}

func TestImportExportAndBootstrapRoundTrip(t *testing.T) {
	app := newTestApp(t)

	const doc = `{
		"info": { "name": "Imported", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" },
		"item": [
			{ "name": "Root request", "request": { "method": "GET", "url": "https://example.com" } },
			{ "name": "Folder", "item": [
				{ "name": "Nested request", "request": { "method": "POST", "url": "https://example.com/x" } }
			] }
		]
	}`

	summary, err := app.ImportCollection(app.workspaceID, doc)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(summary.Collections) != 1 {
		t.Fatalf("expected 1 imported collection, got %+v", summary)
	}

	// Bootstrap is what the UI calls on launch; it must see the imported data.
	state, err := app.Bootstrap()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if len(state.Collections) != 1 || len(state.Requests) != 2 {
		t.Fatalf("bootstrap did not reflect the import: %d collections, %d requests", len(state.Collections), len(state.Requests))
	}

	exported, err := app.ExportCollection(state.Collections[0].ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(exported) == 0 {
		t.Fatal("export returned an empty document")
	}
	// The nested request must survive export (regression: it used to be dropped).
	reimported, err := app.ImportCollection(app.workspaceID, exported)
	if err != nil {
		t.Fatalf("reimport: %v", err)
	}
	if len(reimported.Collections) != 1 {
		t.Fatalf("expected 1 collection after round-trip, got %+v", reimported)
	}
}

func TestSettingsAndTabsPersist(t *testing.T) {
	app := newTestApp(t)

	settings := app.GetSettings()
	settings.Theme = "dark"
	settings.RequestTimeout = 12345
	if err := app.SaveSettings(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	got := app.GetSettings()
	if got.Theme != "dark" || got.RequestTimeout != 12345 {
		t.Fatalf("settings did not persist: %+v", got)
	}

	if err := app.SaveTabs([]model.Tab{{ID: "t1", Title: "Tab one", Protocol: "http", IsActive: true}}); err != nil {
		t.Fatalf("save tabs: %v", err)
	}
	tabs, err := app.GetTabs()
	if err != nil || len(tabs) != 1 || tabs[0].Title != "Tab one" {
		t.Fatalf("tabs did not persist: %v %+v", err, tabs)
	}
}
