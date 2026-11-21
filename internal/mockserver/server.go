// Package mockserver runs local HTTP mock servers defined by the user.
package mockserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"Postwoman/internal/model"

	"github.com/google/uuid"
)

type runningServer struct {
	server *http.Server
	logs   []model.MockRequestLog
	mu     sync.Mutex
}

// Manager starts and stops mock servers and keeps their request logs.
type Manager struct {
	mu      sync.Mutex
	servers map[string]*runningServer
}

func NewManager() *Manager {
	return &Manager{servers: map[string]*runningServer{}}
}

func (m *Manager) Start(ctx context.Context, s model.MockServer) error {
	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	m.mu.Lock()
	if _, exists := m.servers[s.ID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("mock server %q is already running", s.Name)
	}
	m.mu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(s.Port))
	if err != nil {
		return fmt.Errorf("could not listen on port %d: %w", s.Port, err)
	}
	rs := &runningServer{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.handle(rs, s, w, r)
	})
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	rs.server = srv

	m.mu.Lock()
	m.servers[s.ID] = rs
	m.mu.Unlock()

	go func() {
		<-ctx.Done()
		_ = srv.Close()
		m.mu.Lock()
		delete(m.servers, s.ID)
		m.mu.Unlock()
	}()
	go func() {
		_ = srv.Serve(ln)
	}()
	return nil
}

func (m *Manager) handle(rs *runningServer, s model.MockServer, w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	_ = r.Body.Close()
	route := matchRoute(s.Routes, r)
	logEntry := model.MockRequestLog{
		ID: uuid.New().String(), Method: r.Method, Path: r.URL.RequestURI(),
		Body: string(body), Timestamp: time.Now().UnixMilli(),
	}
	for k, vals := range r.Header {
		for _, v := range vals {
			logEntry.Headers = append(logEntry.Headers, model.KeyValue{Key: k, Value: v, Enabled: true})
		}
	}
	if route == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no matching mock route","method":"` + r.Method + `","path":"` + r.URL.Path + `"}`))
		logEntry.Status = http.StatusNotFound
		rs.mu.Lock()
		rs.logs = append([]model.MockRequestLog{logEntry}, rs.logs...)
		rs.mu.Unlock()
		return
	}
	if route.Delay > 0 {
		time.Sleep(time.Duration(route.Delay) * time.Millisecond)
	}
	for _, h := range route.Headers {
		if h.Enabled && h.Key != "" {
			w.Header().Add(h.Key, h.Value)
		}
	}
	status := route.StatusCode
	if status == 0 {
		status = 200
	}
	w.WriteHeader(status)
	if route.Body != "" {
		_, _ = w.Write([]byte(route.Body))
	}
	logEntry.Status = status
	rs.mu.Lock()
	rs.logs = append([]model.MockRequestLog{logEntry}, rs.logs...)
	if len(rs.logs) > 200 {
		rs.logs = rs.logs[:200]
	}
	rs.mu.Unlock()
}

func matchRoute(routes []model.MockRoute, r *http.Request) *model.MockRoute {
	for i := range routes {
		route := &routes[i]
		if !strings.EqualFold(route.Method, r.Method) && route.Method != "*" && route.Method != "" {
			continue
		}
		if route.Path != r.URL.Path && route.Path != "*" {
			continue
		}
		if route.MatchQuery {
			// Compare the full request URI.
			if route.Path != r.URL.RequestURI() {
				continue
			}
		}
		return route
	}
	return nil
}

func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	rs, ok := m.servers[id]
	delete(m.servers, id)
	m.mu.Unlock()
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return rs.server.Shutdown(ctx)
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.servers))
	for id := range m.servers {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.Stop(id)
	}
}

func (m *Manager) Logs(id string) []model.MockRequestLog {
	m.mu.Lock()
	rs, ok := m.servers[id]
	m.mu.Unlock()
	if !ok {
		return []model.MockRequestLog{}
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	out := make([]model.MockRequestLog, len(rs.logs))
	copy(out, rs.logs)
	return out
}

func (m *Manager) IsRunning(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.servers[id]
	return ok
}

func (m *Manager) ClearLogs(id string) {
	m.mu.Lock()
	rs, ok := m.servers[id]
	m.mu.Unlock()
	if !ok {
		return
	}
	rs.mu.Lock()
	rs.logs = nil
	rs.mu.Unlock()
}
