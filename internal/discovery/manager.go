package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Postwoman/internal/ai"
	"Postwoman/internal/model"

	"github.com/google/uuid"
)

// Manager owns discovery sessions. It is safe for concurrent use and keeps the
// AI provider configuration in the shared ai.Manager rather than per feature.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	ai       *ai.Manager
}

type session struct {
	mu      sync.Mutex
	state   model.DiscoverySession
	cancel  context.CancelFunc
	files   map[string]*fileEntry
	profile *projectProfile
}

// NewManager creates a discovery manager backed by the shared AI service.
func NewManager(aiManager *ai.Manager) *Manager {
	return &Manager{sessions: map[string]*session{}, ai: aiManager}
}

// ListDirectory exposes the project browser. Excluded directories are flagged
// without being read, so the user can see why they are skipped.
func (m *Manager) ListDirectory(dir string, opts model.DiscoveryOptions) (*model.DiscoveryDirListing, error) {
	if len(opts.ExcludePatterns) == 0 {
		opts.ExcludePatterns = DefaultExcludes()
	}
	return ListDirectory(dir, opts)
}

// Start launches a discovery run in the background and returns its session ID.
// Progress is delivered through emit, which app.go forwards as a Wails event.
func (m *Manager) Start(opts model.DiscoveryOptions, settings model.AppSettings, emit func(model.DiscoveryProgress)) (string, error) {
	opts = mergeDefaults(opts)
	if strings.TrimSpace(opts.Directory) == "" {
		return "", fmt.Errorf("select a project directory first")
	}
	root, err := filepath.Abs(opts.Directory)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	ctx, cancel := context.WithCancel(context.Background())
	s := &session{
		state: model.DiscoverySession{
			ID: id, Status: "running", Options: opts, Root: root,
			StartedAt: time.Now().UnixMilli(),
		},
		cancel: cancel,
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	go m.run(ctx, s, settings, emit)
	return id, nil
}

// Get returns a snapshot of a session.
func (m *Manager) Get(id string) (*model.DiscoverySession, error) {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil {
		return nil, fmt.Errorf("discovery session not found")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.state
	// Go marshals nil slices as JSON null; the UI expects arrays even when a
	// project yields no endpoints/files/warnings. Normalise before returning.
	if state.Endpoints == nil {
		state.Endpoints = []model.DiscoveredEndpoint{}
	}
	if state.Files == nil {
		state.Files = []model.DiscoveryFileInfo{}
	}
	if state.Folders == nil {
		state.Folders = []string{}
	}
	if state.Warnings == nil {
		state.Warnings = []string{}
	}
	if state.Languages == nil {
		state.Languages = []string{}
	}
	if state.Frameworks == nil {
		state.Frameworks = []string{}
	}
	return &state, nil
}

// Cancel stops a running session. Long scans observe the context between files
// and AI calls, so cancellation is responsive without leaving partial state.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil {
		return fmt.Errorf("discovery session not found")
	}
	s.mu.Lock()
	if s.state.Status == "running" {
		s.state.Status = "cancelling"
	}
	s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// Remove discards a finished session and its cached files.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s := m.sessions[id]; s != nil {
		if s.cancel != nil {
			s.cancel()
		}
		delete(m.sessions, id)
	}
}

// SourceFile returns a window of a project file for the review screen. Only
// files the scanner already read are available, so excluded and secret files
// cannot be opened through this API.
func (m *Manager) SourceFile(id, relPath string, startLine, endLine int) (*model.DiscoverySourceFile, error) {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil {
		return nil, fmt.Errorf("discovery session not found")
	}
	relPath = strings.TrimPrefix(strings.ReplaceAll(relPath, "\\", "/"), "./")
	file := s.files[relPath]
	if file == nil {
		return nil, fmt.Errorf("file %q was not part of this analysis", relPath)
	}
	language := file.Language
	lines := file.Lines
	if len(lines) == 0 {
		// Content for non-route files is released after analysis; re-read it on
		// demand instead of keeping the whole repository in memory.
		content, err := readProjectFile(s.state.Root, relPath)
		if err != nil {
			return nil, err
		}
		lines = strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	total := len(lines)
	if startLine < 1 {
		startLine = 1
	}
	if startLine > total {
		startLine = total
	}
	if endLine <= 0 || endLine > total {
		endLine = total
	}
	if endLine < startLine {
		endLine = startLine
	}
	if endLine-startLine > 400 {
		endLine = startLine + 400
	}
	content := strings.Join(lines[startLine-1:endLine], "\n")
	return &model.DiscoverySourceFile{
		Path: relPath, Language: language, StartLine: startLine, EndLine: endLine,
		TotalLines: total, Content: content, Truncated: endLine < total,
	}, nil
}

// readProjectFile reads a file inside the project root and rejects paths that
// escape it, so the review screen can never read arbitrary files.
func readProjectFile(root, relPath string) (string, error) {
	cleanRoot := filepath.Clean(root)
	target := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(relPath)))
	if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("file %q is outside the project directory", relPath)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("file %q was not part of this analysis", relPath)
	}
	return string(data), nil
}

// Generate renders reviewed endpoints as an import-ready Postwoman collection.
func (m *Manager) Generate(id string, req model.DiscoveryGenerateRequest) (string, error) {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil {
		return "", fmt.Errorf("discovery session not found")
	}
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	content, err := BuildCollection(&state, req)
	if err != nil {
		return "", err
	}
	if warnings, err := ValidateCollection(content); err != nil {
		return "", err
	} else if len(warnings) > 0 {
		s.mu.Lock()
		s.state.Warnings = appendUniqueStrings(s.state.Warnings, warnings)
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.state.Collection = content
	s.mu.Unlock()
	return content, nil
}

// ValidateCollection checks a generated document against the same parser used
// by the import pipeline, so the review screen cannot offer an invalid import.
func ValidateCollection(content string) ([]string, error) {
	parsed, err := importexportParse(content)
	if err != nil {
		return nil, err
	}
	if len(parsed.Collections) == 0 {
		return nil, fmt.Errorf("generated collection contains no items")
	}
	return parsed.Warnings, nil
}

func mergeDefaults(opts model.DiscoveryOptions) model.DiscoveryOptions {
	defaults := Defaults()
	if opts.CollectionName == "" {
		opts.CollectionName = defaults.CollectionName
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = defaults.MaxFiles
	}
	if opts.MaxFileSizeKB <= 0 {
		opts.MaxFileSizeKB = defaults.MaxFileSizeKB
	}
	if opts.MaxAIFiles <= 0 {
		opts.MaxAIFiles = defaults.MaxAIFiles
	}
	if len(opts.ExcludePatterns) == 0 {
		opts.ExcludePatterns = defaults.ExcludePatterns
	}
	return opts
}
