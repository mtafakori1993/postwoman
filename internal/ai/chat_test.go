package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"Postwoman/internal/model"
)

// chatStub starts a server that answers every request with the given status and
// body, mimicking an OpenAI-compatible endpoint.
func chatStub(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func stubClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := NewClientWithProxy(model.AIProviderConfig{
		Kind: "custom", Name: "Custom OpenAI-compatible",
		BaseURL: server.URL, Model: "gpt-test", TimeoutSeconds: 10,
	}, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

func stubChat(t *testing.T, server *httptest.Server) (*ChatResponse, error) {
	t.Helper()
	client := stubClient(t, server)
	return client.Chat(context.Background(), ChatRequest{
		Messages:  []Message{{Role: "user", Content: "list endpoints"}},
		MaxTokens: 128,
		JSONMode:  true,
	})
}

// TestChatOpenAIToleratesNestedUsage guards the regression where modern
// providers include object-valued token details (prompt_tokens_details, …),
// which made a strict map[string]int decode fail with the unhelpful
// "returned an unexpected response" error.
func TestChatOpenAIToleratesNestedUsage(t *testing.T) {
	body := `{
		"model": "gpt-test",
		"choices": [{"message": {"role": "assistant", "content": "{\"endpoints\":[]}"}}],
		"usage": {
			"prompt_tokens": 12,
			"completion_tokens": 34,
			"total_tokens": 46,
			"prompt_tokens_details": {"cached_tokens": 0},
			"completion_tokens_details": {"reasoning_tokens": 8}
		}
	}`
	reply, err := stubChat(t, chatStub(t, http.StatusOK, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply.Content != `{"endpoints":[]}` {
		t.Fatalf("content = %q", reply.Content)
	}
	if reply.Usage["total_tokens"] != 46 {
		t.Fatalf("usage = %v, want total_tokens 46", reply.Usage)
	}
}

func TestChatOpenAIHandlesStringError(t *testing.T) {
	body := `{"error": "model not found"}`
	_, err := stubChat(t, chatStub(t, http.StatusOK, body))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("error = %q, want it to include the provider message", err)
	}
}

func TestChatOpenAIHandlesNestedError(t *testing.T) {
	body := `{"error": {"message": "rate limited", "type": "rate_limit_error", "code": "429"}}`
	_, err := stubChat(t, chatStub(t, http.StatusOK, body))
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("error = %v, want the nested message", err)
	}
}

func TestChatOpenAIAcceptsSingleMessageShape(t *testing.T) {
	// Some gateways (Ollama-style, Copilot proxies) return a top-level message.
	body := `{"model": "gpt-test", "message": {"role": "assistant", "content": "hello"}}`
	reply, err := stubChat(t, chatStub(t, http.StatusOK, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply.Content != "hello" {
		t.Fatalf("content = %q, want hello", reply.Content)
	}
}

func TestChatOpenAIAcceptsContentParts(t *testing.T) {
	// Some gateways return content as an array of text parts rather than a
	// single string.
	body := `{"choices":[{"message":{"role":"assistant","content":[{"type":"text","text":"{\"endpoints\":[]}"}]}}]}`
	reply, err := stubChat(t, chatStub(t, http.StatusOK, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply.Content != `{"endpoints":[]}` {
		t.Fatalf("content = %q", reply.Content)
	}
}

func TestChatOpenAIUnreadableResponseExplainsItself(t *testing.T) {
	server := chatStub(t, http.StatusOK, "<html><body>gateway error</body></html>")
	_, err := stubChat(t, server)
	if err == nil {
		t.Fatal("expected an error for a non-JSON body")
	}
	if !strings.Contains(err.Error(), "unreadable response") {
		t.Fatalf("error = %q, want an unreadable-response message", err)
	}
	// The snippet must surface what actually came back so the user can diagnose
	// a misconfigured base URL or proxy.
	if !strings.Contains(err.Error(), "gateway error") {
		t.Fatalf("error = %q, want a body snippet", err)
	}
}

func TestChatOpenAIRetriesWithoutOptionalParamsOn400(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Unsupported parameter: max_tokens"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	t.Cleanup(server.Close)

	reply, err := stubChat(t, server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply.Content != "ok" {
		t.Fatalf("content = %q, want ok", reply.Content)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("requests = %d, want 2 (initial + retry)", got)
	}
}

func TestChatOpenAIRetainsOriginalErrorWhenRetryFails(t *testing.T) {
	server := chatStub(t, http.StatusBadRequest, `{"error":{"message":"invalid api key"}}`)
	_, err := stubChat(t, server)
	if err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("error = %v, want the original provider error", err)
	}
}

// TestChatOpenAIReportedDiscoveryReply reproduces the exact failure reported
// against the API Discovery Agent with a custom OpenAI-compatible provider: a
// valid {"endpoints":[...]} reply wrapped in a response that also carries nested
// usage details. It must now be parsed and survive ExtractJSON.
func TestChatOpenAIReportedDiscoveryReply(t *testing.T) {
	content := `{"endpoints":[{"id":"post-backendurl-login","name":"User Login",` +
		`"description":"Authenticates a user with email and password.","authRequired":false,` +
		`"authType":"unknown","requestBody":{"email":"string","password":"string"},` +
		`"queryParams":[],"pathParams":[],"headers":[{"name":"Content-Type","in":"header",` +
		`"type":"string","required":true,"description":"Expected to be application/json."}],` +
		`"statusCodes":[200,400,401,500],"confidence":90,"notes":["The path includes ':backendURL'."]}]}`

	body, err := json.Marshal(map[string]any{
		"model": "custom-model",
		"choices": []any{
			map[string]any{"message": map[string]any{"role": "assistant", "content": content}},
		},
		"usage": map[string]any{
			"prompt_tokens": 900, "completion_tokens": 500, "total_tokens": 1400,
			"prompt_tokens_details":     map[string]any{"cached_tokens": 0},
			"completion_tokens_details": map[string]any{"reasoning_tokens": 0},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	reply, err := stubChat(t, chatStub(t, http.StatusOK, string(body)))
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	extracted, err := ExtractJSON(reply.Content)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	if !strings.Contains(extracted, `"id":"post-backendurl-login"`) {
		t.Fatalf("unexpected extracted JSON: %s", extracted)
	}
}
