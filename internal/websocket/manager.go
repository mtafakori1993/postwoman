// Package websocket manages live WebSocket connections for the UI.
package websocket

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"Postwoman/internal/model"

	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"
)

type connection struct {
	id        string
	url       string
	ws        *gws.Conn
	mu        sync.Mutex
	messages  []model.WSMessage
	connected bool
	cancel    context.CancelFunc
}

// Manager owns all live WebSocket connections.
type Manager struct {
	mu      sync.Mutex
	conns   map[string]*connection
	onEvent func(connID string, msg model.WSMessage)
}

func NewManager(onEvent func(connID string, msg model.WSMessage)) *Manager {
	return &Manager{conns: map[string]*connection{}, onEvent: onEvent}
}

func (m *Manager) Connect(ctx context.Context, id, rawURL string, headers []model.KeyValue, protocols []string, rejectUnauthorized bool) (*model.WSMessage, error) {
	if id == "" {
		id = uuid.New().String()
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("WebSocket URL is empty")
	}
	if !strings.HasPrefix(rawURL, "ws://") && !strings.HasPrefix(rawURL, "wss://") {
		if strings.HasPrefix(rawURL, "https://") {
			rawURL = "wss://" + strings.TrimPrefix(rawURL, "https://")
		} else if strings.HasPrefix(rawURL, "http://") {
			rawURL = "ws://" + strings.TrimPrefix(rawURL, "http://")
		} else {
			rawURL = "ws://" + rawURL
		}
	}
	dialer := gws.Dialer{
		HandshakeTimeout: 15 * time.Second,
		TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: !rejectUnauthorized},
	}
	if len(protocols) > 0 {
		dialer.Subprotocols = protocols
	}
	h := http.Header{}
	for _, kv := range headers {
		if kv.Enabled && kv.Key != "" {
			h.Add(kv.Key, kv.Value)
		}
	}
	ws, resp, err := dialer.DialContext(ctx, rawURL, h)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("WebSocket handshake failed: %s", resp.Status)
		}
		return nil, fmt.Errorf("WebSocket connection failed: %w", err)
	}
	connCtx, cancel := context.WithCancel(context.Background())
	c := &connection{id: id, url: rawURL, ws: ws, connected: true, cancel: cancel}
	m.mu.Lock()
	if old, ok := m.conns[id]; ok && old.cancel != nil {
		old.cancel()
		_ = old.ws.Close()
	}
	m.conns[id] = c
	m.mu.Unlock()

	open := model.WSMessage{
		ID: uuid.New().String(), Type: "system",
		Data: fmt.Sprintf("Connected to %s", rawURL), Timestamp: time.Now().UnixMilli(),
	}
	m.appendAndEmit(c, open)

	go m.readLoop(connCtx, c)
	return &open, nil
}

func (m *Manager) readLoop(ctx context.Context, c *connection) {
	defer func() {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		_ = c.ws.Close()
		if m.onEvent != nil {
			m.onEvent(c.id, model.WSMessage{
				ID: uuid.New().String(), Type: "system", Data: "Disconnected",
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		messageType, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		msg := model.WSMessage{
			ID: uuid.New().String(), Type: "received", Data: string(data),
			IsBinary: messageType == gws.BinaryMessage, Timestamp: time.Now().UnixMilli(),
		}
		m.appendAndEmit(c, msg)
	}
}

func (m *Manager) appendAndEmit(c *connection, msg model.WSMessage) {
	c.mu.Lock()
	c.messages = append(c.messages, msg)
	if len(c.messages) > 2000 {
		c.messages = c.messages[len(c.messages)-2000:]
	}
	c.mu.Unlock()
	if m.onEvent != nil {
		m.onEvent(c.id, msg)
	}
}

func (m *Manager) Send(id, data string, isBinary bool) (*model.WSMessage, error) {
	m.mu.Lock()
	c, ok := m.conns[id]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("WebSocket connection %q is not open", id)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return nil, fmt.Errorf("WebSocket connection %q is closed", id)
	}
	messageType := gws.TextMessage
	if isBinary {
		messageType = gws.BinaryMessage
	}
	if err := c.ws.WriteMessage(messageType, []byte(data)); err != nil {
		return nil, fmt.Errorf("send failed: %w", err)
	}
	msg := model.WSMessage{
		ID: uuid.New().String(), Type: "sent", Data: data, IsBinary: isBinary,
		Timestamp: time.Now().UnixMilli(),
	}
	c.messages = append(c.messages, msg)
	if m.onEvent != nil {
		go m.onEvent(id, msg)
	}
	return &msg, nil
}

// Ping sends a WebSocket ping control frame and records a system message so
// the UI can show the round-trip attempt.
func (m *Manager) Ping(id string) error {
	m.mu.Lock()
	c, ok := m.conns[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("WebSocket connection %q is not open", id)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return fmt.Errorf("WebSocket connection %q is closed", id)
	}
	if err := c.ws.WriteControl(gws.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	msg := model.WSMessage{
		ID: uuid.New().String(), Type: "system", Data: "ping sent", Timestamp: time.Now().UnixMilli(),
	}
	c.messages = append(c.messages, msg)
	if m.onEvent != nil {
		go m.onEvent(id, msg)
	}
	return nil
}
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	c, ok := m.conns[id]
	delete(m.conns, id)
	m.mu.Unlock()
	if !ok {
		return nil
	}
	if c.cancel != nil {
		c.cancel()
	}
	return c.ws.Close()
}

// List returns a snapshot of every known connection and its message history.
func (m *Manager) List() []model.WSConnection {
	m.mu.Lock()
	conns := make([]*connection, 0, len(m.conns))
	for _, c := range m.conns {
		conns = append(conns, c)
	}
	m.mu.Unlock()
	out := make([]model.WSConnection, 0, len(conns))
	for _, c := range conns {
		c.mu.Lock()
		msgs := make([]model.WSMessage, len(c.messages))
		copy(msgs, c.messages)
		connected := c.connected
		c.mu.Unlock()
		out = append(out, model.WSConnection{
			ID: c.id, URL: c.url, Connected: connected, Messages: msgs,
		})
	}
	return out
}

func (m *Manager) Messages(id string) []model.WSMessage {
	m.mu.Lock()
	c, ok := m.conns[id]
	m.mu.Unlock()
	if !ok {
		return []model.WSMessage{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]model.WSMessage, len(c.messages))
	copy(out, c.messages)
	return out
}

func (m *Manager) Clear(id string) {
	m.mu.Lock()
	c, ok := m.conns[id]
	m.mu.Unlock()
	if !ok {
		return
	}
	c.mu.Lock()
	c.messages = nil
	c.mu.Unlock()
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	conns := make([]*connection, 0, len(m.conns))
	for _, c := range m.conns {
		conns = append(conns, c)
	}
	m.conns = map[string]*connection{}
	m.mu.Unlock()
	for _, c := range conns {
		if c.cancel != nil {
			c.cancel()
		}
		_ = c.ws.Close()
	}
}
