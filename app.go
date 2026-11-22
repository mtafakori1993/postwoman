package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"Postwoman/internal/ai"
	"Postwoman/internal/collections"
	"Postwoman/internal/database"
	"Postwoman/internal/discovery"
	"Postwoman/internal/httpclient"
	"Postwoman/internal/importexport"
	"Postwoman/internal/jsonutil"
	"Postwoman/internal/mockserver"
	"Postwoman/internal/model"
	"Postwoman/internal/scripting"
	"Postwoman/internal/variables"
	"Postwoman/internal/websocket"
)

// appVersion is the fallback version for local development builds. Release
// builds inject the repository VERSION through -ldflags "-X main.appVersion=…"
// so the application, installer and archives all report the same value.
var appVersion = "1.3.3"

// App is the Wails-bound backend. Every exported method is callable from the
// React frontend through the generated bindings in frontend/wailsjs/go/main.
type App struct {
	ctx           context.Context
	db            *database.DB
	repo          *database.Repository
	httpClient    *httpclient.Client
	scriptEngine  *scripting.Engine
	collectionMgr *collections.Manager
	wsManager     *websocket.Manager
	mockManager   *mockserver.Manager
	aiManager     *ai.Manager
	discoveryMgr  *discovery.Manager
	workspaceID   string
	wailsReady    bool

	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc
}

// emit forwards an event to the frontend, but only when running inside the
// Wails runtime. Outside it (e.g. in tests) emitting would call log.Fatalf, so
// the call is skipped instead.
func (a *App) emit(eventName string, data ...interface{}) {
	if !a.wailsReady || a.ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, eventName, data...)
}

func NewApp() *App {
	return &App{cancels: map[string]context.CancelFunc{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})
	a.cancelMu.Lock()
	if a.cancels == nil {
		a.cancels = map[string]context.CancelFunc{}
	}
	a.cancelMu.Unlock()

	dbPath := database.GetDBPath()
	log.Info().Str("path", dbPath).Msg("initializing database")
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}
	a.db = db
	a.repo = database.NewRepository(db)

	client, err := httpclient.NewClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create HTTP client")
	}
	a.httpClient = client
	a.scriptEngine = scripting.NewEngine(10 * time.Second)
	a.collectionMgr = collections.NewManager(a.repo)
	a.wsManager = websocket.NewManager(func(connID string, msg model.WSMessage) {
		a.emit("ws:message", map[string]interface{}{
			"connectionId": connID,
			"message":      msg,
		})
	})
	a.mockManager = mockserver.NewManager()
	a.aiManager = ai.NewManager()
	a.discoveryMgr = discovery.NewManager(a.aiManager)
	a.restoreCookies()

	ws, err := a.repo.EnsureDefaultWorkspace(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise workspace")
	} else {
		a.workspaceID = ws.ID
	}

	a.wailsReady = true
	a.emit("backend:ready", map[string]string{"workspaceId": a.workspaceID, "version": appVersion})
}

func (a *App) shutdown(ctx context.Context) {
	if a.wsManager != nil {
		a.wsManager.CloseAll()
	}
	if a.mockManager != nil {
		a.mockManager.StopAll()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

// Bootstrap returns everything the UI needs to render on startup in one call.
func (a *App) Bootstrap() (*model.AppState, error) {
	ctx := a.ctx
	workspace, err := a.repo.EnsureDefaultWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	a.workspaceID = workspace.ID

	state := &model.AppState{
		Workspace:       *workspace,
		Workspaces:      []model.Workspace{},
		Collections:     []model.Collection{},
		Folders:         []model.Folder{},
		Requests:        []model.Request{},
		Environments:    []model.Environment{},
		GlobalVariables: []model.GlobalVariable{},
		History:         []model.HistoryEntry{},
		Tabs:            []model.Tab{},
		MockServers:     []model.MockServer{},
		Version:         appVersion,
		Platform:        runtime.GOOS,
	}

	workspaces, err := a.repo.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	state.Workspaces = workspaces

	collectionList, err := a.repo.ListCollections(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	state.Collections = collectionList

	folders, err := a.repo.ListAllFolders(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	state.Folders = folders

	requests, err := a.repo.ListAllRequests(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	state.Requests = requests

	envs, err := a.repo.ListEnvironments(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	state.Environments = envs

	globals, err := a.repo.ListGlobalVariables(ctx)
	if err != nil {
		return nil, err
	}
	state.GlobalVariables = globals

	history, err := a.repo.ListHistory(ctx, 200, 0)
	if err == nil {
		state.History = history
	}

	tabs, err := a.repo.ListTabs(ctx)
	if err == nil {
		state.Tabs = tabs
	}

	mockServers, err := a.repo.ListMockServers(ctx)
	if err == nil {
		for i := range mockServers {
			mockServers[i].IsRunning = a.mockManager != nil && a.mockManager.IsRunning(mockServers[i].ID)
		}
		state.MockServers = mockServers
	}

	_, state.ActiveEnvironmentID, _ = a.repo.GetActiveWorkspace(ctx)
	state.Settings = a.repo.GetSettingsModel(ctx).SanitizedForUI()
	return state, nil
}

// ---------------------------------------------------------------------------
// HTTP
// ---------------------------------------------------------------------------

// SendRequest executes a request and runs pre-request/test scripts when
// requested. Network failures are returned inside the response object.
func (a *App) SendRequest(request model.Request, environment, globals, collectionVars map[string]string, runScripts bool, environmentID string) (*model.Response, error) {
	return a.sendRequestCore(a.ctx, request, environment, globals, collectionVars, runScripts, environmentID)
}

// SendRequestWithID is the cancellable variant used by the editor. The UI
// generates the id and can call CancelRequest to abort the in-flight request.
func (a *App) SendRequestWithID(requestID string, request model.Request, environmentID string) (*model.Response, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	child, cancel := context.WithCancel(ctx)
	if requestID != "" {
		a.trackCancel(requestID, cancel)
		defer a.untrackCancel(requestID)
	}
	defer cancel()
	return a.sendRequestCore(child, request, nil, nil, nil, true, environmentID)
}

// CancelRequest aborts an in-flight request created by SendRequestWithID.
func (a *App) CancelRequest(requestID string) {
	a.cancelMu.Lock()
	cancel := a.cancels[requestID]
	a.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *App) trackCancel(id string, cancel context.CancelFunc) {
	a.cancelMu.Lock()
	if a.cancels == nil {
		a.cancels = map[string]context.CancelFunc{}
	}
	if existing, ok := a.cancels[id]; ok {
		existing()
	}
	a.cancels[id] = cancel
	a.cancelMu.Unlock()
}

func (a *App) untrackCancel(id string) {
	a.cancelMu.Lock()
	delete(a.cancels, id)
	a.cancelMu.Unlock()
}

func (a *App) sendRequestCore(ctx context.Context, request model.Request, environment, globals, collectionVars map[string]string, runScripts bool, environmentID string) (*model.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	settings := a.repo.GetSettingsModel(ctx)

	env := copyMap(environment)
	glob := copyMap(globals)
	collVars := copyMap(collectionVars)
	local := map[string]string{}

	// The UI sends only the active environment id, so load the persisted
	// variables server-side. This is also what makes {{baseUrl}}-style
	// collection variables resolve for requests sent from the editor.
	if len(env) == 0 && environmentID != "" {
		if stored, err := a.repo.GetEnvironment(ctx, environmentID); err == nil && stored != nil {
			for _, v := range stored.Values {
				if v.Enabled {
					env[v.Key] = v.Value
				}
			}
		}
	}
	if len(collVars) == 0 && request.CollectionID != "" {
		if coll, err := a.repo.GetCollection(ctx, request.CollectionID); err == nil && coll != nil {
			for _, v := range coll.Variables {
				if v.Enabled {
					collVars[v.Key] = v.Value
				}
			}
		}
	}
	if len(glob) == 0 {
		if gvars, err := a.repo.ListGlobalVariables(ctx); err == nil {
			for _, v := range gvars {
				if v.Enabled {
					glob[v.Key] = v.Value
				}
			}
		}
	}

	// Apply global defaults where the request did not specify them.
	if request.Settings.Timeout <= 0 {
		request.Settings.Timeout = settings.RequestTimeout
	}
	if request.Settings.Timeout <= 0 {
		request.Settings.Timeout = 30000
	}
	if request.Settings.MaxRedirects <= 0 {
		request.Settings.MaxRedirects = 10
	}
	if request.Settings.Encoding == "" {
		request.Settings.Encoding = "auto"
	}
	if !settings.SSLVerification {
		if request.Settings.TLS == nil {
			request.Settings.TLS = &model.TLS{RejectUnauthorized: false}
		} else {
			request.Settings.TLS.RejectUnauthorized = false
		}
	}
	if settings.SendNoCacheHeader {
		found := false
		for _, h := range request.Headers {
			if strings.EqualFold(h.Key, "Cache-Control") {
				found = true
			}
		}
		if !found {
			request.Headers = append(request.Headers, model.KeyValue{Key: "Cache-Control", Value: "no-cache", Enabled: true})
		}
	}

	// Resolve collection/folder auth inheritance before anything executes.
	request.Auth = a.effectiveAuth(ctx, &request)

	runScripts = runScripts && settings.ScriptExecutionEnabled
	response := &model.Response{Tests: []model.TestResult{}, Console: []model.ConsoleEntry{}}

	if runScripts {
		if script := findScript(request.Scripts, "prerequest"); script != "" {
			res, err := a.scriptEngine.Run(script, &scripting.Context{
				Request: request, Environment: env, CollectionVariables: collVars,
				Globals: glob, Variables: local, EventName: "prerequest", RequestCount: 1,
			})
			if res != nil {
				env, collVars, glob, local = res.Environment, res.CollectionVariables, res.Globals, res.Variables
				response.Console = append(response.Console, res.Console...)
			}
			if err != nil {
				response.Console = append(response.Console, model.ConsoleEntry{Level: "error", Args: []string{"pre-request script: " + err.Error()}})
			}
		}
	}

	resolver := variables.New(local, env, collVars, glob)
	result, err := a.httpClient.Do(ctx, &request, resolver)
	if err != nil {
		return nil, err
	}
	result.Console = append(response.Console, result.Console...)
	result.Tests = append(response.Tests, result.Tests...)

	if runScripts && result.Error == "" {
		if script := findScript(request.Scripts, "test"); script != "" {
			res, err := a.scriptEngine.Run(script, &scripting.Context{
				Request: request, Response: result, Environment: env, CollectionVariables: collVars,
				Globals: glob, Variables: local, EventName: "test", RequestCount: 1,
			})
			if res != nil {
				result.Tests = append(result.Tests, res.Tests...)
				result.Console = append(result.Console, res.Console...)
				env, collVars, glob = res.Environment, res.CollectionVariables, res.Globals
			}
			if err != nil {
				result.Tests = append(result.Tests, model.TestResult{Name: "Script execution", Passed: false, Error: err.Error()})
			}
		}
	}

	// Persist variables mutated by scripts.
	if environmentID != "" {
		if err := a.persistEnvironment(ctx, environmentID, env); err != nil {
			log.Warn().Err(err).Msg("failed to persist environment changes")
		}
	}
	for key, value := range glob {
		if globals[key] != value {
			_ = a.repo.SetGlobalVariable(ctx, model.GlobalVariable{Key: key, Value: value, Type: "default", Enabled: true})
		}
	}
	for key, value := range collVars {
		if collectionVars[key] != value && request.CollectionID != "" {
			if c, err := a.repo.GetCollection(ctx, request.CollectionID); err == nil && c != nil {
				found := false
				for i := range c.Variables {
					if c.Variables[i].Key == key {
						c.Variables[i].Value = value
						found = true
					}
				}
				if !found {
					c.Variables = append(c.Variables, model.Variable{Key: key, Value: value, Type: "default", Enabled: true})
				}
				_ = a.repo.UpdateCollection(ctx, c)
			}
		}
	}

	redacted := redactRequest(request)
	if _, err := a.repo.AddHistoryEntry(ctx, redacted, result, result.ResponseTime); err != nil {
		log.Warn().Err(err).Msg("failed to write history entry")
	} else {
		a.emit("history:updated", nil)
	}
	if settings.HistoryRetentionDays > 0 {
		_ = a.repo.TrimHistory(ctx, 5000)
	}
	a.persistCookies()
	return result, nil
}

// effectiveAuth resolves request → folder → collection auth inheritance. A
// request can opt out with type "noauth" (or any explicit mechanism).
func (a *App) effectiveAuth(ctx context.Context, request *model.Request) model.Auth {
	explicit := strings.ToLower(strings.TrimSpace(request.Auth.Type))
	if explicit != "" && explicit != "inherit" {
		return request.Auth
	}
	// Walk the folder chain from the request's folder up to the root. The
	// nearest explicit mechanism wins.
	seen := map[string]bool{}
	if request.FolderID != nil && *request.FolderID != "" {
		folderID := *request.FolderID
		for folderID != "" && !seen[folderID] {
			seen[folderID] = true
			folder, err := a.repo.GetFolder(ctx, folderID)
			if err != nil || folder == nil {
				break
			}
			if auth := folder.Auth; auth != nil {
				t := strings.ToLower(strings.TrimSpace(auth.Type))
				if t != "" && t != "inherit" {
					return *auth
				}
			}
			if folder.ParentID == nil {
				break
			}
			folderID = *folder.ParentID
		}
	}
	if request.CollectionID != "" {
		if collection, err := a.repo.GetCollection(ctx, request.CollectionID); err == nil && collection != nil {
			t := strings.ToLower(strings.TrimSpace(collection.Auth.Type))
			if t != "" && t != "inherit" {
				return collection.Auth
			}
		}
	}
	return model.Auth{Type: "noauth"}
}

// scopesFor loads the variable scopes used by a request. Passing empty maps is
// the normal path from the editor; the backend fills them from SQLite.
type variableScopes struct {
	local       map[string]string
	environment map[string]string
	collection  map[string]string
	globals     map[string]string
}

func (s variableScopes) resolver() *variables.Resolver {
	return variables.New(s.local, s.environment, s.collection, s.globals)
}

func (a *App) scopesFor(ctx context.Context, request *model.Request, environmentID string) variableScopes {
	scopes := variableScopes{
		local:       map[string]string{},
		environment: map[string]string{},
		collection:  map[string]string{},
		globals:     map[string]string{},
	}
	if environmentID != "" {
		if stored, err := a.repo.GetEnvironment(ctx, environmentID); err == nil && stored != nil {
			for _, v := range stored.Values {
				if v.Enabled {
					scopes.environment[v.Key] = v.Value
				}
			}
		}
	}
	if request != nil && request.CollectionID != "" {
		if coll, err := a.repo.GetCollection(ctx, request.CollectionID); err == nil && coll != nil {
			for _, v := range coll.Variables {
				if v.Enabled {
					scopes.collection[v.Key] = v.Value
				}
			}
		}
	}
	if gvars, err := a.repo.ListGlobalVariables(ctx); err == nil {
		for _, v := range gvars {
			if v.Enabled {
				scopes.globals[v.Key] = v.Value
			}
		}
	}
	return scopes
}

// SendRequestStructured is a convenience wrapper for clients that prefer a
// single object argument.
func (a *App) SendRequestStructured(input model.SendRequestInput) (*model.Response, error) {
	return a.SendRequest(input.Request, input.Environment, input.Globals, input.CollectionVariables, input.RunScripts, input.EnvironmentID)
}

// ResolveVariables exposes the variable resolver so the UI can preview URLs.
func (a *App) ResolveVariables(input string, scopes map[string]string) string {
	return variables.New(scopes).Resolve(input)
}

func findScript(scripts []model.Script, typ string) string {
	for _, s := range scripts {
		if !s.Disabled && s.Type == typ && strings.TrimSpace(s.Script) != "" {
			return s.Script
		}
	}
	return ""
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (a *App) persistEnvironment(ctx context.Context, environmentID string, values map[string]string) error {
	env, err := a.repo.GetEnvironment(ctx, environmentID)
	if err != nil || env == nil {
		return err
	}
	index := map[string]int{}
	for i, v := range env.Values {
		index[v.Key] = i
	}
	for k, v := range values {
		if i, ok := index[k]; ok {
			env.Values[i].Value = v
		} else {
			env.Values = append(env.Values, model.Variable{Key: k, Value: v, Enabled: true, Type: "default"})
		}
	}
	return a.repo.UpdateEnvironment(ctx, env)
}

func redactRequest(req model.Request) model.Request {
	req.Auth = model.Auth{Type: req.Auth.Type}
	sensitive := map[string]bool{
		"authorization": true, "proxy-authorization": true, "cookie": true,
		"x-api-key": true, "api-key": true, "x-auth-token": true,
	}
	for i := range req.Headers {
		if sensitive[strings.ToLower(req.Headers[i].Key)] {
			req.Headers[i].Value = "••••••"
		}
	}
	for i := range req.QueryParams {
		key := strings.ToLower(req.QueryParams[i].Key)
		if strings.Contains(key, "token") || strings.Contains(key, "secret") || key == "api_key" || key == "apikey" {
			req.QueryParams[i].Value = "••••••"
		}
	}
	return req
}

// ---------------------------------------------------------------------------
// Collections, folders and requests
// ---------------------------------------------------------------------------

func (a *App) CreateCollection(workspaceID, name, description string) (*model.Collection, error) {
	if workspaceID == "" {
		workspaceID = a.workspaceID
	}
	c, err := a.collectionMgr.CreateCollection(a.ctx, workspaceID, name, description)
	if err != nil {
		return nil, err
	}
	a.emit("collections:updated", nil)
	return c, nil
}

func (a *App) UpdateCollection(collection model.Collection) error {
	if collection.Auth.Type == "" {
		collection.Auth.Type = "noauth"
	}
	if err := a.repo.UpdateCollection(a.ctx, &collection); err != nil {
		return err
	}
	a.emit("collections:updated", nil)
	return nil
}

func (a *App) DeleteCollection(id string) error {
	if err := a.repo.DeleteCollection(a.ctx, id); err != nil {
		return err
	}
	a.emit("collections:updated", nil)
	return nil
}

func (a *App) GetCollections(workspaceID string) ([]model.Collection, error) {
	if workspaceID == "" {
		workspaceID = a.workspaceID
	}
	return a.repo.ListCollections(a.ctx, workspaceID)
}

func (a *App) GetCollection(id string) (*model.Collection, error) {
	return a.repo.GetCollection(a.ctx, id)
}

func (a *App) DuplicateCollection(id string) (*model.Collection, error) {
	source, err := a.repo.GetCollection(a.ctx, id)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("collection not found")
	}
	copyInfo := source.Info
	copyInfo.Name += " Copy"
	copyInfo.PostmanID = ""
	created, err := a.repo.CreateCollection(a.ctx, source.WorkspaceID, copyInfo, source.Variables, source.Auth, source.Events)
	if err != nil {
		return nil, err
	}

	folders, _ := a.repo.ListFolders(a.ctx, id)
	requests, _ := a.repo.ListRequests(a.ctx, id)
	folderMap := map[string]string{}
	for _, f := range folders {
		var parent *string
		if f.ParentID != nil {
			if mapped, ok := folderMap[*f.ParentID]; ok {
				parent = &mapped
			}
		}
		createdFolder, err := a.repo.CreateFolder(a.ctx, created.ID, parent, f.Name, f.Description, f.Order, f.Auth, f.Events)
		if err != nil {
			return nil, err
		}
		folderMap[f.ID] = createdFolder.ID
	}
	for _, req := range requests {
		req.ID = ""
		req.CollectionID = created.ID
		if req.FolderID != nil {
			if mapped, ok := folderMap[*req.FolderID]; ok {
				req.FolderID = &mapped
			}
		}
		if _, err := a.repo.CreateRequest(a.ctx, &req); err != nil {
			return nil, err
		}
	}
	return created, nil
}

func (a *App) CreateFolder(collectionID, parentID, name, description string) (*model.Folder, error) {
	if strings.TrimSpace(name) == "" {
		name = "New Folder"
	}
	var parent *string
	if parentID != "" {
		parent = &parentID
	}
	return a.repo.CreateFolder(a.ctx, collectionID, parent, name, description, 0, nil, nil)
}

func (a *App) UpdateFolder(folder model.Folder) error {
	return a.repo.UpdateFolder(a.ctx, &folder)
}

func (a *App) DeleteFolder(id string) error {
	return a.repo.DeleteFolder(a.ctx, id)
}

func (a *App) GetFolders(collectionID string) ([]model.Folder, error) {
	if collectionID == "" {
		return a.repo.ListAllFolders(a.ctx, a.workspaceID)
	}
	return a.repo.ListFolders(a.ctx, collectionID)
}

func (a *App) CreateRequest(request model.Request) (*model.Request, error) {
	if request.CollectionID == "" {
		return nil, fmt.Errorf("request must belong to a collection")
	}
	if strings.TrimSpace(request.Name) == "" {
		request.Name = "Untitled Request"
	}
	request.ID = ""
	return a.repo.CreateRequest(a.ctx, &request)
}

func (a *App) UpdateRequest(request model.Request) error {
	if request.ID == "" {
		return fmt.Errorf("request ID is required")
	}
	return a.repo.UpdateRequest(a.ctx, &request)
}

func (a *App) GetRequest(id string) (*model.Request, error) {
	return a.repo.GetRequest(a.ctx, id)
}

func (a *App) GetRequests(collectionID, folderID string) ([]model.Request, error) {
	if collectionID == "" {
		return a.repo.ListAllRequests(a.ctx, a.workspaceID)
	}
	if folderID != "" {
		return a.repo.ListRequestsInFolder(a.ctx, collectionID, folderID)
	}
	return a.repo.ListRequests(a.ctx, collectionID)
}

func (a *App) DeleteRequest(id string) error {
	return a.repo.DeleteRequest(a.ctx, id)
}

func (a *App) DuplicateRequest(id string) (*model.Request, error) {
	return a.repo.DuplicateRequest(a.ctx, id)
}

// ---------------------------------------------------------------------------
// Import / export
// ---------------------------------------------------------------------------

func (a *App) ImportCollection(workspaceID, content string) (*model.ImportSummary, error) {
	if workspaceID == "" {
		workspaceID = a.workspaceID
	}
	summary, err := a.collectionMgr.Import(a.ctx, workspaceID, []byte(content), "import.json")
	if err != nil {
		return nil, err
	}
	a.emit("collections:updated", nil)
	return summary, nil
}

func (a *App) ImportFromFile(workspaceID string) (*model.ImportSummary, error) {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Import collection, environment, OpenAPI or HAR",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "API files (*.json, *.har, *.txt)", Pattern: "*.json;*.har;*.txt"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil || path == "" {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return a.ImportCollection(workspaceID, string(content))
}

func (a *App) ExportCollection(id string) (string, error) {
	data, err := a.collectionMgr.ExportCollection(a.ctx, id)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) ExportCollectionToFile(id string) (string, error) {
	collection, err := a.repo.GetCollection(a.ctx, id)
	if err != nil || collection == nil {
		return "", fmt.Errorf("collection not found")
	}
	data, err := a.collectionMgr.ExportCollection(a.ctx, id)
	if err != nil {
		return "", err
	}
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export collection",
		DefaultFilename: safeFilename(collection.Info.Name) + ".postman_collection.json",
		Filters:         []wailsRuntime.FileFilter{{DisplayName: "Postman collection", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

func (a *App) ExportEnvironment(id string) (string, error) {
	data, err := a.collectionMgr.ExportEnvironment(a.ctx, id)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) ExportEnvironmentToFile(id string) (string, error) {
	env, err := a.repo.GetEnvironment(a.ctx, id)
	if err != nil || env == nil {
		return "", fmt.Errorf("environment not found")
	}
	data, err := a.collectionMgr.ExportEnvironment(a.ctx, id)
	if err != nil {
		return "", err
	}
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export environment",
		DefaultFilename: safeFilename(env.Name) + ".postman_environment.json",
		Filters:         []wailsRuntime.FileFilter{{DisplayName: "Postman environment", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

func (a *App) ExportRequestCurl(request model.Request) string {
	return collections.ExportRequestAsCurl(request)
}

// ---------------------------------------------------------------------------
// Environments and variables
// ---------------------------------------------------------------------------

func (a *App) CreateEnvironment(workspaceID, name string, values []model.Variable) (*model.Environment, error) {
	if workspaceID == "" {
		workspaceID = a.workspaceID
	}
	if strings.TrimSpace(name) == "" {
		name = "New Environment"
	}
	return a.repo.CreateEnvironment(a.ctx, workspaceID, name, values)
}

func (a *App) UpdateEnvironment(env model.Environment) error {
	return a.repo.UpdateEnvironment(a.ctx, &env)
}

func (a *App) DeleteEnvironment(id string) error {
	return a.repo.DeleteEnvironment(a.ctx, id)
}

func (a *App) GetEnvironments(workspaceID string) ([]model.Environment, error) {
	if workspaceID == "" {
		workspaceID = a.workspaceID
	}
	return a.repo.ListEnvironments(a.ctx, workspaceID)
}

func (a *App) GetEnvironment(id string) (*model.Environment, error) {
	return a.repo.GetEnvironment(a.ctx, id)
}

func (a *App) SetActiveEnvironment(workspaceID, environmentID string) error {
	if workspaceID == "" {
		workspaceID = a.workspaceID
	}
	return a.repo.SetActiveWorkspace(a.ctx, workspaceID, environmentID)
}

func (a *App) GetActiveEnvironment() (*model.Environment, error) {
	_, envID, err := a.repo.GetActiveWorkspace(a.ctx)
	if err != nil || envID == "" {
		return nil, err
	}
	return a.repo.GetEnvironment(a.ctx, envID)
}

func (a *App) SetGlobalVariable(variable model.GlobalVariable) error {
	if strings.TrimSpace(variable.Key) == "" {
		return fmt.Errorf("variable key is required")
	}
	return a.repo.SetGlobalVariable(a.ctx, variable)
}

func (a *App) DeleteGlobalVariable(key string) error {
	return a.repo.DeleteGlobalVariable(a.ctx, key)
}

func (a *App) GetGlobalVariables() ([]model.GlobalVariable, error) {
	return a.repo.ListGlobalVariables(a.ctx)
}

// ---------------------------------------------------------------------------
// History
// ---------------------------------------------------------------------------

func (a *App) GetHistory(limit, offset int) ([]model.HistoryEntry, error) {
	return a.repo.ListHistory(a.ctx, limit, offset)
}

func (a *App) ClearHistory() error {
	return a.repo.ClearHistory(a.ctx)
}

func (a *App) DeleteHistoryEntry(id string) error {
	return a.repo.DeleteHistoryEntry(a.ctx, id)
}

// ---------------------------------------------------------------------------
// Settings and tabs
// ---------------------------------------------------------------------------

func (a *App) GetSettings() model.AppSettings {
	return a.repo.GetSettingsModel(a.ctx).SanitizedForUI()
}

func (a *App) SaveSettings(settings model.AppSettings) error {
	// The UI never receives API keys, so merge stored credentials back in and
	// honour an explicit "clear key" request before persisting.
	stored := a.repo.GetSettingsModel(a.ctx)
	merged := settings.MergeAIKeys(stored)
	if err := a.repo.SaveSettingsModel(a.ctx, merged); err != nil {
		return err
	}
	return nil
}

func (a *App) GetSetting(key string) (interface{}, error) {
	all, err := a.repo.ListSettings(a.ctx)
	if err != nil {
		return nil, err
	}
	return all[key], nil
}

func (a *App) SetSetting(key string, value interface{}) error {
	return a.repo.SetSetting(a.ctx, key, value)
}

func (a *App) SaveTabs(tabs []model.Tab) error {
	return a.repo.SaveTabs(a.ctx, tabs)
}

func (a *App) GetTabs() ([]model.Tab, error) {
	return a.repo.ListTabs(a.ctx)
}

// ---------------------------------------------------------------------------
// Collection runner
// ---------------------------------------------------------------------------

func (a *App) RunCollection(opts model.RunOptions) (*model.RunResult, error) {
	ctx := a.ctx
	collection, err := a.repo.GetCollection(ctx, opts.CollectionID)
	if err != nil {
		return nil, err
	}
	if collection == nil {
		return nil, fmt.Errorf("collection not found")
	}
	requests, err := a.repo.ListRequests(ctx, opts.CollectionID)
	if err != nil {
		return nil, err
	}
	if opts.FolderID != "" {
		filtered := requests[:0]
		for _, r := range requests {
			if r.FolderID != nil && *r.FolderID == opts.FolderID {
				filtered = append(filtered, r)
			}
		}
		requests = filtered
	}
	if len(opts.RequestIDs) > 0 {
		allowed := map[string]bool{}
		for _, id := range opts.RequestIDs {
			allowed[id] = true
		}
		filtered := requests[:0]
		for _, r := range requests {
			if allowed[r.ID] {
				filtered = append(filtered, r)
			}
		}
		requests = filtered
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("the collection has no requests to run")
	}

	iterations := opts.Iterations
	if iterations <= 0 {
		iterations = 1
	}
	delay := time.Duration(opts.Delay) * time.Millisecond

	collectionVars := map[string]string{}
	for _, v := range collection.Variables {
		if v.Enabled {
			collectionVars[v.Key] = v.Value
		}
	}
	globals := map[string]string{}
	if rows, err := a.repo.ListGlobalVariables(ctx); err == nil {
		for _, g := range rows {
			if g.Enabled {
				globals[g.Key] = g.Value
			}
		}
	}
	env := map[string]string{}
	envID := opts.EnvironmentID
	if envID == "" {
		_, envID, _ = a.repo.GetActiveWorkspace(ctx)
	}
	if envID != "" {
		if row, err := a.repo.GetEnvironment(ctx, envID); err == nil && row != nil {
			for _, v := range row.Values {
				if v.Enabled {
					env[v.Key] = v.Value
				}
			}
		}
	}

	dataRows, dataErr := loadDataFile(opts.DataFile)
	if dataErr != nil {
		return nil, dataErr
	}

	settings := a.repo.GetSettingsModel(ctx)
	result := &model.RunResult{
		ID:           fmt.Sprintf("run-%d", time.Now().UnixNano()),
		CollectionID: opts.CollectionID,
		StartedAt:    time.Now().UnixMilli(),
		Status:       "running",
		Iterations:   []model.IterationResult{},
	}
	requestCount := 0
	for iteration := 1; iteration <= iterations; iteration++ {
		iter := model.IterationResult{Iteration: iteration, StartedAt: time.Now().UnixMilli()}
		iterationEnv := copyMap(env)
		if len(dataRows) > 0 {
			row := dataRows[(iteration-1)%len(dataRows)]
			for k, v := range row {
				iterationEnv[k] = v
			}
		}
		for _, request := range requests {
			requestCount++
			request.Auth = a.effectiveAuth(ctx, &request)
			started := time.Now()
			run := model.RequestResult{RequestID: request.ID, RequestName: request.Name, Status: "passed", Tests: []model.TestResult{}}
			local := map[string]string{}

			if script := findScript(request.Scripts, "prerequest"); script != "" && settings.ScriptExecutionEnabled {
				res, serr := a.scriptEngine.Run(script, &scripting.Context{
					Request: request, Environment: iterationEnv, CollectionVariables: collectionVars,
					Globals: globals, Variables: local, EventName: "prerequest", Iteration: iteration, RequestCount: requestCount,
				})
				if res != nil {
					iterationEnv, collectionVars, globals = res.Environment, res.CollectionVariables, res.Globals
				}
				if serr != nil {
					run.Status = "error"
					run.Error = "pre-request script: " + serr.Error()
				}
			}
			resolver := variables.New(local, iterationEnv, collectionVars, globals)
			resp, sendErr := a.httpClient.Do(ctx, &request, resolver)
			run.Duration = time.Since(started).Milliseconds()
			if sendErr != nil {
				run.Status = "error"
				run.Error = sendErr.Error()
			} else {
				run.Response = resp
				if resp.Error != "" {
					run.Status = "error"
					run.Error = resp.Error
				}
				if script := findScript(request.Scripts, "test"); script != "" && settings.ScriptExecutionEnabled {
					res, serr := a.scriptEngine.Run(script, &scripting.Context{
						Request: request, Response: resp, Environment: iterationEnv, CollectionVariables: collectionVars,
						Globals: globals, Variables: local, EventName: "test", Iteration: iteration, RequestCount: requestCount,
					})
					if res != nil {
						run.Tests = res.Tests
						iterationEnv, collectionVars, globals = res.Environment, res.CollectionVariables, res.Globals
					}
					if serr != nil {
						run.Tests = append(run.Tests, model.TestResult{Name: "Script execution", Passed: false, Error: serr.Error()})
					}
				}
			}
			for _, t := range run.Tests {
				result.TotalTests++
				if t.Passed {
					result.PassedTests++
				} else {
					result.FailedTests++
					run.Status = "failed"
				}
			}
			iter.Requests = append(iter.Requests, run)
			if opts.StopOnError && run.Status == "error" {
				break
			}
			if delay > 0 {
				time.Sleep(delay)
			}
		}
		iter.CompletedAt = time.Now().UnixMilli()
		result.Iterations = append(result.Iterations, iter)
	}
	result.TotalRequests = requestCount
	result.CompletedAt = time.Now().UnixMilli()
	result.Status = "completed"
	return result, nil
}

// loadDataFile reads a JSON array of objects or a CSV file used for data-driven
// collection runs.
func loadDataFile(path string) ([]map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read data file: %w", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, fmt.Errorf("data file is not a JSON array of objects: %w", err)
		}
		out := make([]map[string]string, 0, len(rows))
		for _, row := range rows {
			flat := map[string]string{}
			for k, v := range row {
				flat[k] = fmt.Sprintf("%v", v)
			}
			out = append(out, flat)
		}
		return out, nil
	}
	reader := csv.NewReader(strings.NewReader(trimmed))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("data file is not valid CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV data file needs a header row and at least one data row")
	}
	header := records[0]
	out := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := map[string]string{}
		for i, key := range header {
			if i < len(record) {
				row[key] = record[i]
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

func (a *App) Search(query string) ([]model.SearchResult, error) {
	return a.repo.Search(a.ctx, a.workspaceID, query)
}

// ---------------------------------------------------------------------------
// WebSocket
// ---------------------------------------------------------------------------

func (a *App) WSConnect(url string, headers []model.KeyValue, protocols []string) (*model.WSConnection, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("WebSocket URL is required")
	}
	settings := a.repo.GetSettingsModel(a.ctx)
	connID := uuid.New().String()
	open, err := a.wsManager.Connect(a.ctx, connID, url, headers, protocols, settings.SSLVerification)
	if err != nil {
		return nil, err
	}
	return &model.WSConnection{
		ID: connID, URL: url, Headers: headers, Protocols: protocols,
		Connected: true, Messages: []model.WSMessage{*open}, CreatedAt: time.Now().UnixMilli(),
	}, nil
}

func (a *App) WSSend(connectionID, message string, isBinary bool) error {
	_, err := a.wsManager.Send(connectionID, message, isBinary)
	return err
}

func (a *App) WSDisconnect(connectionID string) error {
	return a.wsManager.Close(connectionID)
}

func (a *App) WSGet(connectionID string) (*model.WSConnection, error) {
	for _, conn := range a.wsManager.List() {
		if conn.ID == connectionID {
			return &conn, nil
		}
	}
	return &model.WSConnection{ID: connectionID, Connected: false, Messages: a.wsManager.Messages(connectionID)}, nil
}

func (a *App) WSList() []model.WSConnection {
	return a.wsManager.List()
}

// ---------------------------------------------------------------------------
// Mock servers
// ---------------------------------------------------------------------------

// ListMockServers returns every configured mock server with its live running
// state filled in.
func (a *App) ListMockServers() ([]model.MockServer, error) {
	servers, err := a.repo.ListMockServers(a.ctx)
	if err != nil {
		return nil, err
	}
	for i := range servers {
		servers[i].IsRunning = a.mockManager != nil && a.mockManager.IsRunning(servers[i].ID)
	}
	return servers, nil
}

// SaveMockServer creates or updates a mock server definition.
func (a *App) SaveMockServer(server model.MockServer) (*model.MockServer, error) {
	if strings.TrimSpace(server.Name) == "" {
		server.Name = "Mock Server"
	}
	if server.Port <= 0 {
		server.Port = 8787
	}
	if server.ID == "" {
		if err := a.repo.CreateMockServer(a.ctx, &server); err != nil {
			return nil, err
		}
	} else {
		if err := a.repo.UpdateMockServer(a.ctx, &server); err != nil {
			return nil, err
		}
	}
	server.IsRunning = a.mockManager != nil && a.mockManager.IsRunning(server.ID)
	return &server, nil
}

func (a *App) DeleteMockServer(id string) error {
	if a.mockManager != nil {
		_ = a.mockManager.Stop(id)
	}
	return a.repo.DeleteMockServer(a.ctx, id)
}

func (a *App) StartMockServer(id string) error {
	servers, err := a.repo.ListMockServers(a.ctx)
	if err != nil {
		return err
	}
	for _, s := range servers {
		if s.ID == id {
			if err := a.mockManager.Start(a.ctx, s); err != nil {
				return err
			}
			return a.repo.SetMockServerRunning(a.ctx, id, true)
		}
	}
	return fmt.Errorf("mock server not found")
}

func (a *App) StopMockServer(id string) error {
	if err := a.mockManager.Stop(id); err != nil {
		return err
	}
	return a.repo.SetMockServerRunning(a.ctx, id, false)
}

func (a *App) MockServerLogs(id string) []model.MockRequestLog {
	if a.mockManager == nil {
		return []model.MockRequestLog{}
	}
	return a.mockManager.Logs(id)
}

func (a *App) ClearMockServerLogs(id string) {
	if a.mockManager != nil {
		a.mockManager.ClearLogs(id)
	}
}

// ---------------------------------------------------------------------------
// Dialogs, files and misc
// ---------------------------------------------------------------------------

func (a *App) OpenFileDialog(title, pattern string) (string, error) {
	if pattern == "" {
		pattern = "*"
	}
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   title,
		Filters: []wailsRuntime.FileFilter{{DisplayName: "Supported files", Pattern: pattern}},
	})
}

// OpenDirectoryDialog prompts the user to pick a project directory. It is used
// by the API discovery agent instead of making users type absolute paths.
func (a *App) OpenDirectoryDialog(title string) (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{Title: title})
}

func (a *App) SaveFileDialog(title, defaultName string) (string, error) {
	return wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
	})
}

func (a *App) ReadTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) WriteTextFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func (a *App) SaveTextFile(defaultName, content string) (string, error) {
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Save file",
		DefaultFilename: defaultName,
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, []byte(content), 0o644)
}

func (a *App) CopyToClipboard(text string) error {
	wailsRuntime.ClipboardSetText(a.ctx, text)
	return nil
}

func (a *App) ShowNotification(title, message string) {
	a.emit("notification", map[string]string{"title": title, "message": message})
}

func (a *App) OpenExternalURL(url string) {
	wailsRuntime.BrowserOpenURL(a.ctx, url)
}

func (a *App) GetAppVersion() string { return appVersion }

func (a *App) GetPlatform() string { return runtime.GOOS }

// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

func (a *App) CreateWorkspace(name, description string) (*model.Workspace, error) {
	if strings.TrimSpace(name) == "" {
		name = "New Workspace"
	}
	ws, err := a.repo.CreateWorkspace(a.ctx, name, description)
	if err != nil {
		return nil, err
	}
	if err := a.repo.SetActiveWorkspace(a.ctx, ws.ID, ""); err == nil {
		a.workspaceID = ws.ID
	}
	return ws, nil
}

func (a *App) GetWorkspaces() ([]model.Workspace, error) {
	return a.repo.ListWorkspaces(a.ctx)
}

func (a *App) SetActiveWorkspace(workspaceID string) error {
	if err := a.repo.SetActiveWorkspace(a.ctx, workspaceID, ""); err != nil {
		return err
	}
	a.workspaceID = workspaceID
	return nil
}

func (a *App) GetActiveWorkspaceID() (string, error) {
	return a.workspaceID, nil
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "export"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "", "\"", "", "<", "", ">", "", "|", "-")
	name = replacer.Replace(name)
	if len(name) > 80 {
		name = name[:80]
	}
	return name
}

// ---------------------------------------------------------------------------
// Cookies
// ---------------------------------------------------------------------------

// ListCookies returns cookies captured by the shared jar, grouped by origin.
func (a *App) ListCookies() ([]model.StoredCookie, error) {
	if a.httpClient == nil {
		return []model.StoredCookie{}, nil
	}
	return a.httpClient.ListCookies(), nil
}

// SaveCookie creates or replaces a cookie for an origin.
func (a *App) SaveCookie(input model.StoredCookie) error {
	if a.httpClient == nil {
		return fmt.Errorf("HTTP client is not initialised")
	}
	if err := a.httpClient.SetCookie(input.URL, input.Cookie); err != nil {
		return err
	}
	a.persistCookies()
	return nil
}

// DeleteCookie removes a cookie by name for an origin.
func (a *App) DeleteCookie(rawURL, name string) error {
	if a.httpClient == nil {
		return fmt.Errorf("HTTP client is not initialised")
	}
	if err := a.httpClient.DeleteCookie(rawURL, name); err != nil {
		return err
	}
	a.persistCookies()
	return nil
}

// ClearCookies wipes the jar and the persisted cookie snapshot.
func (a *App) ClearCookies() error {
	if a.httpClient == nil {
		return nil
	}
	if err := a.httpClient.ClearCookies(); err != nil {
		return err
	}
	a.persistCookies()
	return nil
}

func (a *App) persistCookies() {
	if a.httpClient == nil || a.repo == nil || a.ctx == nil {
		return
	}
	if err := a.repo.SetSetting(a.ctx, "cookies", a.httpClient.ListCookies()); err != nil {
		log.Warn().Err(err).Msg("failed to persist cookies")
	}
}

func (a *App) restoreCookies() {
	if a.httpClient == nil || a.repo == nil || a.ctx == nil {
		return
	}
	var stored []model.StoredCookie
	if err := a.repo.GetSetting(a.ctx, "cookies", &stored); err != nil {
		return
	}
	if len(stored) > 0 {
		a.httpClient.RestoreCookies(stored)
	}
}

// ---------------------------------------------------------------------------
// Code generation and cURL import
// ---------------------------------------------------------------------------

// ExportRequestCode renders the request (with inherited auth applied) as code
// for cURL, raw HTTP, fetch, Node.js, Python, Go or PHP.
func (a *App) ExportRequestCode(request model.Request, target string) (string, error) {
	request.Auth = a.effectiveAuth(a.ctx, &request)
	return importexport.ExportAsCode(&request, target)
}

// ParseCurl converts a cURL command into an editable request draft.
func (a *App) ParseCurl(command string) (*model.Request, error) {
	return importexport.ParseCurlAsRequest(command)
}

// ---------------------------------------------------------------------------
// JSON response tooling
// ---------------------------------------------------------------------------

func (a *App) ExtractJSONPath(body, path string) (*model.ExtractionResult, error) {
	return jsonutil.Extract(body, path)
}

func (a *App) CompareJSON(left, right string) (*model.JSONDiff, error) {
	return jsonutil.Diff(left, right)
}

func (a *App) GenerateJSONSchema(body string) (string, error) {
	value, err := jsonutil.Decode(body)
	if err != nil {
		return "", fmt.Errorf("response is not valid JSON: %w", err)
	}
	return jsonutil.MarshalSchema(jsonutil.InferSchema(value))
}

func (a *App) ValidateJSONSchema(body, schema string) ([]string, error) {
	return jsonutil.Validate(body, schema)
}

// ---------------------------------------------------------------------------
// OAuth 2.0 interactive flows
// ---------------------------------------------------------------------------

func oauth2FromRequest(request model.Request) (*model.OAuth2Auth, error) {
	if request.Auth.OAuth2 == nil {
		return nil, fmt.Errorf("the request does not use OAuth 2.0 authentication")
	}
	copy := *request.Auth.OAuth2
	return &copy, nil
}

// OAuth2Start builds the authorization URL (with PKCE when enabled) and returns
// the transient auth state the UI must keep on the request.
func (a *App) OAuth2Start(request model.Request, environmentID string) (*model.OAuth2StartResult, error) {
	original, err := oauth2FromRequest(request)
	if err != nil {
		return nil, err
	}
	scopes := a.scopesFor(a.ctx, &request, environmentID)
	resolved := httpclient.ResolveOAuth2(*original, scopes.resolver())
	started, authURL, err := a.httpClient.StartOAuth2(resolved)
	if err != nil {
		return nil, err
	}
	original.PKCE = started.PKCE
	original.CodeVerifier = started.CodeVerifier
	original.State = started.State
	return &model.OAuth2StartResult{Auth: original, URL: authURL}, nil
}

// OAuth2Exchange exchanges an authorization code (or the pasted redirect URL)
// for tokens.
func (a *App) OAuth2Exchange(request model.Request, environmentID, code string) (*model.OAuth2Auth, error) {
	original, err := oauth2FromRequest(request)
	if err != nil {
		return nil, err
	}
	scopes := a.scopesFor(a.ctx, &request, environmentID)
	resolved := httpclient.ResolveOAuth2(*original, scopes.resolver())
	updated, err := a.httpClient.ExchangeOAuth2Code(a.ctx, resolved, code)
	if err != nil {
		return nil, err
	}
	original.AccessToken = updated.AccessToken
	if updated.RefreshToken != "" {
		original.RefreshToken = updated.RefreshToken
	}
	original.TokenType = updated.TokenType
	original.ExpiresAt = updated.ExpiresAt
	original.CodeVerifier = ""
	original.State = ""
	return original, nil
}

// OAuth2FetchToken acquires a token for the non-interactive grants.
func (a *App) OAuth2FetchToken(request model.Request, environmentID string) (*model.OAuth2Auth, error) {
	original, err := oauth2FromRequest(request)
	if err != nil {
		return nil, err
	}
	scopes := a.scopesFor(a.ctx, &request, environmentID)
	resolved := httpclient.ResolveOAuth2(*original, scopes.resolver())
	updated, err := a.httpClient.FetchOAuth2Token(a.ctx, resolved)
	if err != nil {
		return nil, err
	}
	original.AccessToken = updated.AccessToken
	if updated.RefreshToken != "" {
		original.RefreshToken = updated.RefreshToken
	}
	original.TokenType = updated.TokenType
	original.ExpiresAt = updated.ExpiresAt
	return original, nil
}

// ---------------------------------------------------------------------------
// GraphQL
// ---------------------------------------------------------------------------

const graphqlIntrospectionQuery = `query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      kind
      name
      description
      fields(includeDeprecated: false) {
        name
        description
        args { name description type { kind name ofType { kind name ofType { kind name } } } }
        type { kind name ofType { kind name ofType { kind name } } }
      }
      inputFields { name description type { kind name ofType { kind name ofType { kind name } } } }
      enumValues { name description }
    }
  }
}`

// GraphQLIntrospect executes the introspection query for a request without
// recording history, returning the raw JSON schema document.
func (a *App) GraphQLIntrospect(request model.Request, environmentID string) (string, error) {
	scopes := a.scopesFor(a.ctx, &request, environmentID)
	request.Auth = a.effectiveAuth(a.ctx, &request)
	if request.Settings.Timeout <= 0 {
		request.Settings.Timeout = a.repo.GetSettingsModel(a.ctx).RequestTimeout
	}
	if request.Settings.Timeout <= 0 {
		request.Settings.Timeout = 30000
	}
	request.Settings.MaxRedirects = 10
	request.Body = model.Body{Mode: "graphql", GraphQL: &model.GraphQLBody{
		Query:     graphqlIntrospectionQuery,
		Variables: "{}",
	}}
	ctx, cancel := context.WithTimeout(a.ctx, time.Duration(request.Settings.Timeout)*time.Millisecond)
	defer cancel()
	resp, err := a.httpClient.Do(ctx, &request, scopes.resolver())
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("introspection failed with HTTP %d %s", resp.StatusCode, resp.StatusText)
	}
	return resp.Body, nil
}

// ---------------------------------------------------------------------------
// AI: centralised configuration and shared service layer
// ---------------------------------------------------------------------------

// defaultAISystemPrompt is used by every AI feature unless a caller supplies a
// more specific system prompt.
const defaultAISystemPrompt = "You are an expert API engineer embedded in the Postwoman API client. Be concise, concrete and technically accurate. Use fenced code blocks where useful."

// AIProviderDescriptors lists the supported provider protocols so the settings
// screen can render provider-specific fields without hardcoding them.
func (a *App) AIProviderDescriptors() []model.AIProviderDescriptor {
	return ai.Descriptors()
}

// resolveAIConfig picks the saved global provider unless the caller supplied an
// unsaved configuration, in which case stored credentials are merged in so the
// UI never has to echo keys back.
func (a *App) resolveAIConfig(input model.AIProviderConfig) model.AIProviderConfig {
	settings := a.repo.GetSettingsModel(a.ctx)
	if input.ID == "" && input.Kind == "" && input.BaseURL == "" && input.Model == "" {
		return settings.ActiveAIProvider()
	}
	return ai.MergeProviderSecrets(input, settings)
}

// AITestConnection probes the configured provider and reports connectivity
// without ever returning the API key.
func (a *App) AITestConnection(input model.AIProviderConfig) model.AIConnectionTest {
	cfg := a.resolveAIConfig(input)
	client, err := ai.NewClient(cfg)
	if err != nil {
		return model.AIConnectionTest{OK: false, Provider: cfg.Name, Message: "Configuration invalid: " + err.Error()}
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(a.ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	return client.TestConnection(ctx)
}

// AIListModels returns the model identifiers advertised by the provider.
func (a *App) AIListModels(input model.AIProviderConfig) (model.AIModelList, error) {
	cfg := a.resolveAIConfig(input)
	client, err := ai.NewClient(cfg)
	if err != nil {
		return model.AIModelList{Provider: cfg.Name}, err
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(a.ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	models, err := client.ListModels(ctx)
	if err != nil {
		return model.AIModelList{Provider: cfg.Name, Warning: err.Error()}, nil
	}
	return model.AIModelList{Provider: cfg.Name, Models: models}, nil
}

// AIStatus reports whether AI is usable without issuing a request.
func (a *App) AIStatus() model.AIStatus {
	settings := a.repo.GetSettingsModel(a.ctx)
	cfg := settings.ActiveAIProvider()
	status := model.AIStatus{
		Enabled: settings.AIEnabled, Provider: cfg.Name, Model: cfg.Model,
		Kind: cfg.Kind, BaseURL: cfg.BaseURL,
	}
	if !status.Enabled {
		status.Message = "AI is disabled in Settings → AI"
		return status
	}
	if _, err := ai.NewClient(cfg); err != nil {
		status.Message = err.Error()
		return status
	}
	status.Ready = true
	status.Message = "AI is configured"
	return status
}

// AIAssist runs a chat completion through the shared AI service. Every AI
// feature (assistant, docs, JSON generation, discovery) goes through this layer
// so provider credentials live in exactly one place.
func (a *App) AIAssist(input model.AIRequest) (string, error) {
	settings := a.repo.GetSettingsModel(a.ctx)
	manager := a.aiManager
	if manager == nil {
		manager = ai.NewManager()
	}
	system := strings.TrimSpace(input.SystemPrompt)
	if system == "" {
		system = defaultAISystemPrompt
	}
	userParts := []string{}
	if p := strings.TrimSpace(input.Prompt); p != "" {
		userParts = append(userParts, p)
	}
	if r := strings.TrimSpace(input.RequestJSON); r != "" {
		userParts = append(userParts, "Request configuration:\n```json\n"+r+"\n```")
	}
	if b := strings.TrimSpace(input.ResponseBody); b != "" {
		const max = 20000
		if len(b) > max {
			b = b[:max] + "\n…(response truncated for the assistant)"
		}
		userParts = append(userParts, "Response body:\n```\n"+b+"\n```")
	}
	if len(userParts) == 0 {
		return "", fmt.Errorf("a prompt or request/response context is required")
	}
	timeout := settings.AITimeoutSeconds
	if timeout <= 0 {
		timeout = 120
	}
	ctx, cancel := context.WithTimeout(a.ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	reply, err := manager.Chat(ctx, settings, ai.ChatRequest{
		System:      system,
		Messages:    []ai.Message{{Role: "user", Content: strings.Join(userParts, "\n\n")}},
		Model:       input.Model,
		Temperature: input.Temperature,
		MaxTokens:   input.MaxTokens,
		JSONMode:    input.JSONMode,
	})
	if err != nil {
		return "", err
	}
	return reply.Content, nil
}

// ---------------------------------------------------------------------------
// API Discovery Agent
// ---------------------------------------------------------------------------

// DiscoveryListDirectory returns one level of the project browser. Dependency,
// build and secret locations are flagged so the UI can explain exclusions.
func (a *App) DiscoveryListDirectory(dir string, opts model.DiscoveryOptions) (*model.DiscoveryDirListing, error) {
	if a.discoveryMgr == nil {
		a.discoveryMgr = discovery.NewManager(a.aiManager)
	}
	return a.discoveryMgr.ListDirectory(dir, opts)
}

// DiscoveryStart begins a background analysis and streams progress events over
// the "discovery:progress" channel. It returns the session id immediately.
func (a *App) DiscoveryStart(opts model.DiscoveryOptions) (string, error) {
	if a.discoveryMgr == nil {
		a.discoveryMgr = discovery.NewManager(a.aiManager)
	}
	settings := a.repo.GetSettingsModel(a.ctx)
	return a.discoveryMgr.Start(opts, settings, func(progress model.DiscoveryProgress) {
		a.emit("discovery:progress", progress)
	})
}

// DiscoveryGet returns the current session state, including findings and the
// generated collection preview.
func (a *App) DiscoveryGet(sessionID string) (*model.DiscoverySession, error) {
	if a.discoveryMgr == nil {
		return nil, fmt.Errorf("discovery agent is not initialised")
	}
	return a.discoveryMgr.Get(sessionID)
}

// DiscoveryCancel stops a running analysis.
func (a *App) DiscoveryCancel(sessionID string) error {
	if a.discoveryMgr == nil {
		return fmt.Errorf("discovery agent is not initialised")
	}
	return a.discoveryMgr.Cancel(sessionID)
}

// DiscoverySourceFile returns a window of a project file for the review screen.
func (a *App) DiscoverySourceFile(sessionID, filePath string, startLine, endLine int) (*model.DiscoverySourceFile, error) {
	if a.discoveryMgr == nil {
		return nil, fmt.Errorf("discovery agent is not initialised")
	}
	return a.discoveryMgr.SourceFile(sessionID, filePath, startLine, endLine)
}

// DiscoveryGenerate re-renders the reviewed endpoints as a Postwoman collection.
func (a *App) DiscoveryGenerate(req model.DiscoveryGenerateRequest) (string, error) {
	if a.discoveryMgr == nil {
		return "", fmt.Errorf("discovery agent is not initialised")
	}
	return a.discoveryMgr.Generate(req.SessionID, req)
}

// DiscoveryValidate checks a collection JSON document against the import parser.
func (a *App) DiscoveryValidate(collectionJSON string) ([]string, error) {
	return discovery.ValidateCollection(collectionJSON)
}

// DiscoveryImport imports the reviewed endpoints as a native collection.
func (a *App) DiscoveryImport(req model.DiscoveryGenerateRequest) (*model.ImportSummary, error) {
	if a.discoveryMgr == nil {
		return nil, fmt.Errorf("discovery agent is not initialised")
	}
	collection, err := a.discoveryMgr.Generate(req.SessionID, req)
	if err != nil {
		return nil, err
	}
	workspaceID := a.workspaceID
	if workspaceID == "" {
		if workspace, wsErr := a.repo.EnsureDefaultWorkspace(a.ctx); wsErr == nil {
			workspaceID = workspace.ID
		}
	}
	summary, err := a.collectionMgr.Import(a.ctx, workspaceID, []byte(collection), "discovered-api.json")
	if err != nil {
		return nil, err
	}
	a.emit("collections:updated", nil)
	return summary, nil
}

// ---------------------------------------------------------------------------
// Import from URL
// ---------------------------------------------------------------------------

// ImportFromURL fetches an OpenAPI/Swagger/Postman document over HTTP and
// imports it into the active workspace.
func (a *App) ImportFromURL(workspaceID, sourceURL string) (*model.ImportSummary, error) {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return nil, fmt.Errorf("a document URL is required")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, application/yaml, text/yaml, */*")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("could not fetch %s: HTTP %d", sourceURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, err
	}
	if workspaceID == "" {
		workspaceID = a.workspaceID
	}
	summary, err := a.collectionMgr.Import(ctx, workspaceID, body, pathFilename(sourceURL))
	if err != nil {
		return nil, err
	}
	a.emit("collections:updated", nil)
	return summary, nil
}

func pathFilename(raw string) string {
	if parsed, err := url.Parse(raw); err == nil {
		if name := parsed.Path; name != "" {
			parts := strings.Split(name, "/")
			if last := parts[len(parts)-1]; last != "" {
				return last
			}
		}
	}
	return "import.json"
}

// ---------------------------------------------------------------------------
// WebSocket helpers
// ---------------------------------------------------------------------------

func (a *App) WSPing(connectionID string) error {
	return a.wsManager.Ping(connectionID)
}

func (a *App) WSClear(connectionID string) {
	if a.wsManager != nil {
		a.wsManager.Clear(connectionID)
	}
}
