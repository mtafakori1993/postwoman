package ai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	"Postwoman/internal/model"
)

// relayTransport is used by the test forward proxy so it never recurses through
// the ambient proxy environment.
var relayTransport = &http.Transport{Proxy: nil}

// newForwardProxy starts a minimal HTTP forward proxy and counts the requests
// it relays. It only understands absolute-form requests, which is what Go emits
// when a transport is configured with a proxy.
func newForwardProxy(t *testing.T) (*httptest.Server, *int64) {
	t.Helper()
	var hits int64
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		req, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()
		resp, err := relayTransport.RoundTrip(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	t.Cleanup(proxy.Close)
	return proxy, &hits
}

func proxyFromServer(t *testing.T, server *httptest.Server) *model.Proxy {
	t.Helper()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}
	port := 80
	if p := parsed.Port(); p != "" {
		value, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("parse proxy port: %v", err)
		}
		port = value
	}
	return &model.Proxy{Host: parsed.Hostname(), Port: port, Protocol: "http"}
}

// TestClientRoutesThroughProxy proves an AI request follows the configured
// proxy, and that bypassing the upstream host skips it again.
func TestClientRoutesThroughProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-test"}]}`))
	}))
	defer upstream.Close()

	proxy, hits := newForwardProxy(t)
	proxyCfg := proxyFromServer(t, proxy)

	cfg := model.AIProviderConfig{Kind: "custom", BaseURL: upstream.URL, Model: "gpt-test", TimeoutSeconds: 10}

	// Without a proxy the request goes straight to the upstream.
	direct, err := NewClientWithProxy(cfg, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := direct.ListModels(context.Background()); err != nil {
		t.Fatalf("direct list models: %v", err)
	}
	if atomic.LoadInt64(hits) != 0 {
		t.Fatalf("proxy was used without being configured (%d hits)", atomic.LoadInt64(hits))
	}

	// With a proxy the request is relayed through it.
	proxied, err := NewClientWithProxy(cfg, proxyCfg)
	if err != nil {
		t.Fatalf("new proxied client: %v", err)
	}
	models, err := proxied.ListModels(context.Background())
	if err != nil {
		t.Fatalf("proxied list models: %v", err)
	}
	if len(models) != 1 || models[0] != "gpt-test" {
		t.Fatalf("unexpected models via proxy: %v", models)
	}
	if atomic.LoadInt64(hits) == 0 {
		t.Fatal("expected the request to be relayed through the proxy")
	}

	// Bypassing the upstream host sends traffic direct again.
	before := atomic.LoadInt64(hits)
	bypassed := *proxyCfg
	bypassed.Bypass = []string{"127.0.0.1"}
	bypassClient, err := NewClientWithProxy(cfg, &bypassed)
	if err != nil {
		t.Fatalf("new bypass client: %v", err)
	}
	if _, err := bypassClient.ListModels(context.Background()); err != nil {
		t.Fatalf("bypassed list models: %v", err)
	}
	if atomic.LoadInt64(hits) != before {
		t.Fatal("bypassed host should not be relayed through the proxy")
	}
}

func TestNewClientWithProxyRejectsInvalidProxy(t *testing.T) {
	cfg := model.AIProviderConfig{Kind: "custom", BaseURL: "https://example.com/v1", Model: "m"}
	if _, err := NewClientWithProxy(cfg, &model.Proxy{Host: "proxy", Port: 8080, Protocol: "ftp"}); err == nil {
		t.Fatal("expected an unsupported-protocol error")
	}
}
