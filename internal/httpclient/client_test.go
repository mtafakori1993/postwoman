package httpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Postwoman/internal/model"
	"Postwoman/internal/variables"
)

func do(t *testing.T, req *model.Request, scopes ...map[string]string) *model.Response {
	t.Helper()
	client, err := NewClient()
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	resp, err := client.Do(context.Background(), req, variables.New(scopes...))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	return resp
}

func TestGetWithQueryAndHeaders(t *testing.T) {
	var gotPath, gotQuery, gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotHeader = r.Header.Get("X-Custom")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	resp := do(t, &model.Request{
		Method:        "GET",
		URL:           srv.URL + "/users/:id",
		QueryParams:   []model.KeyValue{{Key: "page", Value: "2", Enabled: true}},
		Headers:       []model.KeyValue{{Key: "X-Custom", Value: "{{v}}", Enabled: true}},
		PathVariables: []model.KeyValue{{Key: "id", Value: "42", Enabled: true}},
		Body:          model.Body{Mode: "none"},
		Auth:          model.Auth{Type: "noauth"},
	}, map[string]string{"v": "header-value"})

	if resp.Error != "" {
		t.Fatalf("unexpected response error: %s", resp.Error)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if gotPath != "/users/42" {
		t.Fatalf("path variable not applied: %q", gotPath)
	}
	if gotQuery != "page=2" {
		t.Fatalf("query param not applied: %q", gotQuery)
	}
	if gotHeader != "header-value" {
		t.Fatalf("header not resolved: %q", gotHeader)
	}
	if !strings.Contains(resp.Body, "world") {
		t.Fatalf("body not captured: %q", resp.Body)
	}
}

func TestPostJSONBody(t *testing.T) {
	var body []byte
	var contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(201)
	}))
	defer srv.Close()

	resp := do(t, &model.Request{
		Method: "POST",
		URL:    srv.URL,
		Body:   model.Body{Mode: "json", Raw: `{"name":"{{name}}"}`},
		Auth:   model.Auth{Type: "noauth"},
	}, map[string]string{"name": "Ada"})

	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d (%s)", resp.StatusCode, resp.Error)
	}
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("unexpected content type %q", contentType)
	}
	var parsed map[string]string
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("body is not valid json: %v (%s)", err, body)
	}
	if parsed["name"] != "Ada" {
		t.Fatalf("body variable not resolved: %s", body)
	}
}

func TestURLEncodedBody(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}))
	defer srv.Close()

	do(t, &model.Request{
		Method: "POST",
		URL:    srv.URL,
		Body: model.Body{Mode: "urlencoded", URLEncoded: []model.KeyValue{
			{Key: "a", Value: "1", Enabled: true},
			{Key: "b", Value: "x y", Enabled: true},
		}},
		Auth: model.Auth{Type: "noauth"},
	})

	if !strings.Contains(body, "a=1") || !strings.Contains(body, "b=x+y") {
		t.Fatalf("unexpected urlencoded body: %q", body)
	}
}

func TestBearerAndBasicAuth(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	do(t, &model.Request{
		Method: "GET", URL: srv.URL, Body: model.Body{Mode: "none"},
		Auth: model.Auth{Type: "bearer", Bearer: &model.BearerAuth{Token: "secret-token"}},
	})
	if authHeader != "Bearer secret-token" {
		t.Fatalf("unexpected bearer header: %q", authHeader)
	}

	do(t, &model.Request{
		Method: "GET", URL: srv.URL, Body: model.Body{Mode: "none"},
		Auth: model.Auth{Type: "basic", Basic: &model.BasicAuth{Username: "user", Password: "pass"}},
	})
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Fatalf("expected basic auth header, got %q", authHeader)
	}
}

func TestAPIKeyInHeaderAndQuery(t *testing.T) {
	var headerKey, queryKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerKey = r.Header.Get("X-API-Key")
		queryKey = r.URL.Query().Get("api_key")
	}))
	defer srv.Close()

	do(t, &model.Request{
		Method: "GET", URL: srv.URL, Body: model.Body{Mode: "none"},
		Auth: model.Auth{Type: "apikey", APIKey: &model.APIKeyAuth{Key: "X-API-Key", Value: "abc", AddTo: "header"}},
	})
	if headerKey != "abc" {
		t.Fatalf("api key header not set: %q", headerKey)
	}

	do(t, &model.Request{
		Method: "GET", URL: srv.URL, Body: model.Body{Mode: "none"},
		Auth: model.Auth{Type: "apikey", APIKey: &model.APIKeyAuth{Key: "api_key", Value: "xyz", AddTo: "query"}},
	})
	if queryKey != "xyz" {
		t.Fatalf("api key query not set: %q", queryKey)
	}
}

func TestTimeoutProducesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer srv.Close()

	resp := do(t, &model.Request{
		Method: "GET", URL: srv.URL, Body: model.Body{Mode: "none"},
		Auth:     model.Auth{Type: "noauth"},
		Settings: model.Settings{Timeout: 100, FollowRedirects: true},
	})
	if resp.Error == "" {
		t.Fatal("expected a timeout error on the response")
	}
}

func TestInvalidURLProducesError(t *testing.T) {
	resp := do(t, &model.Request{Method: "GET", URL: "", Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"}})
	if resp.Error == "" {
		t.Fatal("expected an error for an empty URL")
	}
}

func TestUnsupportedProtocol(t *testing.T) {
	resp := do(t, &model.Request{Method: "GET", URL: "ftp://example.com/file", Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"}})
	if !strings.Contains(resp.Error, "unsupported URL protocol") {
		t.Fatalf("expected unsupported protocol error, got %q", resp.Error)
	}
}
