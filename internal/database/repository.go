package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"Postwoman/internal/model"

	"github.com/google/uuid"
)

// Repository provides database operations.
type Repository struct {
	db *DB
}

func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *DB { return r.db }

func now() int64 { return time.Now().UnixMilli() }

func marshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func ptrString(ns sql.NullString) *string {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	s := ns.String
	return &s
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---------------------------------------------------------------- Workspaces

func (r *Repository) CreateWorkspace(ctx context.Context, name, description string) (*model.Workspace, error) {
	id := uuid.New().String()
	ts := now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, id, name, description, ts, ts)
	if err != nil {
		return nil, err
	}
	return &model.Workspace{ID: id, Name: name, Description: description, CreatedAt: ts, UpdatedAt: ts}, nil
}

func (r *Repository) GetWorkspace(ctx context.Context, id string) (*model.Workspace, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(description, ''), created_at, updated_at FROM workspaces WHERE id = ?`, id)
	var w model.Workspace
	err := row.Scan(&w.ID, &w.Name, &w.Description, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &w, err
}

func (r *Repository) ListWorkspaces(ctx context.Context) ([]model.Workspace, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, COALESCE(description, ''), created_at, updated_at FROM workspaces ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Workspace{}
	for rows.Next() {
		var w model.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateWorkspace(ctx context.Context, id, name, description string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE workspaces SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, now(), id)
	return err
}

func (r *Repository) DeleteWorkspace(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id); err != nil {
		return err
	}
	// If the active workspace was removed, point at another one.
	active, _, err := r.GetActiveWorkspace(ctx)
	if err != nil {
		return err
	}
	if active == id {
		list, err := r.ListWorkspaces(ctx)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			_, err := r.EnsureDefaultWorkspace(ctx)
			return err
		}
		return r.SetActiveWorkspace(ctx, list[0].ID, "")
	}
	return nil
}

// EnsureDefaultWorkspace guarantees at least one workspace exists and is active.
func (r *Repository) EnsureDefaultWorkspace(ctx context.Context) (*model.Workspace, error) {
	list, err := r.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		active, _, err := r.GetActiveWorkspace(ctx)
		if err != nil {
			return nil, err
		}
		if active == "" {
			if err := r.SetActiveWorkspace(ctx, list[0].ID, ""); err != nil {
				return nil, err
			}
		}
		for i := range list {
			if list[i].ID == active {
				return &list[i], nil
			}
		}
		return &list[0], nil
	}
	ws, err := r.CreateWorkspace(ctx, "My Workspace", "Personal workspace")
	if err != nil {
		return nil, err
	}
	if err := r.SetActiveWorkspace(ctx, ws.ID, ""); err != nil {
		return nil, err
	}
	return ws, nil
}

func (r *Repository) SetActiveWorkspace(ctx context.Context, workspaceID, environmentID string) error {
	var env any
	if environmentID != "" {
		env = environmentID
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO active_workspace (id, workspace_id, active_environment_id) VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET workspace_id = excluded.workspace_id, active_environment_id = excluded.active_environment_id`,
		workspaceID, env)
	return err
}

func (r *Repository) GetActiveWorkspace(ctx context.Context) (workspaceID, environmentID string, err error) {
	var env sql.NullString
	row := r.db.QueryRowContext(ctx, `SELECT workspace_id, active_environment_id FROM active_workspace WHERE id = 1`)
	err = row.Scan(&workspaceID, &env)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if env.Valid {
		environmentID = env.String
	}
	return workspaceID, environmentID, err
}

// --------------------------------------------------------------- Collections

func (r *Repository) CreateCollection(ctx context.Context, workspaceID string, info model.CollectionInfo, variables []model.Variable, auth model.Auth, events []model.CollectionEvent) (*model.Collection, error) {
	id := uuid.New().String()
	ts := now()
	if info.Schema == "" {
		info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	}
	if info.PostmanID == "" {
		info.PostmanID = id
	}
	if variables == nil {
		variables = []model.Variable{}
	}
	if events == nil {
		events = []model.CollectionEvent{}
	}
	if auth.Type == "" {
		auth.Type = "noauth"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO collections (id, workspace_id, info_json, variables_json, auth_json, events_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, workspaceID, marshal(info), marshal(variables), marshal(auth), marshal(events), ts, ts)
	if err != nil {
		return nil, err
	}
	return &model.Collection{
		ID: id, WorkspaceID: workspaceID, Info: info, Variables: variables,
		Auth: auth, Events: events, CreatedAt: ts, UpdatedAt: ts,
	}, nil
}

func scanCollection(scan func(dest ...any) error) (*model.Collection, error) {
	var c model.Collection
	var infoJSON, varsJSON, authJSON, eventsJSON string
	if err := scan(&c.ID, &c.WorkspaceID, &infoJSON, &varsJSON, &authJSON, &eventsJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(infoJSON), &c.Info)
	_ = json.Unmarshal([]byte(varsJSON), &c.Variables)
	_ = json.Unmarshal([]byte(authJSON), &c.Auth)
	_ = json.Unmarshal([]byte(eventsJSON), &c.Events)
	if c.Variables == nil {
		c.Variables = []model.Variable{}
	}
	if c.Events == nil {
		c.Events = []model.CollectionEvent{}
	}
	if c.Auth.Type == "" {
		c.Auth.Type = "noauth"
	}
	if c.Info.Schema == "" {
		c.Info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	}
	return &c, nil
}

const collectionCols = `id, workspace_id, info_json, variables_json, COALESCE(auth_json, '{"type":"noauth"}'), COALESCE(events_json, '[]'), created_at, updated_at`

func (r *Repository) GetCollection(ctx context.Context, id string) (*model.Collection, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+collectionCols+` FROM collections WHERE id = ?`, id)
	c, err := scanCollection(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *Repository) ListCollections(ctx context.Context, workspaceID string) ([]model.Collection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+collectionCols+` FROM collections WHERE workspace_id = ? ORDER BY updated_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Collection{}
	for rows.Next() {
		c, err := scanCollection(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateCollection(ctx context.Context, c *model.Collection) error {
	if c.Auth.Type == "" {
		c.Auth.Type = "noauth"
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections SET info_json = ?, variables_json = ?, auth_json = ?, events_json = ?, updated_at = ? WHERE id = ?`,
		marshal(c.Info), marshal(c.Variables), marshal(c.Auth), marshal(c.Events), now(), c.ID)
	return err
}

func (r *Repository) DeleteCollection(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, id)
	return err
}

// ------------------------------------------------------------------- Folders

func (r *Repository) CreateFolder(ctx context.Context, collectionID string, parentID *string, name, description string, order int, auth *model.Auth, events []model.CollectionEvent) (*model.Folder, error) {
	id := uuid.New().String()
	ts := now()
	if events == nil {
		events = []model.CollectionEvent{}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO folders (id, collection_id, parent_id, name, description, "order", auth_json, events_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, collectionID, parentID, name, description, order, marshal(auth), marshal(events), ts, ts)
	if err != nil {
		return nil, err
	}
	return &model.Folder{ID: id, CollectionID: collectionID, ParentID: parentID, Name: name, Description: description, Order: order, Auth: auth, Events: events, CreatedAt: ts, UpdatedAt: ts}, nil
}

func scanFolder(scan func(dest ...any) error) (*model.Folder, error) {
	var f model.Folder
	var parent sql.NullString
	var authJSON, eventsJSON sql.NullString
	if err := scan(&f.ID, &f.CollectionID, &parent, &f.Name, &f.Description, &f.Order, &authJSON, &eventsJSON, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}
	f.ParentID = ptrString(parent)
	if authJSON.Valid && authJSON.String != "" && authJSON.String != "null" {
		var a model.Auth
		if json.Unmarshal([]byte(authJSON.String), &a) == nil {
			f.Auth = &a
		}
	}
	if eventsJSON.Valid && eventsJSON.String != "" && eventsJSON.String != "null" {
		_ = json.Unmarshal([]byte(eventsJSON.String), &f.Events)
	}
	if f.Events == nil {
		f.Events = []model.CollectionEvent{}
	}
	return &f, nil
}

const folderCols = `id, collection_id, parent_id, name, COALESCE(description, ''), "order", auth_json, events_json, created_at, updated_at`

func (r *Repository) GetFolder(ctx context.Context, id string) (*model.Folder, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+folderCols+` FROM folders WHERE id = ?`, id)
	f, err := scanFolder(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return f, err
}

func (r *Repository) ListFolders(ctx context.Context, collectionID string) ([]model.Folder, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+folderCols+` FROM folders WHERE collection_id = ? ORDER BY "order", name`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Folder{}
	for rows.Next() {
		f, err := scanFolder(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateFolder(ctx context.Context, f *model.Folder) error {
	if f.Events == nil {
		f.Events = []model.CollectionEvent{}
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE folders SET parent_id = ?, name = ?, description = ?, "order" = ?, auth_json = ?, events_json = ?, updated_at = ? WHERE id = ?`,
		f.ParentID, f.Name, f.Description, f.Order, marshal(f.Auth), marshal(f.Events), now(), f.ID)
	return err
}

// DeleteFolder removes the folder, its descendants and all requests inside it.
func (r *Repository) DeleteFolder(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Collect descendant ids (including self).
	ids := []string{id}
	frontier := []string{id}
	for len(frontier) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(frontier)), ",")
		args := make([]any, len(frontier))
		for i, v := range frontier {
			args[i] = v
		}
		rows, err := tx.QueryContext(ctx, `SELECT id FROM folders WHERE parent_id IN (`+placeholders+`)`, args...)
		if err != nil {
			return err
		}
		var next []string
		for rows.Next() {
			var child string
			if err := rows.Scan(&child); err != nil {
				rows.Close()
				return err
			}
			next = append(next, child)
		}
		rows.Close()
		ids = append(ids, next...)
		frontier = next
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, v := range ids {
		args[i] = v
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM requests WHERE folder_id IN (`+placeholders+`)`, args...); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM folders WHERE id IN (`+placeholders+`)`, args...); err != nil {
		return err
	}
	return tx.Commit()
}

// ------------------------------------------------------------------ Requests

func normalizeRequest(req *model.Request) {
	if req.Method == "" {
		req.Method = "GET"
	}
	req.Method = strings.ToUpper(req.Method)
	if req.Headers == nil {
		req.Headers = []model.KeyValue{}
	}
	if req.QueryParams == nil {
		req.QueryParams = []model.KeyValue{}
	}
	if req.PathVariables == nil {
		req.PathVariables = []model.KeyValue{}
	}
	if req.Scripts == nil {
		req.Scripts = []model.Script{}
	}
	if req.Body.Mode == "" {
		req.Body.Mode = "none"
	}
	if req.Auth.Type == "" {
		req.Auth.Type = "noauth"
	}
	if req.Settings.Timeout <= 0 {
		req.Settings.Timeout = 30000
	}
	if req.Settings.MaxRedirects <= 0 {
		req.Settings.MaxRedirects = 10
	}
	if req.Settings.Encoding == "" {
		req.Settings.Encoding = "auto"
	}
}

func (r *Repository) CreateRequest(ctx context.Context, req *model.Request) (*model.Request, error) {
	normalizeRequest(req)
	id := uuid.New().String()
	ts := now()
	var folder any
	if req.FolderID != nil && *req.FolderID != "" {
		folder = *req.FolderID
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO requests (id, collection_id, folder_id, name, description, method, url,
			headers_json, query_params_json, path_variables_json, body_json, auth_json, scripts_json, settings_json, "order", created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.CollectionID, folder, req.Name, req.Description, req.Method, req.URL,
		marshal(req.Headers), marshal(req.QueryParams), marshal(req.PathVariables), marshal(req.Body),
		marshal(req.Auth), marshal(req.Scripts), marshal(req.Settings), req.Order, ts, ts)
	if err != nil {
		return nil, err
	}
	req.ID = id
	req.CreatedAt = ts
	req.UpdatedAt = ts
	return req, nil
}

func scanRequest(scan func(dest ...any) error) (*model.Request, error) {
	var req model.Request
	var folder sql.NullString
	var headers, params, pathVars, body, auth, scripts, settings string
	if err := scan(&req.ID, &req.CollectionID, &folder, &req.Name, &req.Description, &req.Method, &req.URL,
		&headers, &params, &pathVars, &body, &auth, &scripts, &settings, &req.Order, &req.CreatedAt, &req.UpdatedAt); err != nil {
		return nil, err
	}
	req.FolderID = ptrString(folder)
	_ = json.Unmarshal([]byte(headers), &req.Headers)
	_ = json.Unmarshal([]byte(params), &req.QueryParams)
	_ = json.Unmarshal([]byte(pathVars), &req.PathVariables)
	_ = json.Unmarshal([]byte(body), &req.Body)
	_ = json.Unmarshal([]byte(auth), &req.Auth)
	_ = json.Unmarshal([]byte(scripts), &req.Scripts)
	_ = json.Unmarshal([]byte(settings), &req.Settings)
	normalizeRequest(&req)
	return &req, nil
}

const requestCols = `id, collection_id, folder_id, name, COALESCE(description, ''), method, url,
	headers_json, query_params_json, path_variables_json, body_json, auth_json, scripts_json, settings_json, "order", created_at, updated_at`

func (r *Repository) GetRequest(ctx context.Context, id string) (*model.Request, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+requestCols+` FROM requests WHERE id = ?`, id)
	req, err := scanRequest(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return req, err
}

func (r *Repository) ListRequests(ctx context.Context, collectionID string) ([]model.Request, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+requestCols+` FROM requests WHERE collection_id = ? ORDER BY "order", name`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Request{}
	for rows.Next() {
		req, err := scanRequest(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *req)
	}
	return out, rows.Err()
}

func (r *Repository) ListRequestsInFolder(ctx context.Context, collectionID, folderID string) ([]model.Request, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+requestCols+` FROM requests WHERE collection_id = ? AND folder_id = ? ORDER BY "order", name`, collectionID, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Request{}
	for rows.Next() {
		req, err := scanRequest(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *req)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateRequest(ctx context.Context, req *model.Request) error {
	normalizeRequest(req)
	var folder any
	if req.FolderID != nil && *req.FolderID != "" {
		folder = *req.FolderID
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE requests SET collection_id = ?, folder_id = ?, name = ?, description = ?, method = ?, url = ?,
			headers_json = ?, query_params_json = ?, path_variables_json = ?, body_json = ?,
			auth_json = ?, scripts_json = ?, settings_json = ?, "order" = ?, updated_at = ?
		WHERE id = ?`,
		req.CollectionID, folder, req.Name, req.Description, req.Method, req.URL,
		marshal(req.Headers), marshal(req.QueryParams), marshal(req.PathVariables), marshal(req.Body),
		marshal(req.Auth), marshal(req.Scripts), marshal(req.Settings), req.Order, now(), req.ID)
	return err
}

func (r *Repository) DeleteRequest(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM requests WHERE id = ?`, id)
	return err
}

func (r *Repository) DuplicateRequest(ctx context.Context, id string) (*model.Request, error) {
	req, err := r.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("request not found")
	}
	req.ID = ""
	req.Name += " Copy"
	req.CreatedAt = 0
	req.UpdatedAt = 0
	return r.CreateRequest(ctx, req)
}

// -------------------------------------------------------------- Environments

func (r *Repository) CreateEnvironment(ctx context.Context, workspaceID, name string, values []model.Variable) (*model.Environment, error) {
	id := uuid.New().String()
	ts := now()
	if values == nil {
		values = []model.Variable{}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO environments (id, workspace_id, name, values_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, id, workspaceID, name, marshal(values), ts, ts)
	if err != nil {
		return nil, err
	}
	return &model.Environment{ID: id, WorkspaceID: workspaceID, Name: name, Values: values, CreatedAt: ts, UpdatedAt: ts}, nil
}

func (r *Repository) GetEnvironment(ctx context.Context, id string) (*model.Environment, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, values_json, created_at, updated_at FROM environments WHERE id = ?`, id)
	var e model.Environment
	var values string
	err := row.Scan(&e.ID, &e.WorkspaceID, &e.Name, &values, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(values), &e.Values)
	if e.Values == nil {
		e.Values = []model.Variable{}
	}
	return &e, nil
}

func (r *Repository) ListEnvironments(ctx context.Context, workspaceID string) ([]model.Environment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, workspace_id, name, values_json, created_at, updated_at FROM environments WHERE workspace_id = ? ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Environment{}
	for rows.Next() {
		var e model.Environment
		var values string
		if err := rows.Scan(&e.ID, &e.WorkspaceID, &e.Name, &values, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(values), &e.Values)
		if e.Values == nil {
			e.Values = []model.Variable{}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateEnvironment(ctx context.Context, e *model.Environment) error {
	if e.Values == nil {
		e.Values = []model.Variable{}
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE environments SET name = ?, values_json = ?, updated_at = ? WHERE id = ?`,
		e.Name, marshal(e.Values), now(), e.ID)
	return err
}

func (r *Repository) DeleteEnvironment(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM environments WHERE id = ?`, id); err != nil {
		return err
	}
	_, envID, err := r.GetActiveWorkspace(ctx)
	if err != nil {
		return err
	}
	if envID == id {
		wsID, _, _ := r.GetActiveWorkspace(ctx)
		return r.SetActiveWorkspace(ctx, wsID, "")
	}
	return nil
}

// ----------------------------------------------------------- Global variables

func (r *Repository) SetGlobalVariable(ctx context.Context, v model.GlobalVariable) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO global_variables (key, value, type, description, enabled, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, type = excluded.type,
			description = excluded.description, enabled = excluded.enabled, updated_at = excluded.updated_at`,
		v.Key, v.Value, v.Type, v.Description, boolInt(v.Enabled), now())
	return err
}

func (r *Repository) ListGlobalVariables(ctx context.Context) ([]model.GlobalVariable, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value, type, COALESCE(description, ''), enabled, updated_at FROM global_variables ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.GlobalVariable{}
	for rows.Next() {
		var v model.GlobalVariable
		var enabled int
		if err := rows.Scan(&v.Key, &v.Value, &v.Type, &v.Description, &enabled, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.Enabled = enabled == 1
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) DeleteGlobalVariable(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM global_variables WHERE key = ?`, key)
	return err
}

// ------------------------------------------------------------------- History

func (r *Repository) AddHistoryEntry(ctx context.Context, req model.Request, resp *model.Response, duration int64) (string, error) {
	id := uuid.New().String()
	ts := now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO request_history (id, request_json, response_json, timestamp, duration)
		VALUES (?, ?, ?, ?, ?)`, id, marshal(req), marshal(resp), ts, duration)
	return id, err
}

func (r *Repository) ListHistory(ctx context.Context, limit, offset int) ([]model.HistoryEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, request_json, response_json, timestamp, duration
		FROM request_history ORDER BY timestamp DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.HistoryEntry{}
	for rows.Next() {
		var h model.HistoryEntry
		var reqJSON, respJSON string
		if err := rows.Scan(&h.ID, &reqJSON, &respJSON, &h.Timestamp, &h.Duration); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(reqJSON), &h.Request)
		if respJSON != "" && respJSON != "null" {
			var resp model.Response
			if json.Unmarshal([]byte(respJSON), &resp) == nil {
				h.Response = &resp
			}
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *Repository) ClearHistory(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM request_history`)
	return err
}

func (r *Repository) DeleteHistoryEntry(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM request_history WHERE id = ?`, id)
	return err
}

func (r *Repository) TrimHistory(ctx context.Context, keep int) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM request_history WHERE id NOT IN (
			SELECT id FROM request_history ORDER BY timestamp DESC LIMIT ?
		)`, keep)
	return err
}

// ---------------------------------------------------------------------- Tabs

func (r *Repository) SaveTabs(ctx context.Context, tabs []model.Tab) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM tabs`); err != nil {
		return err
	}
	ts := now()
	for _, t := range tabs {
		var requestID any
		if t.RequestID != "" {
			requestID = t.RequestID
		}
		var requestJSON, responseJSON any
		if t.Request != nil {
			requestJSON = marshal(t.Request)
		}
		if t.Response != nil {
			responseJSON = marshal(t.Response)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tabs (id, request_id, title, is_dirty, is_active, protocol, request_json, response_json, sidebar_width, response_panel_height, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, requestID, t.Title, boolInt(t.IsDirty), boolInt(t.IsActive), t.Protocol,
			requestJSON, responseJSON, nullInt(t.SidebarWidth), nullInt(t.ResponsePanelHeight), ts, ts); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func (r *Repository) ListTabs(ctx context.Context) ([]model.Tab, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(request_id, ''), title, is_dirty, is_active, protocol, request_json, response_json,
			COALESCE(sidebar_width, 0), COALESCE(response_panel_height, 0), created_at, updated_at
		FROM tabs ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Tab{}
	for rows.Next() {
		var t model.Tab
		var requestJSON, responseJSON sql.NullString
		var dirty, active int
		if err := rows.Scan(&t.ID, &t.RequestID, &t.Title, &dirty, &active, &t.Protocol, &requestJSON, &responseJSON,
			&t.SidebarWidth, &t.ResponsePanelHeight, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.IsDirty = dirty == 1
		t.IsActive = active == 1
		if requestJSON.Valid {
			var req model.Request
			if json.Unmarshal([]byte(requestJSON.String), &req) == nil {
				t.Request = &req
			}
		}
		if responseJSON.Valid {
			var resp model.Response
			if json.Unmarshal([]byte(responseJSON.String), &resp) == nil {
				t.Response = &resp
			}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ------------------------------------------------------------------ Settings

func (r *Repository) SetSetting(ctx context.Context, key string, value any) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, marshal(value), now())
	return err
}

func (r *Repository) GetSetting(ctx context.Context, key string, dest any) error {
	row := r.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?`, key)
	var raw string
	err := row.Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (r *Repository) ListSettings(ctx context.Context) (map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM app_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]any{}
	for rows.Next() {
		var key, raw string
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		var v any
		if json.Unmarshal([]byte(raw), &v) == nil {
			out[key] = v
		}
	}
	return out, rows.Err()
}

// -------------------------------------------------------------- Mock servers

func (r *Repository) ListMockServers(ctx context.Context) ([]model.MockServer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, port, routes_json, is_running, created_at, updated_at FROM mock_servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.MockServer{}
	for rows.Next() {
		var m model.MockServer
		var routes string
		var running int
		if err := rows.Scan(&m.ID, &m.Name, &m.Port, &routes, &running, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.IsRunning = running == 1
		_ = json.Unmarshal([]byte(routes), &m.Routes)
		if m.Routes == nil {
			m.Routes = []model.MockRoute{}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) CreateMockServer(ctx context.Context, m *model.MockServer) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	ts := now()
	if m.Routes == nil {
		m.Routes = []model.MockRoute{}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO mock_servers (id, name, port, routes_json, is_running, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, m.ID, m.Name, m.Port, marshal(m.Routes), boolInt(m.IsRunning), ts, ts)
	m.CreatedAt, m.UpdatedAt = ts, ts
	return err
}

func (r *Repository) UpdateMockServer(ctx context.Context, m *model.MockServer) error {
	if m.Routes == nil {
		m.Routes = []model.MockRoute{}
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE mock_servers SET name = ?, port = ?, routes_json = ?, is_running = ?, updated_at = ? WHERE id = ?`,
		m.Name, m.Port, marshal(m.Routes), boolInt(m.IsRunning), now(), m.ID)
	return err
}

func (r *Repository) DeleteMockServer(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mock_servers WHERE id = ?`, id)
	return err
}

// SetMockServerRunning updates only the running flag.
func (r *Repository) SetMockServerRunning(ctx context.Context, id string, running bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE mock_servers SET is_running = ?, updated_at = ? WHERE id = ?`, boolInt(running), now(), id)
	return err
}
