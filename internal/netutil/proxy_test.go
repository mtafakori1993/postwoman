package netutil

import (
	"net"
	"net/http"
	"testing"
	"time"

	"Postwoman/internal/model"
)

func TestNormalizeProxyRejectsBadInput(t *testing.T) {
	if _, err := NormalizeProxy(&model.Proxy{Host: "127.0.0.1", Port: 8080, Protocol: "ftp"}); err == nil {
		t.Fatal("expected an unsupported-protocol error")
	}
	if _, err := NormalizeProxy(&model.Proxy{Host: "127.0.0.1", Port: 0, Protocol: "http"}); err == nil {
		t.Fatal("expected a port error")
	}
	if _, err := NormalizeProxy(&model.Proxy{Host: "127.0.0.1", Port: 70000, Protocol: "http"}); err == nil {
		t.Fatal("expected an out-of-range port error")
	}
}

func TestNormalizeProxyTreatsMissingHostAsNoProxy(t *testing.T) {
	got, err := NormalizeProxy(&model.Proxy{Port: 8080})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil proxy, got %+v", got)
	}
}

func TestNormalizeProxyDefaultsProtocolAndTrimsBypass(t *testing.T) {
	got, err := NormalizeProxy(&model.Proxy{Host: " proxy.local ", Port: 3128, Bypass: []string{" example.com ", ""}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Protocol != "http" {
		t.Errorf("protocol = %q, want http", got.Protocol)
	}
	if got.Host != "proxy.local" {
		t.Errorf("host = %q, want trimmed value", got.Host)
	}
	if len(got.Bypass) != 1 || got.Bypass[0] != "example.com" {
		t.Errorf("bypass = %v, want [example.com]", got.Bypass)
	}
}

func TestHostMatches(t *testing.T) {
	cases := []struct {
		host, pattern string
		want          bool
	}{
		{"example.com", "example.com", true},
		{"api.example.com", "example.com", true},
		{"api.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"localhost", "localhost", true},
		{"other.com", "example.com", false},
		{"example.com", "", false},
	}
	for _, tc := range cases {
		if got := HostMatches(tc.host, tc.pattern); got != tc.want {
			t.Errorf("HostMatches(%q, %q) = %v, want %v", tc.host, tc.pattern, got, tc.want)
		}
	}
}

func TestProxyFuncHonoursBypass(t *testing.T) {
	proxyFunc, err := ProxyFunc(&model.Proxy{Host: "proxy.local", Port: 8080, Protocol: "http", Bypass: []string{"internal.local"}})
	if err != nil {
		t.Fatalf("ProxyFunc: %v", err)
	}

	direct, err := http.NewRequest(http.MethodGet, "https://internal.local/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := proxyFunc(direct)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("bypassed host should go direct, got %v", got)
	}

	through, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err = proxyFunc(through)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Host != "proxy.local:8080" {
		t.Fatalf("expected the request to use the proxy, got %v", got)
	}
}

func TestProxyFuncFallsBackToEnvironment(t *testing.T) {
	proxyFunc, err := ProxyFunc(nil)
	if err != nil {
		t.Fatalf("ProxyFunc(nil): %v", err)
	}
	if proxyFunc == nil {
		t.Fatal("expected the environment proxy function, got nil")
	}
}

func TestProbeReachableAndUnreachable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	if err := Probe(&model.Proxy{Host: "127.0.0.1", Port: port, Protocol: "http"}, time.Second); err != nil {
		t.Fatalf("expected a reachable proxy, got %v", err)
	}

	if err := Probe(&model.Proxy{Host: "127.0.0.1", Port: 1, Protocol: "http"}, 500*time.Millisecond); err == nil {
		t.Fatal("expected an unreachable-proxy error")
	}

	if err := Probe(&model.Proxy{}, time.Second); err == nil {
		t.Fatal("expected a missing-host error")
	}
}
