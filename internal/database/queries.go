package database

import (
	"context"
	"encoding/json"
	"strings"

	"Postwoman/internal/model"
)

// ListAllFolders returns every folder in a workspace, including nested ones.
// The UI builds the tree client-side, so a flat list is the most efficient
// shape to ship over the bridge.
func (r *Repository) ListAllFolders(ctx context.Context, workspaceID string) ([]model.Folder, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT f.id, f.collection_id, f.parent_id, f.name, COALESCE(f.description, ''), f."order",
			f.auth_json, f.events_json, f.created_at, f.updated_at
		FROM folders f
		JOIN collections c ON c.id = f.collection_id
		WHERE c.workspace_id = ?
		ORDER BY f."order", f.name`, workspaceID)
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

// ListAllRequests returns every request in a workspace, including requests in
// nested folders.
func (r *Repository) ListAllRequests(ctx context.Context, workspaceID string) ([]model.Request, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.collection_id, r.folder_id, r.name, COALESCE(r.description, ''), r.method, r.url,
			r.headers_json, r.query_params_json, r.path_variables_json, r.body_json,
			r.auth_json, r.scripts_json, r.settings_json, r."order", r.created_at, r.updated_at
		FROM requests r
		JOIN collections c ON c.id = r.collection_id
		WHERE c.workspace_id = ?
		ORDER BY r."order", r.name`, workspaceID)
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

// GetSettingsModel loads application settings, falling back to defaults for
// missing or corrupt values.
func (r *Repository) GetSettingsModel(ctx context.Context) model.AppSettings {
	settings := model.DefaultAppSettings()
	var stored model.AppSettings
	if err := r.GetSetting(ctx, "app", &stored); err == nil {
		settings = mergeSettings(settings, stored)
	}
	return settings
}

func mergeSettings(base, stored model.AppSettings) model.AppSettings {
	if stored.Theme != "" {
		base.Theme = stored.Theme
	}
	if stored.FontSize > 0 {
		base.FontSize = stored.FontSize
	}
	if stored.EditorFontSize > 0 {
		base.EditorFontSize = stored.EditorFontSize
	}
	if stored.EditorTabSize > 0 {
		base.EditorTabSize = stored.EditorTabSize
	}
	if stored.RequestTimeout > 0 {
		base.RequestTimeout = stored.RequestTimeout
	}
	if stored.HistoryRetentionDays > 0 {
		base.HistoryRetentionDays = stored.HistoryRetentionDays
	}
	if stored.MaxResponseSize > 0 {
		base.MaxResponseSize = stored.MaxResponseSize
	}
	base.SSLVerification = stored.SSLVerification
	base.ScriptExecutionEnabled = stored.ScriptExecutionEnabled
	base.AutoSave = stored.AutoSave
	base.ShowRequestHeaders = stored.ShowRequestHeaders
	base.ShowResponseHeaders = stored.ShowResponseHeaders
	base.FollowRedirects = stored.FollowRedirects
	base.SendNoCacheHeader = stored.SendNoCacheHeader
	base.CompactMode = stored.CompactMode
	base.AIEnabled = stored.AIEnabled
	if stored.AITemperature > 0 {
		base.AITemperature = stored.AITemperature
	}
	if stored.AIMaxTokens > 0 {
		base.AIMaxTokens = stored.AIMaxTokens
	}
	if stored.AITimeoutSeconds > 0 {
		base.AITimeoutSeconds = stored.AITimeoutSeconds
	}
	if stored.AccentColor != "" {
		base.AccentColor = stored.AccentColor
	}
	if stored.EditorFontFamily != "" {
		base.EditorFontFamily = stored.EditorFontFamily
	}
	if stored.UIFont != "" {
		base.UIFont = stored.UIFont
	}
	if stored.LayoutPreset != "" {
		base.LayoutPreset = stored.LayoutPreset
	}
	if stored.AIProvider != "" {
		base.AIProvider = stored.AIProvider
	}
	if stored.AIBaseURL != "" {
		base.AIBaseURL = stored.AIBaseURL
	}
	if stored.AIModel != "" {
		base.AIModel = stored.AIModel
	}
	// The API key is empty until the user configures the assistant; copying it
	// unconditionally would be indistinguishable from "not set", so only an
	// explicit non-empty value replaces the default (which is empty anyway).
	base.AIAPIKey = stored.AIAPIKey
	if stored.AIProviderID != "" {
		base.AIProviderID = stored.AIProviderID
	}
	if len(stored.AIProviders) > 0 {
		base.AIProviders = stored.AIProviders
	}
	// Older installs only stored the flat provider fields; materialise them as
	// the first provider entry so the unified settings screen has something to
	// edit, and keep the legacy selector in sync.
	if len(base.AIProviders) == 0 {
		base.AIProviders = []model.AIProviderConfig{{
			ID:             orSettingDefault(base.AIProvider, "openai"),
			Name:           orSettingDefault(base.AIProvider, "OpenAI"),
			Kind:           orSettingDefault(base.AIProvider, "openai"),
			BaseURL:        base.AIBaseURL,
			APIKey:         base.AIAPIKey,
			Model:          base.AIModel,
			Temperature:    base.AITemperature,
			MaxTokens:      base.AIMaxTokens,
			TimeoutSeconds: base.AITimeoutSeconds,
		}}
	}
	if base.AIProviderID == "" {
		base.AIProviderID = base.AIProviders[0].ID
	}
	active := base.ActiveAIProvider()
	base.AIProvider = active.Kind
	base.AIBaseURL = active.BaseURL
	base.AIModel = active.Model
	if active.Temperature > 0 {
		base.AITemperature = active.Temperature
	}
	if active.MaxTokens > 0 {
		base.AIMaxTokens = active.MaxTokens
	}
	if active.TimeoutSeconds > 0 {
		base.AITimeoutSeconds = active.TimeoutSeconds
	}
	if active.APIKey != "" {
		base.AIAPIKey = active.APIKey
	}
	// --- migrations for changed defaults ------------------------------------
	// v1.3.3 changed the interface size default to 15 px. Sizes that are exactly
	// a previous default (13/16 px) are treated as unset so the new default
	// reaches existing installations; any other size is a deliberate choice.
	if stored.FontSize == 13 || stored.FontSize == 16 {
		base.FontSize = 15
	}
	// v1.3.4 removed the Bold-only "Open Sans" interface face (it forced every
	// normal-weight label into a bold face, distorting the typography) and went
	// back to the platform's natural system sans. Existing installs that were
	// migrated to "open-sans" are moved back; deliberate choices are kept.
	if stored.UIFont == "open-sans" {
		base.UIFont = "system"
	}
	base.Proxy = stored.Proxy
	if stored.DefaultHeaders != nil {
		base.DefaultHeaders = stored.DefaultHeaders
	}
	if stored.KeyboardShortcuts != nil {
		base.KeyboardShortcuts = stored.KeyboardShortcuts
	}
	return base
}

func (r *Repository) SaveSettingsModel(ctx context.Context, settings model.AppSettings) error {
	return r.SetSetting(ctx, "app", settings)
}

func orSettingDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// Search performs a case-insensitive search across collections, folders,
// requests and environments. Results are ranked so exact name matches sort
// first.
func (r *Repository) Search(ctx context.Context, workspaceID, query string) ([]model.SearchResult, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	results := []model.SearchResult{}
	if q == "" {
		return results, nil
	}

	collections, err := r.ListCollections(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	collectionNames := map[string]string{}
	for _, c := range collections {
		collectionNames[c.ID] = c.Info.Name
		if score, match := scoreText(c.Info.Name, q); score > 0 {
			results = append(results, model.SearchResult{
				Type: "collection", ID: c.ID, Name: c.Info.Name, Match: match, Score: score,
			})
		}
	}

	folders, err := r.ListAllFolders(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	folderNames := map[string]string{}
	for _, f := range folders {
		folderNames[f.ID] = f.Name
		if score, match := scoreText(f.Name, q); score > 0 {
			results = append(results, model.SearchResult{
				Type: "folder", ID: f.ID, Name: f.Name, Parent: collectionNames[f.CollectionID], Match: match, Score: score,
			})
		}
	}

	requests, err := r.ListAllRequests(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, req := range requests {
		score, _ := scoreText(req.Name, q)
		if score == 0 {
			score, _ = scoreText(req.Method+" "+req.URL, q)
		}
		if score == 0 {
			continue
		}
		parent := collectionNames[req.CollectionID]
		if req.FolderID != nil {
			parent = folderNames[*req.FolderID]
		}
		results = append(results, model.SearchResult{
			Type: "request", ID: req.ID, Name: req.Name, Parent: parent,
			Match: req.Method + " " + req.URL, Score: score,
		})
	}

	envs, err := r.ListEnvironments(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for _, e := range envs {
		if score, match := scoreText(e.Name, q); score > 0 {
			results = append(results, model.SearchResult{
				Type: "environment", ID: e.ID, Name: e.Name, Match: match, Score: score,
			})
		}
	}

	// Simple insertion sort keeps the result list stable and avoids pulling in
	// a sort dependency for what is at most a few hundred entries.
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	if len(results) > 200 {
		results = results[:200]
	}
	return results, nil
}

func scoreText(text, query string) (int, string) {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, query)
	if idx < 0 {
		return 0, ""
	}
	score := 10
	if idx == 0 {
		score += 20
	}
	if lower == query {
		score += 50
	}
	// Show a short excerpt around the match.
	start := idx - 20
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 40
	if end > len(text) {
		end = len(text)
	}
	return score, text[start:end]
}

// CountsByCollection returns request counts keyed by collection ID. Used by
// the sidebar to show how many requests a collection contains without loading
// them all.
func (r *Repository) CountsByCollection(ctx context.Context, workspaceID string) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.collection_id, COUNT(1)
		FROM requests r
		JOIN collections c ON c.id = r.collection_id
		WHERE c.workspace_id = ?
		GROUP BY r.collection_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		out[id] = count
	}
	return out, rows.Err()
}

// RawSetting returns the JSON representation of a setting key, useful for
// debugging and for forwarding unknown keys to the UI.
func (r *Repository) RawSetting(ctx context.Context, key string) (json.RawMessage, error) {
	var raw string
	if err := r.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?`, key).Scan(&raw); err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}
