package mockserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"Postwoman/internal/model"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestMockServerServesRouteAndLogs(t *testing.T) {
	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager()
	server := model.MockServer{
		ID:   "mock-1",
		Name: "Test",
		Port: port,
		Routes: []model.MockRoute{
			{
				ID: "r1", Method: "GET", Path: "/hello", StatusCode: 200,
				Headers: []model.KeyValue{{Key: "Content-Type", Value: "application/json", Enabled: true}},
				Body:    `{"message":"hi"}`,
			},
		},
	}
	if err := manager.Start(ctx, server); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !manager.IsRunning("mock-1") {
		t.Fatal("expected server to report running")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/hello", port))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if string(body) != `{"message":"hi"}` {
		t.Fatalf("body = %q", body)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q", resp.Header.Get("Content-Type"))
	}

	// Unmatched route should be a 404.
	resp2, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/missing", port))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 404 {
		t.Fatalf("expected 404 for unmatched route, got %d", resp2.StatusCode)
	}

	logs := manager.Logs("mock-1")
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logs))
	}

	if err := manager.Stop("mock-1"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if manager.IsRunning("mock-1") {
		t.Fatal("expected server to be stopped")
	}
}
