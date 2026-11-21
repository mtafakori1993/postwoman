package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"Postwoman/internal/model"
)

func TestOAuth2PKCEFlow(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at-123", "refresh_token": "rt-456", "expires_in": 3600, "token_type": "Bearer",
		})
	}))
	defer srv.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	oauth := model.OAuth2Auth{
		GrantType: "authorization_code", AuthURL: "https://auth.example.com/authorize", TokenURL: srv.URL,
		ClientID: "client-id", ClientSecret: "client-secret", RedirectURL: "http://localhost/callback",
		Scope: "read write", PKCE: true,
	}
	started, authURL, err := client.StartOAuth2(oauth)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.CodeVerifier == "" || started.State == "" {
		t.Fatal("expected PKCE verifier and state")
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	for _, key := range []string{"code_challenge", "code_challenge_method", "client_id", "state", "response_type"} {
		if q.Get(key) == "" {
			t.Fatalf("authorization URL missing %s: %s", key, authURL)
		}
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected challenge method: %s", q.Get("code_challenge_method"))
	}

	updated, err := client.ExchangeOAuth2Code(context.Background(), started, "the-code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if updated.AccessToken != "at-123" || updated.RefreshToken != "rt-456" {
		t.Fatalf("tokens not stored: %+v", updated)
	}
	if updated.ExpiresAt <= time.Now().UnixMilli() {
		t.Fatalf("expiry not set: %d", updated.ExpiresAt)
	}
	if gotForm.Get("grant_type") != "authorization_code" || gotForm.Get("code") != "the-code" {
		t.Fatalf("unexpected token form: %v", gotForm)
	}
	if gotForm.Get("code_verifier") != started.CodeVerifier {
		t.Fatalf("PKCE verifier not sent")
	}
}

func TestOAuth2PasteRedirectURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc"}`))
	}))
	defer srv.Close()
	client, _ := NewClient()
	oauth := model.OAuth2Auth{GrantType: "authorization_code", TokenURL: srv.URL, ClientID: "id"}
	updated, err := client.ExchangeOAuth2Code(context.Background(), oauth, "http://localhost/callback?code=xyz&state=s")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if updated.AccessToken != "abc" {
		t.Fatalf("unexpected token: %+v", updated)
	}
}

func TestOAuth2ClientCredentialsAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("client_secret") == "bad" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"cc-token","expires_in":"60"}`))
	}))
	defer srv.Close()
	client, _ := NewClient()
	updated, err := client.FetchOAuth2Token(context.Background(), model.OAuth2Auth{
		GrantType: "client_credentials", TokenURL: srv.URL, ClientID: "id", ClientSecret: "secret", Scope: "read",
	})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if updated.AccessToken != "cc-token" || updated.TokenType != "Bearer" || updated.ExpiresAt == 0 {
		t.Fatalf("unexpected token: %+v", updated)
	}
	if _, err := client.FetchOAuth2Token(context.Background(), model.OAuth2Auth{
		GrantType: "client_credentials", TokenURL: srv.URL, ClientID: "id", ClientSecret: "bad",
	}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 error, got %v", err)
	}
}
