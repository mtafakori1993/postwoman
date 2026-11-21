package importexport

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"Postwoman/internal/database"
)

const postmanCollection = `{
  "info": {
    "name": "Sample API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    { "key": "baseUrl", "value": "https://api.example.com" }
  ],
  "item": [
    {
      "name": "Users",
      "item": [
        {
          "name": "List users",
          "request": {
            "method": "GET",
            "header": [{ "key": "Accept", "value": "application/json" }],
            "url": {
              "raw": "{{baseUrl}}/users?page=1",
              "query": [{ "key": "page", "value": "1" }]
            }
          },
          "event": [
            { "listen": "test", "script": { "exec": ["pm.test('ok', () => pm.response.to.have.status(200));"] } }
          ]
        }
      ]
    },
    {
      "name": "Create user",
      "request": {
        "method": "POST",
        "header": [{ "key": "Content-Type", "value": "application/json" }],
        "body": { "mode": "raw", "raw": "{\"name\":\"Ada\"}" },
        "url": "{{baseUrl}}/users"
      }
    }
  ]
}`

const openAPISpec = `{
  "openapi": "3.0.0",
  "info": { "title": "Pet Store", "version": "1.0.0" },
  "servers": [{ "url": "https://petstore.example.com/v1" }],
  "paths": {
    "/pets": {
      "get": {
        "summary": "List pets",
        "responses": { "200": { "description": "ok" } }
      },
      "post": {
        "summary": "Create pet",
        "requestBody": {
          "content": {
            "application/json": { "schema": { "type": "object", "properties": { "name": { "type": "string" } } } }
          }
        },
        "responses": { "201": { "description": "created" } }
      }
    }
  }
}`

const swaggerSpec = `{
  "swagger": "2.0",
  "info": { "title": "Legacy", "version": "1.0" },
  "host": "legacy.example.com",
  "basePath": "/api",
  "schemes": ["https"],
  "paths": {
    "/ping": { "get": { "summary": "Ping", "responses": { "200": { "description": "ok" } } } }
  }
}`

const harDoc = `{
  "log": {
    "version": "1.2",
    "creator": { "name": "test", "version": "1" },
    "entries": [
      {
        "startedDateTime": "2024-01-01T00:00:00Z",
        "time": 12,
        "request": {
          "method": "GET",
          "url": "https://example.com/things?limit=5",
          "httpVersion": "HTTP/1.1",
          "headers": [{ "name": "Accept", "value": "application/json" }],
          "queryString": [{ "name": "limit", "value": "5" }],
          "cookies": [],
          "headersSize": -1,
          "bodySize": 0
        },
        "response": { "status": 200, "statusText": "OK", "httpVersion": "HTTP/1.1", "headers": [], "cookies": [], "content": { "size": 0, "mimeType": "application/json" }, "redirectURL": "", "headersSize": -1, "bodySize": 0 }
      }
    ]
  }
}`

func TestDetectFormats(t *testing.T) {
	cases := []struct {
		name string
		data string
		want string
	}{
		{"postman", postmanCollection, "postman"},
		{"openapi", openAPISpec, "openapi3"},
		{"swagger", swaggerSpec, "swagger2"},
		{"har", harDoc, "har"},
		{"curl", `curl -X POST https://example.com -d 'a=1'`, "curl"},
		{"graphql", "type Query {\n  users: [User]\n}", "graphql"},
	}
	for _, tc := range cases {
		if got := Detect([]byte(tc.data), "file.json"); got != tc.want {
			t.Errorf("%s: Detect = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestParsePostmanCollection(t *testing.T) {
	parsed, err := Parse([]byte(postmanCollection), "sample.postman_collection.json")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Format != "postman" {
		t.Fatalf("format = %q", parsed.Format)
	}
	if len(parsed.Collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(parsed.Collections))
	}
	coll := parsed.Collections[0]
	if coll.Info.Name != "Sample API" {
		t.Fatalf("name = %q", coll.Info.Name)
	}
	if len(coll.Requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(coll.Requests))
	}
	if len(coll.Folders) != 1 || coll.Folders[0].Name != "Users" {
		t.Fatalf("expected the Users folder, got %+v", coll.Folders)
	}
	var nested bool
	for _, r := range coll.Requests {
		if r.Name == "List users" {
			nested = r.FolderID != nil && *r.FolderID == coll.Folders[0].TempID
			if len(r.Scripts) == 0 {
				t.Fatal("expected the test script to be imported")
			}
			if len(r.QueryParams) == 0 {
				t.Fatal("expected query params to be imported")
			}
		}
	}
	if !nested {
		t.Fatal("expected the nested request to be linked to its folder")
	}
}

func TestParseOpenAPIAndSwagger(t *testing.T) {
	openapi, err := Parse([]byte(openAPISpec), "openapi.json")
	if err != nil {
		t.Fatalf("openapi parse: %v", err)
	}
	if openapi.Format != "openapi3" || len(openapi.Collections) != 1 {
		t.Fatalf("unexpected openapi parse: %+v", openapi.Format)
	}
	if len(openapi.Collections[0].Requests) != 2 {
		t.Fatalf("expected 2 openapi requests, got %d", len(openapi.Collections[0].Requests))
	}
	for _, r := range openapi.Collections[0].Requests {
		if !strings.HasPrefix(r.URL, "https://petstore.example.com/v1") {
			t.Fatalf("server URL not applied: %q", r.URL)
		}
	}

	swagger, err := Parse([]byte(swaggerSpec), "swagger.json")
	if err != nil {
		t.Fatalf("swagger parse: %v", err)
	}
	if len(swagger.Collections) != 1 || len(swagger.Collections[0].Requests) != 1 {
		t.Fatalf("unexpected swagger parse: %+v", swagger.Collections)
	}
	if swagger.Collections[0].Requests[0].URL != "https://legacy.example.com/api/ping" {
		t.Fatalf("swagger base path not applied: %q", swagger.Collections[0].Requests[0].URL)
	}
}

func TestParseHAR(t *testing.T) {
	parsed, err := Parse([]byte(harDoc), "capture.har")
	if err != nil {
		t.Fatalf("har parse: %v", err)
	}
	if len(parsed.Collections) != 1 || len(parsed.Collections[0].Requests) != 1 {
		t.Fatalf("unexpected har parse: %+v", parsed.Collections)
	}
	req := parsed.Collections[0].Requests[0]
	if req.Method != "GET" || !strings.Contains(req.URL, "example.com/things") {
		t.Fatalf("unexpected har request: %+v", req)
	}
}

func TestParseCurl(t *testing.T) {
	parsed, err := Parse([]byte(`curl -X POST 'https://api.example.com/items' -H 'Content-Type: application/json' -d '{"a":1}'`), "req.txt")
	if err != nil {
		t.Fatalf("curl parse: %v", err)
	}
	if len(parsed.Collections) != 1 || len(parsed.Collections[0].Requests) != 1 {
		t.Fatalf("unexpected curl parse: %+v", parsed.Collections)
	}
	req := parsed.Collections[0].Requests[0]
	if req.Method != "POST" || req.URL != "https://api.example.com/items" {
		t.Fatalf("unexpected curl request: %+v", req)
	}
	if req.Body.Raw != `{"a":1}` {
		t.Fatalf("curl body not captured: %q", req.Body.Raw)
	}
}

func TestPersistAndExportRoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := database.New(filepath.Join(t.TempDir(), "import.db"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	ws, err := repo.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}

	parsed, err := Parse([]byte(postmanCollection), "sample.postman_collection.json")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	collectionIDs, _, err := Persist(ctx, repo, ws.ID, parsed)
	if err != nil {
		t.Fatalf("persist: %v", err)
	}
	if len(collectionIDs) != 1 {
		t.Fatalf("expected 1 persisted collection, got %d", len(collectionIDs))
	}

	stored, err := repo.GetCollection(ctx, collectionIDs[0])
	if err != nil || stored == nil {
		t.Fatalf("stored collection not found: %v", err)
	}
	if stored.Info.Name != "Sample API" {
		t.Fatalf("stored name = %q", stored.Info.Name)
	}
	requests, err := repo.ListAllRequests(ctx, ws.ID)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 stored requests, got %d", len(requests))
	}

	// Export and re-import to prove the data round-trips without loss.
	exported, err := ExportCollection(ctx, repo, collectionIDs[0])
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !json.Valid(exported) {
		t.Fatalf("exported collection is not valid JSON")
	}
	reparsed, err := Parse(exported, "exported.postman_collection.json")
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if len(reparsed.Collections) != 1 {
		t.Fatalf("expected 1 collection on reparse, got %d", len(reparsed.Collections))
	}
	if len(reparsed.Collections[0].Requests) != 2 {
		t.Fatalf("expected 2 requests after round-trip, got %d", len(reparsed.Collections[0].Requests))
	}
	if len(reparsed.Collections[0].Folders) != 1 {
		t.Fatalf("expected the folder to survive the round-trip, got %d", len(reparsed.Collections[0].Folders))
	}
}

func TestExportAsCurl(t *testing.T) {
	parsed, err := Parse([]byte(postmanCollection), "sample.json")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	req := parsed.Collections[0].Requests[1] // Create user (POST)
	cmd := ExportAsCurl(&req)
	if !strings.HasPrefix(cmd, "curl") || !strings.Contains(cmd, "POST") {
		t.Fatalf("unexpected curl export: %q", cmd)
	}
	if !strings.Contains(cmd, "/users") {
		t.Fatalf("url missing from curl export: %q", cmd)
	}
}
