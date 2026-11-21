package importexport

import (
	"strings"
	"testing"

	"Postwoman/internal/model"
)

func sampleRequest() *model.Request {
	return &model.Request{
		Name:   "Create user",
		Method: "POST",
		URL:    "https://api.example.com/users?expand=team",
		Headers: []model.KeyValue{
			{Key: "X-Trace", Value: "abc", Enabled: true},
			{Key: "disabled", Value: "nope", Enabled: false},
		},
		QueryParams: []model.KeyValue{
			{Key: "expand", Value: "team", Enabled: true}, // already in the URL
			{Key: "notify", Value: "true", Enabled: true},
		},
		Body: model.Body{Mode: "json", Raw: `{"name":"Ada"}`},
		Auth: model.Auth{Type: "bearer", Bearer: &model.BearerAuth{Token: "{{token}}"}},
	}
}

var targets = []string{"curl", "http", "fetch", "node", "python", "go", "php"}

func TestExportAsCodeTargets(t *testing.T) {
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			out, err := ExportAsCode(sampleRequest(), target)
			if err != nil {
				t.Fatalf("export: %v", err)
			}
			if strings.Contains(out, "disabled: nope") {
				t.Fatal("disabled header leaked into generated code")
			}
			if !strings.Contains(out, "expand=team") {
				t.Fatalf("URL query lost: %s", out)
			}
			if !strings.Contains(out, "notify=true") {
				t.Fatalf("query params not merged: %s", out)
			}
			if !strings.Contains(out, "{\"name\":\"Ada\"}") && !strings.Contains(out, `{\"name\":\"Ada\"}`) {
				if target != "http" {
					t.Fatalf("body missing in %s output: %s", target, out)
				}
			}
		})
	}
}

func TestExportAsCodeRejectsUnknownTarget(t *testing.T) {
	if _, err := ExportAsCode(sampleRequest(), "cobol"); err == nil {
		t.Fatal("expected an error for an unsupported target")
	}
}

func TestExportAsCurlIncludesAuth(t *testing.T) {
	out := ExportAsCurl(sampleRequest())
	if !strings.Contains(out, "Authorization: Bearer {{token}}") {
		t.Fatalf("auth header missing: %s", out)
	}
	if !strings.Contains(out, "notify=true") {
		t.Fatalf("query params missing: %s", out)
	}
}

func TestParseCurlAsRequest(t *testing.T) {
	req, err := ParseCurlAsRequest(`curl 'https://api.example.com/items?limit=5' -X POST -H 'Content-Type: application/json' -H 'X-Api: 1' -d '{"a":1}' -u alice:secret`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Method != "POST" || req.URL != "https://api.example.com/items" {
		t.Fatalf("unexpected request: %+v", req)
	}
	if len(req.QueryParams) != 1 || req.QueryParams[0].Key != "limit" {
		t.Fatalf("query params not parsed: %+v", req.QueryParams)
	}
	if req.Auth.Type != "basic" || req.Auth.Basic == nil || req.Auth.Basic.Username != "alice" {
		t.Fatalf("basic auth not parsed: %+v", req.Auth)
	}
	if req.Body.Mode != "json" || !strings.Contains(req.Body.Raw, "\"a\":1") {
		t.Fatalf("body not parsed: %+v", req.Body)
	}
	if _, err := ParseCurlAsRequest("wget https://example.com"); err == nil {
		t.Fatal("expected non-curl input to fail")
	}
}
