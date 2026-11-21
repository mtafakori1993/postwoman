package importexport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"Postwoman/internal/database"
	"Postwoman/internal/model"
)

// Persist writes a parsed document into the repository. It returns the created
// collection IDs and environment IDs. Nothing is written if the whole document
// fails to parse, so a failed import cannot leave a half-populated workspace.
func Persist(ctx context.Context, repo *database.Repository, workspaceID string, parsed *Parsed) ([]string, []string, error) {
	collectionIDs := []string{}
	environmentIDs := []string{}

	for _, ic := range parsed.Collections {
		if ic == nil {
			continue
		}
		info := ic.Info
		if strings.TrimSpace(info.Name) == "" {
			info.Name = "Imported Collection"
		}
		coll, err := repo.CreateCollection(ctx, workspaceID, info, ic.Variables, ic.Auth, ic.Events)
		if err != nil {
			return collectionIDs, environmentIDs, fmt.Errorf("create collection %q: %w", info.Name, err)
		}
		collectionIDs = append(collectionIDs, coll.ID)

		// Create folders in dependency order: a parent always exists before a
		// child because we keep making passes until no folder is left over.
		tempToReal := map[string]string{}
		remaining := append([]model.ImportedFolder{}, ic.Folders...)
		for len(remaining) > 0 {
			progress := false
			next := make([]model.ImportedFolder, 0, len(remaining))
			for _, f := range remaining {
				if f.ParentTempID != "" {
					if _, ok := tempToReal[f.ParentTempID]; !ok {
						next = append(next, f)
						continue
					}
				}
				var parentID *string
				if f.ParentTempID != "" {
					id := tempToReal[f.ParentTempID]
					parentID = &id
				}
				auth := f.Auth
				folder, err := repo.CreateFolder(ctx, coll.ID, parentID, f.Name, f.Description, f.Order, auth, f.Events)
				if err != nil {
					return collectionIDs, environmentIDs, fmt.Errorf("create folder %q: %w", f.Name, err)
				}
				tempToReal[f.TempID] = folder.ID
				progress = true
			}
			if !progress {
				// Remaining folders have a parent that does not exist (for
				// example a malformed document). Attach them to the root and
				// report the problem instead of dropping them.
				for _, f := range next {
					folder, err := repo.CreateFolder(ctx, coll.ID, nil, f.Name, f.Description, f.Order, f.Auth, f.Events)
					if err != nil {
						return collectionIDs, environmentIDs, fmt.Errorf("create folder %q: %w", f.Name, err)
					}
					tempToReal[f.TempID] = folder.ID
					parsed.Warn("Folder %q referenced a missing parent folder and was moved to the collection root", f.Name)
				}
				break
			}
			remaining = next
		}

		for i := range ic.Requests {
			req := &ic.Requests[i]
			req.CollectionID = coll.ID
			if req.FolderID != nil {
				if real, ok := tempToReal[*req.FolderID]; ok {
					req.FolderID = &real
				} else {
					req.FolderID = nil
				}
			}
			if _, err := repo.CreateRequest(ctx, req); err != nil {
				return collectionIDs, environmentIDs, fmt.Errorf("create request %q: %w", req.Name, err)
			}
		}
	}

	for _, ie := range parsed.Environments {
		env, err := repo.CreateEnvironment(ctx, workspaceID, ie.Name, ie.Values)
		if err != nil {
			return collectionIDs, environmentIDs, fmt.Errorf("create environment %q: %w", ie.Name, err)
		}
		environmentIDs = append(environmentIDs, env.ID)
	}

	for _, g := range parsed.Globals {
		if err := repo.SetGlobalVariable(ctx, model.GlobalVariable{
			Key: g.Key, Value: g.Value, Type: orString(g.Type, "default"),
			Description: g.Description, Enabled: g.Enabled,
		}); err != nil {
			return collectionIDs, environmentIDs, fmt.Errorf("set global variable %q: %w", g.Key, err)
		}
	}

	return collectionIDs, environmentIDs, nil
}

// ExportCollection renders a stored collection as a Postman v2.1 document,
// preserving nested folders, requests, scripts, auth and variables.
func ExportCollection(ctx context.Context, repo *database.Repository, collectionID string) ([]byte, error) {
	coll, err := repo.GetCollection(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if coll == nil {
		return nil, fmt.Errorf("collection not found")
	}
	folders, err := repo.ListFolders(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	requests, err := repo.ListRequests(ctx, collectionID)
	if err != nil {
		return nil, err
	}

	// Build the folder tree. Postman items are rendered bottom-up so a folder's
	// exported `item` array is complete before it is placed in its parent.
	sorted := append([]model.Folder{}, folders...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Order < sorted[i].Order {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	folderNodes := map[string]*exportFolder{}
	for _, f := range sorted {
		folderNodes[f.ID] = &exportFolder{
			ID: f.ID, Name: f.Name, Description: f.Description, Order: f.Order,
			Auth: f.Auth, Events: f.Events, Items: []any{},
		}
	}

	// Attach requests to their folders (or the collection root) first.
	rootRequests := []any{}
	for i := range requests {
		req := requests[i]
		item := requestToPostman(&req)
		if req.FolderID != nil && *req.FolderID != "" {
			if folder, ok := folderNodes[*req.FolderID]; ok {
				folder.Items = append(folder.Items, item)
				continue
			}
		}
		rootRequests = append(rootRequests, item)
	}

	childrenOf := map[string][]string{}
	rootFolderIDs := []string{}
	for _, f := range sorted {
		if f.ParentID != nil && *f.ParentID != "" {
			if _, ok := folderNodes[*f.ParentID]; ok {
				childrenOf[*f.ParentID] = append(childrenOf[*f.ParentID], f.ID)
				continue
			}
		}
		rootFolderIDs = append(rootFolderIDs, f.ID)
	}

	var renderFolder func(id string) map[string]any
	renderFolder = func(id string) map[string]any {
		node := folderNodes[id]
		items := make([]any, 0, len(node.Items)+len(childrenOf[id]))
		items = append(items, node.Items...)
		for _, childID := range childrenOf[id] {
			items = append(items, renderFolder(childID))
		}
		node.Items = items
		return node.toItem()
	}

	rootItems := []any{}
	for _, id := range rootFolderIDs {
		rootItems = append(rootItems, renderFolder(id))
	}
	rootItems = append(rootItems, rootRequests...)

	variables := make([]map[string]any, 0, len(coll.Variables))
	for _, v := range coll.Variables {
		entry := map[string]any{
			"key": v.Key, "value": v.Value, "type": orString(v.Type, "default"),
			"enabled": v.Enabled,
		}
		if v.Description != "" {
			entry["description"] = v.Description
		}
		variables = append(variables, entry)
	}
	info := map[string]any{
		"_postman_id": orString(coll.Info.PostmanID, coll.ID),
		"name":        coll.Info.Name,
		"schema":      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}
	if coll.Info.Description != "" {
		info["description"] = coll.Info.Description
	}
	doc := map[string]any{
		"info":     info,
		"item":     rootItems,
		"variable": variables,
		"auth":     authToPostman(coll.Auth),
	}
	if events := eventsToPostman(coll.Events); len(events) > 0 {
		doc["event"] = events
	}
	return json.MarshalIndent(doc, "", "  ")
}

type exportFolder struct {
	ID          string
	Name        string
	Description string
	Order       int
	Auth        *model.Auth
	Events      []model.CollectionEvent
	Items       []any
}

func (f *exportFolder) toItem() map[string]any {
	item := map[string]any{
		"name": f.Name,
		"item": f.Items,
	}
	if f.Description != "" {
		item["description"] = f.Description
	}
	if f.Auth != nil && f.Auth.Type != "" && f.Auth.Type != "noauth" {
		item["auth"] = authToPostman(*f.Auth)
	}
	if events := eventsToPostman(f.Events); len(events) > 0 {
		item["event"] = events
	}
	return item
}

func requestToPostman(req *model.Request) map[string]any {
	headers := make([]map[string]any, 0, len(req.Headers))
	for _, h := range req.Headers {
		entry := map[string]any{"key": h.Key, "value": h.Value, "type": "text", "disabled": !h.Enabled}
		if h.Description != "" {
			entry["description"] = h.Description
		}
		headers = append(headers, entry)
	}
	urlObj := map[string]any{"raw": req.URL}
	if len(req.QueryParams) > 0 {
		query := make([]map[string]any, 0, len(req.QueryParams))
		for _, q := range req.QueryParams {
			entry := map[string]any{"key": q.Key, "value": q.Value, "disabled": !q.Enabled}
			if q.Description != "" {
				entry["description"] = q.Description
			}
			query = append(query, entry)
		}
		urlObj["query"] = query
	}
	if len(req.PathVariables) > 0 {
		vars := make([]map[string]any, 0, len(req.PathVariables))
		for _, v := range req.PathVariables {
			vars = append(vars, map[string]any{"key": v.Key, "value": v.Value})
		}
		urlObj["variable"] = vars
	}
	request := map[string]any{
		"method": req.Method,
		"header": headers,
		"url":    urlObj,
		"body":   bodyToPostman(req.Body),
	}
	if req.Auth.Type != "" && req.Auth.Type != "noauth" {
		request["auth"] = authToPostman(req.Auth)
	}
	if req.Description != "" {
		request["description"] = req.Description
	}
	item := map[string]any{
		"name":    req.Name,
		"request": request,
	}
	if req.Description != "" {
		item["description"] = req.Description
	}
	if events := scriptsToPostman(req.Scripts); len(events) > 0 {
		item["event"] = events
	}
	return item
}

func bodyToPostman(body model.Body) map[string]any {
	switch body.Mode {
	case "", "none":
		return map[string]any{"mode": "none"}
	case "json", "xml", "html", "javascript", "text", "raw":
		lang := body.Mode
		if lang == "raw" {
			lang = "text"
		}
		return map[string]any{
			"mode": "raw",
			"raw":  body.Raw,
			"options": map[string]any{
				"raw": map[string]any{"language": lang},
			},
		}
	case "urlencoded":
		return map[string]any{"mode": "urlencoded", "urlencoded": keyValuesToPostman(body.URLEncoded)}
	case "formdata":
		return map[string]any{"mode": "formdata", "formdata": keyValuesToPostman(body.FormData)}
	case "graphql":
		g := map[string]any{"query": "", "variables": "{}"}
		if body.GraphQL != nil {
			g["query"] = body.GraphQL.Query
			if body.GraphQL.Variables != "" {
				g["variables"] = body.GraphQL.Variables
			}
		}
		return map[string]any{"mode": "graphql", "graphql": g}
	case "binary":
		return map[string]any{"mode": "file", "file": map[string]any{"src": body.FileName}}
	}
	return map[string]any{"mode": "none"}
}

func keyValuesToPostman(in []model.KeyValue) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, kv := range in {
		entry := map[string]any{
			"key": kv.Key, "value": kv.Value, "type": orString(kv.Type, "text"),
			"disabled": !kv.Enabled,
		}
		if kv.Description != "" {
			entry["description"] = kv.Description
		}
		if kv.Src != "" {
			entry["src"] = kv.Src
		}
		out = append(out, entry)
	}
	return out
}

func scriptsToPostman(scripts []model.Script) []map[string]any {
	out := []map[string]any{}
	for _, s := range scripts {
		if strings.TrimSpace(s.Script) == "" {
			continue
		}
		exec := strings.Split(strings.ReplaceAll(s.Script, "\r\n", "\n"), "\n")
		out = append(out, map[string]any{
			"listen": s.Type,
			"script": map[string]any{"type": "text/javascript", "exec": exec},
		})
	}
	return out
}

func eventsToPostman(events []model.CollectionEvent) []map[string]any {
	out := []map[string]any{}
	for _, e := range events {
		if len(e.Script.Exec) == 0 {
			continue
		}
		out = append(out, map[string]any{
			"listen": e.Listen,
			"script": map[string]any{"type": "text/javascript", "exec": e.Script.Exec},
		})
	}
	return out
}

func authToPostman(auth model.Auth) map[string]any {
	if auth.Type == "" || auth.Type == "noauth" {
		return map[string]any{"type": "noauth"}
	}
	pairs := []map[string]any{}
	add := func(k, v string) {
		pairs = append(pairs, map[string]any{"key": k, "value": v, "type": "string"})
	}
	switch auth.Type {
	case "apikey":
		if auth.APIKey != nil {
			add("key", auth.APIKey.Key)
			add("value", auth.APIKey.Value)
			add("in", auth.APIKey.AddTo)
		}
	case "bearer":
		if auth.Bearer != nil {
			add("token", auth.Bearer.Token)
		}
	case "basic":
		if auth.Basic != nil {
			add("username", auth.Basic.Username)
			add("password", auth.Basic.Password)
		}
	case "digest":
		if auth.Digest != nil {
			add("username", auth.Digest.Username)
			add("password", auth.Digest.Password)
			add("realm", auth.Digest.Realm)
			add("nonce", auth.Digest.Nonce)
			add("algorithm", auth.Digest.Algorithm)
			add("qop", auth.Digest.Qop)
		}
	case "oauth1":
		if auth.OAuth1 != nil {
			add("consumerKey", auth.OAuth1.ConsumerKey)
			add("consumerSecret", auth.OAuth1.ConsumerSecret)
			add("token", auth.OAuth1.Token)
			add("tokenSecret", auth.OAuth1.TokenSecret)
			add("signatureMethod", auth.OAuth1.SignatureMethod)
			add("version", auth.OAuth1.Version)
			add("realm", auth.OAuth1.Realm)
		}
	case "oauth2":
		if auth.OAuth2 != nil {
			add("grant_type", auth.OAuth2.GrantType)
			add("accessToken", auth.OAuth2.AccessToken)
			add("refreshToken", auth.OAuth2.RefreshToken)
			add("clientId", auth.OAuth2.ClientID)
			add("clientSecret", auth.OAuth2.ClientSecret)
			add("authUrl", auth.OAuth2.AuthURL)
			add("accessTokenUrl", auth.OAuth2.TokenURL)
			add("scope", auth.OAuth2.Scope)
			add("redirect_uri", auth.OAuth2.RedirectURL)
		}
	case "awsv4":
		if auth.AWSV4 != nil {
			add("accessKey", auth.AWSV4.AccessKey)
			add("secretKey", auth.AWSV4.SecretKey)
			add("region", auth.AWSV4.Region)
			add("service", auth.AWSV4.Service)
			add("sessionToken", auth.AWSV4.SessionToken)
		}
	case "jwt":
		if auth.JWT != nil {
			add("token", auth.JWT.Token)
		}
	case "hawk":
		if auth.Hawk != nil {
			add("authId", auth.Hawk.ID)
			add("authKey", auth.Hawk.Key)
			add("algorithm", auth.Hawk.Algorithm)
			add("user", auth.Hawk.User)
		}
	case "ntlm":
		if auth.NTLM != nil {
			add("username", auth.NTLM.Username)
			add("password", auth.NTLM.Password)
			add("domain", auth.NTLM.Domain)
			add("workstation", auth.NTLM.Workstation)
		}
	}
	return map[string]any{"type": auth.Type, auth.Type: pairs}
}

// ExportEnvironment renders an environment as a Postman environment file.
func ExportEnvironment(ctx context.Context, repo *database.Repository, environmentID string) ([]byte, error) {
	env, err := repo.GetEnvironment(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	if env == nil {
		return nil, fmt.Errorf("environment not found")
	}
	values := make([]map[string]any, 0, len(env.Values))
	for _, v := range env.Values {
		values = append(values, map[string]any{
			"key": v.Key, "value": v.Value, "type": orString(v.Type, "default"),
			"enabled": v.Enabled, "description": v.Description,
		})
	}
	payload := map[string]any{
		"id":                      env.ID,
		"name":                    env.Name,
		"values":                  values,
		"_postman_variable_scope": "environment",
		"_postman_exported_at":    time.Now().UTC().Format(time.RFC3339),
		"_postman_exported_using": "Postwoman/1.0",
	}
	return json.MarshalIndent(payload, "", "  ")
}
