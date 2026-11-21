package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"Postwoman/internal/collections"
	"Postwoman/internal/database"
	"Postwoman/internal/httpclient"
	"Postwoman/internal/mockserver"
	"Postwoman/internal/model"
	"Postwoman/internal/scripting"
	"Postwoman/internal/variables"
	"Postwoman/internal/websocket"
)

const appVersion = "1.0.0"

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
	workspaceID   string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})

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
		wailsRuntime.EventsEmit(ctx, "ws:message", map[string]interface{}{
			"connectionId": connID,
			"message":      msg,
		})
	})
	a.mockManager = mockserver.NewManager()

	ws, err := a.repo.EnsureDefaultWorkspace(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise workspace")
	} else {
		a.workspaceID = ws.ID
	}

	wailsRuntime.EventsEmit(ctx, "backend:ready", map[string]string{"workspaceId": a.workspaceID, "version": appVersion})
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
	state.Settings = a.repo.GetSettingsModel(ctx)
	return state, nil
}

// ---------------------------------------------------------------------------
// HTTP
// ---------------------------------------------------------------------------

// SendRequest executes a request and runs pre-request/test scripts when
// requested. Network failures are returned inside the response object.
func (a *App) SendRequest(request model.Request, environment, globals, collectionVars map[string]string, runScripts bool, environmentID string) (*model.Response, error) {
	ctx := a.ctx
	settings := a.repo.GetSettingsModel(ctx)

	env := copyMap(environment)
	glob := copyMap(globals)
	collVars := copyMap(collectionVars)
	local := map[string]string{}

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
		wailsRuntime.EventsEmit(ctx, "history:updated", nil)
	}
	if settings.HistoryRetentionDays > 0 {
		_ = a.repo.TrimHistory(ctx, 5000)
	}
	return result, nil
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
	wailsRuntime.EventsEmit(a.ctx, "collections:updated", nil)
	return c, nil
}

func (a *App) UpdateCollection(collection model.Collection) error {
	if collection.Auth.Type == "" {
		collection.Auth.Type = "noauth"
	}
	return a.repo.UpdateCollection(a.ctx, &collection)
}

func (a *App) DeleteCollection(id string) error {
	if err := a.repo.DeleteCollection(a.ctx, id); err != nil {
		return err
	}
	wailsRuntime.EventsEmit(a.ctx, "collections:updated", nil)
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
	wailsRuntime.EventsEmit(a.ctx, "collections:updated", nil)
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
	return a.repo.GetSettingsModel(a.ctx)
}

func (a *App) SaveSettings(settings model.AppSettings) error {
	return a.repo.SaveSettingsModel(a.ctx, settings)
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
	msgs := a.wsManager.Messages(connectionID)
	return &model.WSConnection{
		ID: connectionID, Connected: true, Messages: msgs,
	}, nil
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
	wailsRuntime.EventsEmit(a.ctx, "notification", map[string]string{"title": title, "message": message})
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
