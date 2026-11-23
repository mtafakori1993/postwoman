package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"Postwoman/internal/ai"
	"Postwoman/internal/model"
)

func TestDecodeAISuggestionsAcceptsLooseValues(t *testing.T) {
	reply := "```json\n" + `{
		"endpoints": [{
			"id": "post-login",
			"method": "post",
			"path": "/login",
			"name": "User login",
			"description": "Authenticates a user.",
			"authRequired": "true",
			"authType": "JWT cookie",
			"requestBody": "{\"email\":\"string\",\"password\":\"string\"}",
			"queryParams": ["email"],
			"pathParams": [],
			"headers": [{"name":"Content-Type","type":"string","required":"yes","description":"application/json"}],
			"statusCodes": ["200 OK", "400", 500, 500],
			"confidence": "88",
			"notes": "First note.\n- Second note."
		}]
	}` + "\n```"

	suggestions, err := decodeAISuggestions(reply)
	if err != nil {
		t.Fatalf("decodeAISuggestions: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("suggestions = %d, want 1", len(suggestions))
	}
	got := suggestions[0]
	if got.ID != "post-login" || got.Method != "POST" || got.Path != "/login" {
		t.Fatalf("identity = %q/%q/%q", got.ID, got.Method, got.Path)
	}
	if got.AuthRequired == nil || !*got.AuthRequired {
		t.Fatalf("authRequired = %v, want true", got.AuthRequired)
	}
	if got.AuthType != "session" {
		t.Fatalf("authType = %q, want session", got.AuthType)
	}
	body, ok := got.RequestBody.(map[string]any)
	if !ok || body["email"] != "string" || body["password"] != "string" {
		t.Fatalf("requestBody = %#v", got.RequestBody)
	}
	if len(got.QueryParams) != 1 || got.QueryParams[0].Name != "email" || got.QueryParams[0].In != "query" {
		t.Fatalf("queryParams = %#v", got.QueryParams)
	}
	if len(got.Headers) != 1 || got.Headers[0].In != "header" || !got.Headers[0].Required {
		t.Fatalf("headers = %#v", got.Headers)
	}
	if !reflect.DeepEqual(got.StatusCodes, []int{200, 400, 500}) {
		t.Fatalf("statusCodes = %#v", got.StatusCodes)
	}
	if got.Confidence != 88 {
		t.Fatalf("confidence = %d, want 88", got.Confidence)
	}
	if !reflect.DeepEqual(got.Notes, []string{"First note.", "Second note."}) {
		t.Fatalf("notes = %#v", got.Notes)
	}
}

func TestDecodeAISuggestionsAcceptsBareArray(t *testing.T) {
	suggestions, err := decodeAISuggestions(`[{"id":"get-health"},{"id":"post-login"}]`)
	if err != nil {
		t.Fatalf("decodeAISuggestions: %v", err)
	}
	if len(suggestions) != 2 || suggestions[0].ID != "get-health" || suggestions[1].ID != "post-login" {
		t.Fatalf("suggestions = %#v", suggestions)
	}
}

func TestDecodeAISuggestionsSalvagesTruncatedReply(t *testing.T) {
	reply := `{"endpoints":[{"id":"get-health","name":"Health","statusCodes":[200,`
	suggestions, err := decodeAISuggestions(reply)
	if err != nil {
		t.Fatalf("decodeAISuggestions: %v", err)
	}
	if len(suggestions) != 1 || suggestions[0].ID != "get-health" {
		t.Fatalf("suggestions = %#v", suggestions)
	}
	if !reflect.DeepEqual(suggestions[0].StatusCodes, []int{200}) {
		t.Fatalf("statusCodes = %#v", suggestions[0].StatusCodes)
	}
}

func TestDecodeAISuggestionsRejectsProse(t *testing.T) {
	_, err := decodeAISuggestions("I cannot determine the endpoints from this code.")
	if err == nil {
		t.Fatal("expected an error for a prose-only reply")
	}
}

func TestApplySuggestionsMergesFlexibleFields(t *testing.T) {
	ep := &model.DiscoveredEndpoint{
		ID: "post-login", Method: "POST", Path: "/login", Name: "Login",
		Confidence: 60, Origin: "static",
	}
	a := &analyzer{
		endpoints: map[string]*model.DiscoveredEndpoint{"POST /login": ep},
		order:     []string{"POST /login"},
	}
	authRequired := true
	a.applySuggestions([]aiSuggestion{{
		ID: "post-login", Description: "Signs in a user.", AuthRequired: &authRequired, AuthType: "session",
		RequestBody: map[string]any{"email": "string"},
		QueryParams: []model.DiscoveryParam{{Name: "email", In: "query", Type: "string"}},
		Headers:     []model.DiscoveryParam{{Name: "Content-Type", In: "header", Type: "string", Required: true}},
		StatusCodes: []int{200, 400}, Confidence: 88, Notes: []string{"AI note."},
	}})

	if ep.Description != "Signs in a user." {
		t.Fatalf("description = %q", ep.Description)
	}
	if !ep.AuthRequired || ep.AuthType != "session" {
		t.Fatalf("auth = %v/%q", ep.AuthRequired, ep.AuthType)
	}
	if !strings.Contains(ep.RequestBody, `"email": "string"`) {
		t.Fatalf("requestBody = %q", ep.RequestBody)
	}
	if len(ep.QueryParams) != 1 || len(ep.Headers) != 1 {
		t.Fatalf("params = %#v headers = %#v", ep.QueryParams, ep.Headers)
	}
	if !reflect.DeepEqual(ep.StatusCodes, []int{200, 400}) || ep.Confidence != 88 {
		t.Fatalf("statusCodes = %#v confidence = %d", ep.StatusCodes, ep.Confidence)
	}
	if ep.Origin != "merged" {
		t.Fatalf("origin = %q, want merged", ep.Origin)
	}
}

func TestApplySuggestionsMatchesByMethodAndPath(t *testing.T) {
	ep := &model.DiscoveredEndpoint{ID: "post-login", Method: "POST", Path: "/login", Confidence: 60, Origin: "static"}
	a := &analyzer{
		endpoints: map[string]*model.DiscoveredEndpoint{"POST /login": ep},
		order:     []string{"POST /login"},
	}
	a.applySuggestions([]aiSuggestion{{ID: "model-rewrote-this", Method: "POST", Path: "/login", Description: "Signs in."}})
	if ep.Description != "Signs in." {
		t.Fatalf("description = %q", ep.Description)
	}
}

// TestEnrichWithAIRetriesTruncatedReply reproduces the reported failure: the
// first reply is only prose because the reasoning model hit its completion
// limit. Enrichment must retry with a bigger budget and merge the second reply.
func TestEnrichWithAIRetriesTruncatedReply(t *testing.T) {
	var calls int32
	var tokenMu sync.Mutex
	tokens := []int{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if value, ok := payload["max_tokens"].(float64); ok {
			tokenMu.Lock()
			tokens = append(tokens, int(value))
			tokenMu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		if atomic.AddInt32(&calls, 1) == 1 {
			body, _ := json.Marshal(map[string]any{
				"model": "test-model",
				"choices": []any{map[string]any{
					"message":       map[string]any{"role": "assistant", "content": "We need to produce JSON for these endpoints. The first endpoint is..."},
					"finish_reason": "length",
				}},
			})
			_, _ = w.Write(body)
			return
		}
		content := `{"endpoints":[{"id":"post-login","name":"User Login",` +
			`"description":"Authenticates a user with email and password.","authRequired":true,` +
			`"authType":"session","requestBody":{"email":"string","password":"string"},` +
			`"queryParams":[],"pathParams":[],"headers":[],"statusCodes":[200,400],` +
			`"confidence":90,"notes":["AI note."]}]}`
		body, _ := json.Marshal(map[string]any{
			"model": "test-model",
			"choices": []any{map[string]any{
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			}},
		})
		_, _ = w.Write(body)
	}))
	defer server.Close()

	settings := model.AppSettings{
		AIEnabled:    true,
		AIProviderID: "test",
		AIProviders: []model.AIProviderConfig{{
			ID: "test", Name: "Test provider", Kind: "custom",
			BaseURL: server.URL, Model: "test-model", TimeoutSeconds: 10,
		}},
	}
	ep := &model.DiscoveredEndpoint{
		ID: "post-login", Method: "POST", Path: "/login", Handler: "login",
		SourceFile: "AuthController.java", Framework: "Spring Boot",
		HandlerSnippet: `@PostMapping("/login") public void login() {}`,
		Confidence:     60, Origin: "static",
	}
	a := &analyzer{
		profile:   &projectProfile{Languages: []string{"Java"}, Frameworks: []string{"Spring Boot"}},
		endpoints: map[string]*model.DiscoveredEndpoint{"POST /login": ep},
		order:     []string{"POST /login"},
	}

	warnings := a.enrichWithAI(context.Background(), settings, ai.NewManager(), model.DiscoveryOptions{MaxAIFiles: 5}, func(string, int, string, string) {})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("provider calls = %d, want 2 (initial + retry)", got)
	}
	tokenMu.Lock()
	gotTokens := append([]int(nil), tokens...)
	tokenMu.Unlock()
	if !reflect.DeepEqual(gotTokens, []int{aiInitialMaxTokens, aiRetryMaxTokens}) {
		t.Fatalf("max tokens = %v, want [%d %d]", gotTokens, aiInitialMaxTokens, aiRetryMaxTokens)
	}
	if ep.Description != "Authenticates a user with email and password." {
		t.Fatalf("description = %q", ep.Description)
	}
	if !ep.AuthRequired || ep.AuthType != "session" {
		t.Fatalf("auth = %v/%q", ep.AuthRequired, ep.AuthType)
	}
	if !strings.Contains(ep.RequestBody, `"email": "string"`) {
		t.Fatalf("requestBody = %q", ep.RequestBody)
	}
	if ep.Confidence != 90 || ep.Origin != "merged" {
		t.Fatalf("confidence = %d origin = %q", ep.Confidence, ep.Origin)
	}
}

func TestBuildAIPayloadIncludesResponseSchema(t *testing.T) {
	payload := buildAIPayload(
		&projectProfile{Languages: []string{"Go"}, Frameworks: []string{"chi"}},
		[]*model.DiscoveredEndpoint{{ID: "get-health", Method: "GET", Path: "/health", HandlerSnippet: "r.Get(\"/health\", health)"}},
	)
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if _, ok := decoded["responseSchema"]; !ok {
		t.Fatal("payload is missing responseSchema")
	}
	if rules, ok := decoded["rules"].([]any); !ok || len(rules) == 0 {
		t.Fatalf("payload rules = %#v", decoded["rules"])
	}
	if !strings.Contains(payload, `"id":"get-health"`) {
		t.Fatalf("payload does not include the endpoint id: %s", payload)
	}
}

func TestEnrichWithAIRetriesTruncatedPartialReply(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		content := `{"endpoints":[{"id":"get-health","name":"Health Check","description":"Liveness probe.","statusCodes":[200]},` +
			`{"id":"post-login","name":"User Login","description":"Authenticates a user.","statusCodes":[200,401]}]}`
		finish := "stop"
		if atomic.AddInt32(&calls, 1) == 1 {
			// A valid but truncated prefix: the parser can salvage the first
			// endpoint, and the retry must recover the second.
			content = `{"endpoints":[{"id":"get-health","name":"Health Check","description":"Liveness probe.","statusCodes":[200`
			finish = "length"
		}
		body, _ := json.Marshal(map[string]any{
			"model": "test-model",
			"choices": []any{map[string]any{
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": finish,
			}},
		})
		_, _ = w.Write(body)
	}))
	defer server.Close()

	settings := model.AppSettings{
		AIEnabled:    true,
		AIProviderID: "test",
		AIProviders: []model.AIProviderConfig{{
			ID: "test", Name: "Test provider", Kind: "custom",
			BaseURL: server.URL, Model: "test-model", TimeoutSeconds: 10,
		}},
	}
	health := &model.DiscoveredEndpoint{ID: "get-health", Method: "GET", Path: "/health", Confidence: 60, Origin: "static"}
	login := &model.DiscoveredEndpoint{ID: "post-login", Method: "POST", Path: "/login", Confidence: 60, Origin: "static"}
	a := &analyzer{
		profile: &projectProfile{},
		endpoints: map[string]*model.DiscoveredEndpoint{
			"GET /health": health,
			"POST /login": login,
		},
		order: []string{"GET /health", "POST /login"},
	}

	warnings := a.enrichWithAI(context.Background(), settings, ai.NewManager(), model.DiscoveryOptions{MaxAIFiles: 5}, func(string, int, string, string) {})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("provider calls = %d, want 2 (initial + retry)", got)
	}
	if health.Description != "Liveness probe." || health.Origin != "merged" {
		t.Fatalf("health = %#v", health)
	}
	if login.Description != "Authenticates a user." || login.Origin != "merged" {
		t.Fatalf("login was not recovered by the retry: %#v", login)
	}
}

func TestMatchesExpectedEndpoints(t *testing.T) {
	expected := []*model.DiscoveredEndpoint{{
		ID: "post-login", Method: "POST", Path: "/login",
	}}
	if matchesExpectedEndpoints([]aiSuggestion{{ID: "<echo the input id unchanged>"}}, expected) {
		t.Fatal("a response-schema example must not count as a real suggestion")
	}
	if !matchesExpectedEndpoints([]aiSuggestion{{ID: "post-login"}}, expected) {
		t.Fatal("an exact id should match")
	}
	if !matchesExpectedEndpoints([]aiSuggestion{{ID: "rewritten", Method: "post", Path: "/login"}}, expected) {
		t.Fatal("a matching method+path should match")
	}
}
