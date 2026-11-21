package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"

	"Postwoman/internal/model"
)

func TestConnectSendReceiveAndClose(t *testing.T) {
	upgrader := gws.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(mt, append([]byte("echo:"), data...)); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	received := make(chan string, 8)
	manager := NewManager(func(_ string, msg model.WSMessage) {
		if msg.Type == "received" {
			received <- msg.Data
		}
	})

	ctx := context.Background()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	if _, err := manager.Connect(ctx, "conn-1", wsURL, nil, nil, true); err != nil {
		t.Fatalf("connect: %v", err)
	}

	if _, err := manager.Send("conn-1", "hello", false); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case got := <-received:
		if got != "echo:hello" {
			t.Fatalf("unexpected message: %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the echoed message")
	}

	msgs := manager.Messages("conn-1")
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages (system, sent, received), got %d", len(msgs))
	}

	if err := manager.Close("conn-1"); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestConnectToBadURLFails(t *testing.T) {
	manager := NewManager(nil)
	_, err := manager.Connect(context.Background(), "x", "ws://127.0.0.1:1", nil, nil, true)
	if err == nil {
		t.Fatal("expected an error connecting to a closed port")
	}
}
