// Package importexport parses and generates API collection formats.
package importexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"Postwoman/internal/model"

	"gopkg.in/yaml.v3"
)

// Parsed is the result of parsing an import document.
type Parsed struct {
	Format       string
	Name         string
	Collections  []*model.ImportedCollection
	Environments []model.ImportedEnvironment
	Globals      []model.Variable
	Warnings     []string
	Raw          string
}

func (p *Parsed) Warn(format string, args ...any) {
	p.Warnings = append(p.Warnings, fmt.Sprintf(format, args...))
}

// Detect guesses the document format.
func Detect(data []byte, filename string) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "unknown"
	}
	lower := strings.ToLower(string(trimmed))
	if strings.HasPrefix(lower, "curl ") || strings.HasPrefix(lower, "curl.exe ") {
		return "curl"
	}
	var doc map[string]any
	if json.Unmarshal(trimmed, &doc) == nil {
		return detectMap(doc)
	}
	if err := yaml.Unmarshal(trimmed, &doc); err == nil {
		if f := detectMap(doc); f != "unknown" {
			return f
		}
	}
	if regexp.MustCompile(`(?m)^\s*(type\s+(Query|Mutation|Subscription)|schema\s*\{)`).Match(trimmed) {
		return "graphql"
	}
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".graphql", ".gql":
		return "graphql"
	case ".yaml", ".yml":
		return "openapi3"
	case ".har":
		return "har"
	case ".sh", ".bash":
		return "curl"
	}
	return "unknown"
}

func detectMap(doc map[string]any) string {
	if doc == nil {
		return "unknown"
	}
	if _, ok := doc["openapi"]; ok {
		return "openapi3"
	}
	if _, ok := doc["swagger"]; ok {
		return "swagger2"
	}
	if _, ok := doc["asyncapi"]; ok {
		return "asyncapi"
	}
	if logObj, ok := doc["log"].(map[string]any); ok {
		if _, ok := logObj["entries"]; ok {
			return "har"
		}
	}
	if _, ok := doc["info"]; ok {
		if _, ok := doc["item"]; ok {
			return "postman"
		}
	}
	if _, ok := doc["_postman_variable_scope"]; ok {
		return "postman-environment"
	}
	if _, ok := doc["values"]; ok {
		if _, ok := doc["name"]; ok {
			return "postman-environment"
		}
	}
	if _, ok := doc["__schema"]; ok {
		return "graphql-introspection"
	}
	if data, ok := doc["data"].(map[string]any); ok {
		if _, ok := data["__schema"]; ok {
			return "graphql-introspection"
		}
	}
	if _, ok := doc["postwoman"]; ok {
		return "native"
	}
	if _, ok := doc["collections"]; ok {
		if _, ok := doc["environments"]; ok {
			return "native"
		}
	}
	return "unknown"
}

// Parse detects and parses a document.
func Parse(data []byte, filename string) (*Parsed, error) {
	format := Detect(data, filename)
	switch format {
	case "postman":
		return parsePostman(data)
	case "postman-environment":
		return parsePostmanEnvironment(data)
	case "openapi3":
		return parseOpenAPI(data, false)
	case "swagger2":
		return parseOpenAPI(data, true)
	case "har":
		return parseHAR(data)
	case "curl":
		return parseCurl(string(data))
	case "graphql":
		return parseGraphQLSDL(string(data))
	case "graphql-introspection":
		return parseGraphQLIntrospection(data)
	case "asyncapi":
		return parseAsyncAPI(data)
	case "native":
		return parseNative(data)
	case "unknown":
		return nil, fmt.Errorf("unrecognised format — supported: Postman 2.0/2.1, Postman environment, OpenAPI 3.x, Swagger 2.0, cURL, HAR, GraphQL, AsyncAPI")
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

// --------------------------------------------------------------- helpers

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func asBool(v any, def bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true"
	case float64:
		return t != 0
	}
	return def
}

func asInt(v any, def int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		if n, err := strconv.Atoi(t); err == nil {
			return n
		}
	}
	return def
}

func description(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if m := asMap(v); m != nil {
		return asString(m["content"])
	}
	return ""
}

func kv(key, value string) model.KeyValue {
	return model.KeyValue{Key: key, Value: value, Enabled: true}
}

// ------------------------------------------------------------- Postman

func parsePostman(data []byte) (*Parsed, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid Postman JSON: %w", err)
	}
	info := asMap(doc["info"])
	if info == nil {
		return nil, fmt.Errorf("Postman collection is missing the info object")
	}
	parsed := &Parsed{Format: "postman", Name: asString(info["name"]), Raw: string(data)}
	coll := &model.ImportedCollection{
		Info: model.CollectionInfo{
			PostmanID:   asString(info["_postman_id"]),
			Name:        asString(info["name"]),
			Description: description(info["description"]),
			Schema:      asString(info["schema"]),
		},
		Auth:   parsePostmanAuth(doc["auth"], parsed),
		Events: parsePostmanCollectionEvents(doc["event"], parsed),
	}
	for _, v := range asSlice(doc["variable"]) {
		if m := asMap(v); m != nil {
			coll.Variables = append(coll.Variables, model.Variable{
				Key: asString(m["key"]), Value: asString(m["value"]), Type: orString(asString(m["type"]), "default"),
				Description: asString(m["description"]), Enabled: asBool(m["enabled"], true),
			})
		}
	}
	if schema := asString(info["schema"]); schema != "" && !strings.Contains(schema, "v2.1") && !strings.Contains(schema, "v2.0") {
		parsed.Warn("Collection schema %q is not a supported Postman v2.x schema; imported on a best-effort basis", schema)
	}
	items, ok := doc["item"].([]any)
	if !ok {
		parsed.Warn("Collection has no items")
		items = []any{}
	}
	parsePostmanItems(items, "", coll, parsed)
	parsed.Collections = []*model.ImportedCollection{coll}
	return parsed, nil
}

func parsePostmanItems(items []any, parentTemp string, coll *model.ImportedCollection, parsed *Parsed) {
	for idx, raw := range items {
		item := asMap(raw)
		if item == nil {
			parsed.Warn("Skipped a non-object item at position %d", idx)
			continue
		}
		name := asString(item["name"])
		if _, isFolder := item["item"]; isFolder {
			tempID := fmt.Sprintf("folder-%d-%s", len(coll.Folders)+1, sanitizeID(name))
			var folderAuth *model.Auth
			if item["auth"] != nil {
				a := parsePostmanAuth(item["auth"], parsed)
				folderAuth = &a
			}
			folder := model.ImportedFolder{
				TempID: tempID, ParentTempID: parentTemp, Name: name,
				Description: description(item["description"]), Order: idx,
				Auth: folderAuth, Events: parsePostmanCollectionEvents(item["event"], parsed),
			}
			coll.Folders = append(coll.Folders, folder)
			parsePostmanItems(asSlice(item["item"]), tempID, coll, parsed)
			continue
		}
		req := parsePostmanRequest(item, parsed)
		req.Name = name
		if req.Name == "" {
			req.Name = "Untitled Request"
		}
		req.Order = idx
		if parentTemp != "" {
			pid := parentTemp
			req.FolderID = &pid
		}
		coll.Requests = append(coll.Requests, *req)
	}
}

func parsePostmanRequest(item map[string]any, parsed *Parsed) *model.Request {
	req := &model.Request{
		Name:        asString(item["name"]),
		Description: description(item["description"]),
		Method:      "GET",
		Body:        model.Body{Mode: "none"},
		Auth:        model.Auth{Type: "inherit"},
		Settings:    model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
	}
	reqData := item["request"]
	switch t := reqData.(type) {
	case string:
		req.URL = t
	case map[string]any:
		req.Method = orString(strings.ToUpper(asString(t["method"])), "GET")
		req.URL, req.QueryParams, req.PathVariables = parsePostmanURL(t["url"], parsed)
		req.Headers = parsePostmanHeaders(t["header"])
		req.Body = parsePostmanBody(t["body"], parsed)
		if a := parsePostmanAuth(t["auth"], parsed); a.Type != "" && a.Type != "noauth" {
			req.Auth = a
		}
		if proto := asMap(t["protocolProfileBehavior"]); proto != nil {
			if v, ok := proto["followRedirects"]; ok {
				req.Settings.FollowRedirects = asBool(v, true)
			}
			if v, ok := proto["disableBodyPruning"]; ok {
				_ = v
			}
		}
	}
	req.Scripts = parsePostmanEvents(item["event"], parsed)
	return req
}

func parsePostmanHeaders(v any) []model.KeyValue {
	out := []model.KeyValue{}
	switch t := v.(type) {
	case []any:
		for _, h := range t {
			m := asMap(h)
			if m == nil {
				continue
			}
			out = append(out, model.KeyValue{
				Key: asString(m["key"]), Value: asString(m["value"]),
				Description: asString(m["description"]), Enabled: !asBool(m["disabled"], false),
			})
		}
	case string:
		for _, line := range strings.Split(t, "\n") {
			line = strings.TrimRight(line, "\r")
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				out = append(out, kv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])))
			}
		}
	}
	return out
}

func parsePostmanURL(v any, parsed *Parsed) (string, []model.KeyValue, []model.KeyValue) {
	switch t := v.(type) {
	case string:
		raw, params := splitRawQuery(t)
		return raw, params, nil
	case map[string]any:
		raw := asString(t["raw"])
		var query []model.KeyValue
		for _, q := range asSlice(t["query"]) {
			m := asMap(q)
			if m == nil {
				continue
			}
			query = append(query, model.KeyValue{
				Key: asString(m["key"]), Value: asString(m["value"]),
				Description: asString(m["description"]), Enabled: !asBool(m["disabled"], false),
			})
		}
		var pathVars []model.KeyValue
		for _, pv := range asSlice(t["variable"]) {
			m := asMap(pv)
			if m == nil {
				continue
			}
			pathVars = append(pathVars, model.KeyValue{
				Key: asString(m["key"]), Value: asString(m["value"]),
				Description: asString(m["description"]), Enabled: true,
			})
		}
		if raw == "" {
			protocol := asString(t["protocol"])
			if protocol == "" {
				protocol = "https"
			}
			host := ""
			switch h := t["host"].(type) {
			case []any:
				var parts []string
				for _, p := range h {
					parts = append(parts, asString(p))
				}
				host = strings.Join(parts, ".")
			case string:
				host = h
			}
			port := asString(t["port"])
			if port != "" {
				host += ":" + port
			}
			var segs []string
			for _, p := range asSlice(t["path"]) {
				segs = append(segs, asString(p))
			}
			raw = protocol + "://" + host + "/" + strings.Join(segs, "/")
		} else {
			raw, _ = splitRawQuery(raw)
		}
		// Derive path variables from :name segments if none declared.
		if len(pathVars) == 0 {
			if u, err := url.Parse(raw); err == nil {
				for _, seg := range strings.Split(u.Path, "/") {
					if strings.HasPrefix(seg, ":") && len(seg) > 1 {
						pathVars = append(pathVars, kv(seg[1:], ""))
					}
				}
			}
		}
		return raw, query, pathVars
	}
	return "", nil, nil
}

func splitRawQuery(raw string) (string, []model.KeyValue) {
	i := strings.Index(raw, "?")
	if i < 0 {
		return raw, nil
	}
	base := raw[:i]
	vals, err := url.ParseQuery(raw[i+1:])
	if err != nil {
		return raw, nil
	}
	var params []model.KeyValue
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range vals[k] {
			params = append(params, kv(k, v))
		}
	}
	return base, params
}

func parsePostmanBody(v any, parsed *Parsed) model.Body {
	m := asMap(v)
	if m == nil {
		return model.Body{Mode: "none"}
	}
	mode := asString(m["mode"])
	body := model.Body{Mode: "none"}
	switch mode {
	case "", "none":
		return body
	case "raw":
		lang := ""
		if opts := asMap(m["options"]); opts != nil {
			lang = asString(asMap(opts["raw"])["language"])
		}
		switch lang {
		case "json":
			body.Mode = "json"
		case "xml":
			body.Mode = "xml"
		case "javascript":
			body.Mode = "javascript"
		case "html":
			body.Mode = "html"
		default:
			body.Mode = "text"
		}
		body.Raw = asString(m["raw"])
	case "urlencoded":
		body.Mode = "urlencoded"
		for _, f := range asSlice(m["urlencoded"]) {
			fm := asMap(f)
			if fm == nil {
				continue
			}
			body.URLEncoded = append(body.URLEncoded, model.KeyValue{
				Key: asString(fm["key"]), Value: asString(fm["value"]),
				Description: asString(fm["description"]), Enabled: !asBool(fm["disabled"], false),
			})
		}
	case "formdata":
		body.Mode = "formdata"
		for _, f := range asSlice(m["formdata"]) {
			fm := asMap(f)
			if fm == nil {
				continue
			}
			field := model.KeyValue{
				Key: asString(fm["key"]), Value: asString(fm["value"]), Type: asString(fm["type"]),
				Description: asString(fm["description"]), Enabled: !asBool(fm["disabled"], false),
			}
			if field.Type == "file" {
				switch src := fm["src"].(type) {
				case string:
					field.Src = src
				case []any:
					if len(src) > 0 {
						field.Src = asString(src[0])
					}
				}
				if field.Src != "" {
					field.FileName = path.Base(field.Src)
				}
			}
			body.FormData = append(body.FormData, field)
		}
	case "graphql":
		g := asMap(m["graphql"])
		body.Mode = "graphql"
		body.GraphQL = &model.GraphQLBody{Query: asString(g["query"]), Variables: asString(g["variables"])}
	case "file":
		f := asMap(m["file"])
		body.Mode = "binary"
		body.FileName = path.Base(asString(f["src"]))
		parsed.Warn("Binary body file %q cannot be embedded; re-select the file in the Body tab", body.FileName)
	default:
		parsed.Warn("Unsupported Postman body mode %q; imported as raw text", mode)
		body.Mode = "text"
		body.Raw = asString(m["raw"])
	}
	return body
}

func parsePostmanEvents(v any, parsed *Parsed) []model.Script {
	var out []model.Script
	for _, e := range asSlice(v) {
		m := asMap(e)
		if m == nil {
			continue
		}
		listen := asString(m["listen"])
		scriptObj := asMap(m["script"])
		exec := ""
		if scriptObj != nil {
			switch ex := scriptObj["exec"].(type) {
			case string:
				exec = ex
			case []any:
				var lines []string
				for _, l := range ex {
					lines = append(lines, asString(l))
				}
				exec = strings.Join(lines, "\n")
			}
		}
		switch listen {
		case "prerequest":
			out = append(out, model.Script{Type: "prerequest", Script: exec})
		case "test":
			out = append(out, model.Script{Type: "test", Script: exec})
		case "preresponse":
			parsed.Warn("Pre-response scripts are not supported and were skipped")
		default:
			if listen != "" {
				parsed.Warn("Unknown script event %q was skipped", listen)
			}
		}
	}
	return out
}

// parsePostmanCollectionEvents converts Postman event scripts into the
// CollectionEvent shape used by collections and folders.
func parsePostmanCollectionEvents(v any, parsed *Parsed) []model.CollectionEvent {
	scripts := parsePostmanEvents(v, parsed)
	out := make([]model.CollectionEvent, 0, len(scripts))
	for _, s := range scripts {
		if strings.TrimSpace(s.Script) == "" {
			continue
		}
		out = append(out, model.CollectionEvent{
			Listen: s.Type,
			Script: model.CollectionScript{
				Type: "text/javascript",
				Exec: strings.Split(s.Script, "\n"),
			},
		})
	}
	return out
}

func parsePostmanAuth(v any, parsed *Parsed) model.Auth {
	m := asMap(v)
	if m == nil {
		return model.Auth{Type: "noauth"}
	}
	typ := strings.ToLower(asString(m["type"]))
	get := func(section, key string) string {
		for _, item := range asSlice(m[section]) {
			im := asMap(item)
			if im == nil {
				continue
			}
			if strings.EqualFold(asString(im["key"]), key) {
				return asString(im["value"])
			}
		}
		return ""
	}
	switch typ {
	case "", "noauth":
		return model.Auth{Type: "noauth"}
	case "basic":
		return model.Auth{Type: "basic", Basic: &model.BasicAuth{Username: get("basic", "username"), Password: get("basic", "password")}}
	case "bearer":
		return model.Auth{Type: "bearer", Bearer: &model.BearerAuth{Token: get("bearer", "token")}}
	case "apikey":
		return model.Auth{Type: "apikey", APIKey: &model.APIKeyAuth{Key: get("apikey", "key"), Value: get("apikey", "value"), AddTo: orString(get("apikey", "in"), "header")}}
	case "digest":
		return model.Auth{Type: "digest", Digest: &model.DigestAuth{Username: get("digest", "username"), Password: get("digest", "password"), Realm: get("digest", "realm")}}
	case "oauth1":
		return model.Auth{Type: "oauth1", OAuth1: &model.OAuth1Auth{
			ConsumerKey: get("oauth1", "consumerKey"), ConsumerSecret: get("oauth1", "consumerSecret"),
			Token: get("oauth1", "token"), TokenSecret: get("oauth1", "tokenSecret"),
			SignatureMethod: orString(get("oauth1", "signatureMethod"), "HMAC-SHA1"),
			Version:         orString(get("oauth1", "version"), "1.0"), Realm: get("oauth1", "realm"),
		}}
	case "oauth2":
		return model.Auth{Type: "oauth2", OAuth2: &model.OAuth2Auth{
			AccessToken: get("oauth2", "accessToken"), RefreshToken: get("oauth2", "refreshToken"),
			ClientID: get("oauth2", "clientId"), ClientSecret: get("oauth2", "clientSecret"),
			AuthURL: get("oauth2", "authUrl"), TokenURL: get("oauth2", "tokenUrl"),
			Scope: get("oauth2", "scope"), RedirectURL: get("oauth2", "redirect_uri"),
			GrantType: orString(get("oauth2", "grant_type"), "authorization_code"),
		}}
	case "awsv4":
		return model.Auth{Type: "awsv4", AWSV4: &model.AWSV4Auth{
			AccessKey: get("awsv4", "accessKey"), SecretKey: get("awsv4", "secretKey"),
			Region: get("awsv4", "region"), Service: get("awsv4", "service"), SessionToken: get("awsv4", "sessionToken"),
		}}
	case "hawk":
		return model.Auth{Type: "hawk", Hawk: &model.HawkAuth{ID: get("hawk", "authId"), Key: get("hawk", "authKey"), Algorithm: get("hawk", "algorithm")}}
	case "ntlm":
		return model.Auth{Type: "ntlm", NTLM: &model.NTLMAuth{Username: get("ntlm", "username"), Password: get("ntlm", "password"), Domain: get("ntlm", "domain"), Workstation: get("ntlm", "workstation")}}
	case "jwt":
		return model.Auth{Type: "jwt", JWT: &model.JWTAuth{Token: get("jwt", "secret")}}
	default:
		parsed.Warn("Unsupported auth type %q; imported as no auth", typ)
		return model.Auth{Type: "noauth"}
	}
}

func parsePostmanEnvironment(data []byte) (*Parsed, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid Postman environment JSON: %w", err)
	}
	parsed := &Parsed{Format: "postman-environment", Name: asString(doc["name"]), Raw: string(data)}
	scope := asString(doc["_postman_variable_scope"])
	var values []model.Variable
	for _, v := range asSlice(doc["values"]) {
		m := asMap(v)
		if m == nil {
			continue
		}
		values = append(values, model.Variable{
			Key: asString(m["key"]), Value: asString(m["value"]), Type: orString(asString(m["type"]), "default"),
			Description: asString(m["description"]), Enabled: asBool(m["enabled"], true),
		})
	}
	if scope == "globals" {
		parsed.Globals = values
	} else {
		parsed.Environments = []model.ImportedEnvironment{{Name: orString(parsed.Name, "Imported Environment"), Values: values, Scope: "environment"}}
	}
	if _, ok := doc["_postman_exported_at"]; !ok {
		parsed.Warn("Environment file has no export metadata; imported anyway")
	}
	return parsed, nil
}

// --------------------------------------------------------------- OpenAPI

func parseOpenAPI(data []byte, swagger bool) (*Parsed, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI/Swagger document: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("empty OpenAPI/Swagger document")
	}
	format := "openapi3"
	if swagger {
		format = "swagger2"
	}
	info := asMap(doc["info"])
	name := asString(info["title"])
	if name == "" {
		name = "Imported API"
	}
	parsed := &Parsed{Format: format, Name: name, Raw: string(data)}
	coll := &model.ImportedCollection{Info: model.CollectionInfo{
		Name: name, Description: description(info["description"]),
		Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}}
	coll.Auth = openAPIGlobalAuth(doc, swagger, parsed)

	baseURL := ""
	if swagger {
		scheme := "https"
		if schemes := asSlice(doc["schemes"]); len(schemes) > 0 {
			scheme = asString(schemes[0])
		}
		baseURL = scheme + "://" + asString(doc["host"]) + asString(doc["basePath"])
	} else {
		if servers := asSlice(doc["servers"]); len(servers) > 0 {
			baseURL = asString(asMap(servers[0])["url"])
			baseURL = substituteServerVars(baseURL, asMap(asMap(servers[0])["variables"]))
		}
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	paths := asMap(doc["paths"])
	if paths == nil {
		parsed.Warn("Document contains no paths")
	}
	tagFolders := map[string]string{}
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	methods := []string{"get", "post", "put", "patch", "delete", "head", "options", "trace"}
	for _, p := range keys {
		item := asMap(paths[p])
		if item == nil {
			continue
		}
		pathLevelParams := asSlice(item["parameters"])
		for _, method := range methods {
			op := asMap(item[method])
			if op == nil {
				continue
			}
			req := &model.Request{
				Name:        orString(asString(op["summary"]), orString(asString(op["operationId"]), strings.ToUpper(method)+" "+p)),
				Description: description(op["description"]), Method: strings.ToUpper(method),
				Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "inherit"},
				Settings: model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
			}
			req.URL = baseURL + p
			applyOpenAPIParams(req, append(append([]any{}, pathLevelParams...), asSlice(op["parameters"])...), swagger, doc)
			applyOpenAPIRequestBody(req, op, doc, swagger, parsed)
			if auth := openAPIOperationAuth(doc, op, swagger); auth != nil {
				req.Auth = *auth
			}
			tags := asSlice(op["tags"])
			if len(tags) > 0 {
				tag := asString(tags[0])
				tempID, ok := tagFolders[tag]
				if !ok {
					tempID = "tag-" + sanitizeID(tag)
					tagFolders[tag] = tempID
					coll.Folders = append(coll.Folders, model.ImportedFolder{TempID: tempID, Name: tag})
				}
				pid := tempID
				req.FolderID = &pid
			}
			coll.Requests = append(coll.Requests, *req)
		}
	}
	if swagger {
		if defs := asMap(doc["definitions"]); defs != nil {
			parsed.Warn("%d Swagger definition(s) were used to generate example bodies but are not imported as models", len(defs))
		}
	} else {
		if comps := asMap(doc["components"]); comps != nil {
			if schemas := asMap(comps["schemas"]); schemas != nil {
				parsed.Warn("%d OpenAPI schema(s) were used to generate example bodies but are not imported as models", len(schemas))
			}
		}
	}
	parsed.Collections = []*model.ImportedCollection{coll}
	return parsed, nil
}

func substituteServerVars(raw string, vars map[string]any) string {
	for k, v := range vars {
		vm := asMap(v)
		def := asString(vm["default"])
		raw = strings.ReplaceAll(raw, "{"+k+"}", def)
	}
	return raw
}

func applyOpenAPIParams(req *model.Request, params []any, swagger bool, doc map[string]any) {
	for _, p := range params {
		pm := asMap(p)
		if pm == nil {
			continue
		}
		name := asString(pm["name"])
		in := asString(pm["in"])
		schema := asMap(pm["schema"])
		if schema == nil && swagger {
			schema = pm
		}
		example := schemaExample(schema, doc, 0)
		val := asString(pm["example"])
		if val == "" && example != nil {
			val = asString(example)
		}
		switch in {
		case "path":
			req.PathVariables = append(req.PathVariables, model.KeyValue{Key: name, Value: val, Description: asString(pm["description"]), Enabled: true})
		case "query":
			req.QueryParams = append(req.QueryParams, model.KeyValue{Key: name, Value: val, Description: asString(pm["description"]), Enabled: asBool(pm["required"], false) || val != ""})
		case "header":
			if strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Content-Type") {
				continue
			}
			req.Headers = append(req.Headers, model.KeyValue{Key: name, Value: val, Description: asString(pm["description"]), Enabled: val != ""})
		case "body":
			if swagger {
				example := schemaExample(schema, doc, 0)
				if example != nil {
					b, _ := json.MarshalIndent(example, "", "  ")
					req.Body = model.Body{Mode: "json", Raw: string(b)}
				}
			}
		}
	}
}

func applyOpenAPIRequestBody(req *model.Request, op map[string]any, doc map[string]any, swagger bool, parsed *Parsed) {
	if swagger {
		return
	}
	rb := asMap(op["requestBody"])
	if rb == nil {
		return
	}
	content := asMap(rb["content"])
	if content == nil {
		return
	}
	// Prefer JSON, then form types, then anything else.
	order := []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}
	var chosen string
	for _, ct := range order {
		if _, ok := content[ct]; ok {
			chosen = ct
			break
		}
	}
	if chosen == "" {
		for ct := range content {
			chosen = ct
			break
		}
	}
	media := asMap(content[chosen])
	schema := asMap(media["schema"])
	example := media["example"]
	if example == nil && schema != nil {
		example = schemaExample(schema, doc, 0)
	}
	switch chosen {
	case "application/json":
		var raw string
		if s, ok := example.(string); ok {
			raw = s
		} else {
			b, _ := json.MarshalIndent(example, "", "  ")
			raw = string(b)
		}
		req.Body = model.Body{Mode: "json", Raw: raw}
	case "application/x-www-form-urlencoded":
		req.Body = model.Body{Mode: "urlencoded"}
		if m := asMap(example); m != nil {
			for k, v := range m {
				req.Body.URLEncoded = append(req.Body.URLEncoded, kv(k, asString(v)))
			}
		}
	case "multipart/form-data":
		req.Body = model.Body{Mode: "formdata"}
		if m := asMap(example); m != nil {
			for k, v := range m {
				req.Body.FormData = append(req.Body.FormData, model.KeyValue{Key: k, Value: asString(v), Enabled: true, Type: "text"})
			}
		}
	default:
		req.Body = model.Body{Mode: "text", Raw: asString(example)}
		parsed.Warn("Request body content type %q imported as raw text", chosen)
	}
	if asBool(rb["required"], false) && req.Body.Mode == "none" {
		parsed.Warn("Required request body for %s %s could not be generated", req.Method, req.URL)
	}
}

func openAPIGlobalAuth(doc map[string]any, swagger bool, parsed *Parsed) model.Auth {
	security := asSlice(doc["security"])
	if len(security) == 0 {
		return model.Auth{Type: "noauth"}
	}
	return securityToAuth(asMap(security[0]), doc, swagger)
}

func openAPIOperationAuth(doc map[string]any, op map[string]any, swagger bool) *model.Auth {
	security := asSlice(op["security"])
	if len(security) == 0 {
		return nil
	}
	a := securityToAuth(asMap(security[0]), doc, swagger)
	return &a
}

func securityToAuth(sec map[string]any, doc map[string]any, swagger bool) model.Auth {
	for schemeName := range sec {
		var schemes map[string]any
		if swagger {
			schemes = asMap(doc["securityDefinitions"])
		} else {
			schemes = asMap(asMap(doc["components"])["securitySchemes"])
		}
		s := asMap(schemes[schemeName])
		if s == nil {
			continue
		}
		typ := strings.ToLower(asString(s["type"]))
		scheme := strings.ToLower(asString(s["scheme"]))
		switch {
		case typ == "http" && scheme == "basic":
			return model.Auth{Type: "basic", Basic: &model.BasicAuth{}}
		case typ == "http" && scheme == "bearer":
			return model.Auth{Type: "bearer", Bearer: &model.BearerAuth{}}
		case typ == "oauth2":
			return model.Auth{Type: "oauth2", OAuth2: &model.OAuth2Auth{GrantType: "authorization_code"}}
		case typ == "apikey":
			addTo := asString(s["in"])
			if addTo == "" {
				addTo = "header"
			}
			return model.Auth{Type: "apikey", APIKey: &model.APIKeyAuth{Key: asString(s["name"]), AddTo: addTo}}
		}
	}
	return model.Auth{Type: "noauth"}
}

func schemaExample(schema map[string]any, doc map[string]any, depth int) any {
	if schema == nil || depth > 8 {
		return nil
	}
	if ref := asString(schema["$ref"]); ref != "" {
		target := resolveRef(doc, ref)
		if target != nil {
			return schemaExample(target, doc, depth+1)
		}
		return map[string]any{}
	}
	if v, ok := schema["example"]; ok {
		return v
	}
	if v, ok := schema["default"]; ok {
		return v
	}
	if enum := asSlice(schema["enum"]); len(enum) > 0 {
		return enum[0]
	}
	for _, comb := range []string{"allOf", "oneOf", "anyOf"} {
		if list := asSlice(schema[comb]); len(list) > 0 {
			if comb == "allOf" {
				merged := map[string]any{}
				for _, item := range list {
					if m := asMap(schemaExample(asMap(item), doc, depth+1)); m != nil {
						for k, v := range m {
							merged[k] = v
						}
					}
				}
				return merged
			}
			return schemaExample(asMap(list[0]), doc, depth+1)
		}
	}
	typ := asString(schema["type"])
	if typ == "" {
		if _, ok := schema["properties"]; ok {
			typ = "object"
		} else if _, ok := schema["items"]; ok {
			typ = "array"
		}
	}
	switch typ {
	case "object":
		out := map[string]any{}
		props := asMap(schema["properties"])
		required := map[string]bool{}
		for _, r := range asSlice(schema["required"]) {
			required[asString(r)] = true
		}
		for k, v := range props {
			// Include all properties to give users a useful starting body.
			out[k] = schemaExample(asMap(v), doc, depth+1)
			_ = required
		}
		return out
	case "array":
		item := schemaExample(asMap(schema["items"]), doc, depth+1)
		if item == nil {
			item = "string"
		}
		return []any{item}
	case "integer":
		return 0
	case "number":
		return 0.0
	case "boolean":
		return true
	case "string":
		switch asString(schema["format"]) {
		case "date":
			return "2024-01-01"
		case "date-time":
			return "2024-01-01T00:00:00Z"
		case "email":
			return "user@example.com"
		case "uuid":
			return "00000000-0000-4000-8000-000000000000"
		case "uri", "url":
			return "https://example.com"
		}
		return "string"
	}
	return nil
}

func resolveRef(doc map[string]any, ref string) map[string]any {
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	var cur any = doc
	for _, p := range parts {
		m := asMap(cur)
		if m == nil {
			return nil
		}
		cur = m[p]
	}
	return asMap(cur)
}

// -------------------------------------------------------------------- HAR

func parseHAR(data []byte) (*Parsed, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid HAR JSON: %w", err)
	}
	logObj := asMap(doc["log"])
	if logObj == nil {
		return nil, fmt.Errorf("HAR file has no log object")
	}
	parsed := &Parsed{Format: "har", Name: "HAR Import", Raw: string(data)}
	coll := &model.ImportedCollection{Info: model.CollectionInfo{
		Name: "HAR Import", Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}}
	hostFolders := map[string]string{}
	for idx, raw := range asSlice(logObj["entries"]) {
		entry := asMap(raw)
		reqData := asMap(entry["request"])
		if reqData == nil {
			continue
		}
		rawURL := asString(reqData["url"])
		req := &model.Request{
			Name:   fmt.Sprintf("%s %s", strings.ToUpper(asString(reqData["method"])), rawURL),
			Method: orString(strings.ToUpper(asString(reqData["method"])), "GET"),
			URL:    rawURL, Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"},
			Settings: model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
			Order:    idx,
		}
		req.URL, req.QueryParams = splitRawQuery(rawURL)
		for _, h := range asSlice(reqData["headers"]) {
			hm := asMap(h)
			if hm == nil {
				continue
			}
			name := asString(hm["name"])
			if strings.HasPrefix(name, ":") || strings.EqualFold(name, "cookie") || strings.EqualFold(name, "content-length") {
				continue
			}
			req.Headers = append(req.Headers, model.KeyValue{Key: name, Value: asString(hm["value"]), Enabled: true})
		}
		if qs := asSlice(reqData["queryString"]); len(qs) > 0 {
			req.QueryParams = nil
			for _, q := range qs {
				qm := asMap(q)
				if qm == nil {
					continue
				}
				req.QueryParams = append(req.QueryParams, kv(asString(qm["name"]), asString(qm["value"])))
			}
		}
		if post := asMap(reqData["postData"]); post != nil {
			mimeType := strings.ToLower(asString(post["mimeType"]))
			switch {
			case strings.Contains(mimeType, "json"):
				req.Body = model.Body{Mode: "json", Raw: asString(post["text"])}
			case strings.Contains(mimeType, "x-www-form-urlencoded"):
				req.Body = model.Body{Mode: "urlencoded"}
				for _, p := range asSlice(post["params"]) {
					pm := asMap(p)
					if pm == nil {
						continue
					}
					req.Body.URLEncoded = append(req.Body.URLEncoded, kv(asString(pm["name"]), asString(pm["value"])))
				}
			case strings.Contains(mimeType, "multipart"):
				req.Body = model.Body{Mode: "formdata"}
				for _, p := range asSlice(post["params"]) {
					pm := asMap(p)
					if pm == nil {
						continue
					}
					field := model.KeyValue{Key: asString(pm["name"]), Value: asString(pm["value"]), Enabled: true, Type: "text"}
					if fn := asString(pm["fileName"]); fn != "" {
						field.Type = "file"
						field.FileName = fn
					}
					req.Body.FormData = append(req.Body.FormData, field)
				}
			default:
				req.Body = model.Body{Mode: "text", Raw: asString(post["text"])}
			}
		}
		// Group by host.
		if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
			host := u.Host
			tempID, ok := hostFolders[host]
			if !ok {
				tempID = "host-" + sanitizeID(host)
				hostFolders[host] = tempID
				coll.Folders = append(coll.Folders, model.ImportedFolder{TempID: tempID, Name: host})
			}
			pid := tempID
			req.FolderID = &pid
		}
		coll.Requests = append(coll.Requests, *req)
	}
	parsed.Collections = []*model.ImportedCollection{coll}
	return parsed, nil
}

// --------------------------------------------------------------- GraphQL

var queryBlockRe = regexp.MustCompile(`(?s)type\s+(Query|Mutation|Subscription)\s*(?:implements[^{]+)?\{([^}]*)\}`)
var fieldRe = regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*(\([^)]*\))?\s*:\s*([\[\]!A-Za-z0-9_]+)`)

func parseGraphQLSDL(text string) (*Parsed, error) {
	parsed := &Parsed{Format: "graphql", Name: "GraphQL Schema", Raw: text}
	coll := &model.ImportedCollection{Info: model.CollectionInfo{
		Name: "GraphQL Schema", Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}}
	blocks := queryBlockRe.FindAllStringSubmatch(text, -1)
	if len(blocks) == 0 {
		parsed.Warn("No Query/Mutation types found; imported the schema as a single GraphQL request")
		coll.Requests = append(coll.Requests, model.Request{
			Name: "GraphQL Request", Method: "POST", URL: "",
			Body:     model.Body{Mode: "graphql", GraphQL: &model.GraphQLBody{Query: text, Variables: "{}"}},
			Auth:     model.Auth{Type: "noauth"},
			Settings: model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
		})
		parsed.Collections = []*model.ImportedCollection{coll}
		return parsed, nil
	}
	for _, block := range blocks {
		kind := block[1]
		fields := fieldRe.FindAllStringSubmatch(block[2], -1)
		for _, f := range fields {
			name := f[1]
			query := fmt.Sprintf("%s {\n  %s\n}", strings.ToLower(kind), name)
			coll.Requests = append(coll.Requests, model.Request{
				Name: kind + " " + name, Method: "POST", URL: "",
				Body:     model.Body{Mode: "graphql", GraphQL: &model.GraphQLBody{Query: query, Variables: "{}"}},
				Auth:     model.Auth{Type: "noauth"},
				Settings: model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
			})
		}
	}
	if len(coll.Requests) == 0 {
		parsed.Warn("Schema parsed but no root fields were found")
	}
	parsed.Collections = []*model.ImportedCollection{coll}
	return parsed, nil
}

func parseGraphQLIntrospection(data []byte) (*Parsed, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid GraphQL introspection JSON: %w", err)
	}
	schema := asMap(doc["__schema"])
	if schema == nil {
		schema = asMap(asMap(doc["data"])["__schema"])
	}
	if schema == nil {
		return nil, fmt.Errorf("GraphQL introspection JSON has no __schema object")
	}
	parsed := &Parsed{Format: "graphql", Name: "GraphQL Schema", Raw: string(data)}
	coll := &model.ImportedCollection{Info: model.CollectionInfo{
		Name: "GraphQL Schema", Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}}
	for _, rawType := range asSlice(schema["types"]) {
		typ := asMap(rawType)
		if typ == nil {
			continue
		}
		name := asString(typ["name"])
		if name != "Query" && name != "Mutation" && name != "Subscription" {
			continue
		}
		for _, rawField := range asSlice(typ["fields"]) {
			field := asMap(rawField)
			if field == nil {
				continue
			}
			fieldName := asString(field["name"])
			query := fmt.Sprintf("%s {\n  %s\n}", strings.ToLower(name), fieldName)
			coll.Requests = append(coll.Requests, model.Request{
				Name: name + " " + fieldName, Method: "POST", URL: "",
				Body:     model.Body{Mode: "graphql", GraphQL: &model.GraphQLBody{Query: query, Variables: "{}"}},
				Auth:     model.Auth{Type: "noauth"},
				Settings: model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
			})
		}
	}
	if len(coll.Requests) == 0 {
		parsed.Warn("Introspection document contained no Query/Mutation fields")
	}
	parsed.Collections = []*model.ImportedCollection{coll}
	return parsed, nil
}

// --------------------------------------------------------------- AsyncAPI

func parseAsyncAPI(data []byte) (*Parsed, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid AsyncAPI document: %w", err)
	}
	info := asMap(doc["info"])
	name := orString(asString(info["title"]), "AsyncAPI Import")
	parsed := &Parsed{Format: "asyncapi", Name: name, Raw: string(data)}
	coll := &model.ImportedCollection{Info: model.CollectionInfo{
		Name: name, Description: description(info["description"]),
		Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}}
	baseURL := ""
	if servers := asMap(doc["servers"]); servers != nil {
		for _, s := range servers {
			sm := asMap(s)
			baseURL = asString(sm["url"])
			if baseURL == "" {
				protocol := asString(sm["protocol"])
				host := asString(sm["host"])
				if protocol != "" && host != "" {
					baseURL = protocol + "://" + host
				}
			}
			if baseURL != "" {
				break
			}
		}
	}
	channels := asMap(doc["channels"])
	keys := make([]string, 0, len(channels))
	for k := range channels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, ch := range keys {
		chObj := asMap(channels[ch])
		if chObj == nil {
			continue
		}
		for _, op := range []string{"publish", "subscribe"} {
			opObj := asMap(chObj[op])
			if opObj == nil {
				continue
			}
			req := &model.Request{
				Name:   orString(asString(opObj["operationId"]), strings.ToUpper(op)+" "+ch),
				Method: "POST", URL: baseURL + ch,
				Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"},
				Settings:    model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
				Description: description(opObj["description"]),
			}
			if msg := asMap(opObj["message"]); msg != nil {
				if payload := asMap(msg["payload"]); payload != nil {
					example := schemaExample(payload, doc, 0)
					b, _ := json.MarshalIndent(example, "", "  ")
					req.Body = model.Body{Mode: "json", Raw: string(b)}
				}
			}
			coll.Requests = append(coll.Requests, *req)
		}
	}
	parsed.Warn("AsyncAPI channels were imported as HTTP requests. WebSocket/MQTT execution is not available for AsyncAPI operations in this build.")
	parsed.Collections = []*model.ImportedCollection{coll}
	return parsed, nil
}

// ---------------------------------------------------------------- native

func parseNative(data []byte) (*Parsed, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid Postwoman export: %w", err)
	}
	parsed := &Parsed{Format: "native", Name: "Postwoman Export", Raw: string(data)}
	for _, raw := range asSlice(doc["collections"]) {
		m := asMap(raw)
		if m == nil {
			continue
		}
		coll := &model.ImportedCollection{}
		if info := asMap(m["info"]); info != nil {
			coll.Info = model.CollectionInfo{Name: asString(info["name"]), Description: description(info["description"]), Schema: asString(info["schema"])}
		}
		if coll.Info.Schema == "" {
			coll.Info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		}
		if m["auth"] != nil {
			_ = json.Unmarshal(mustJSON(m["auth"]), &coll.Auth)
		}
		if vars := asSlice(m["variables"]); len(vars) > 0 {
			_ = json.Unmarshal(mustJSON(vars), &coll.Variables)
		}
		// Folders/requests are stored flat in the native format.
		for _, rf := range asSlice(m["folders"]) {
			var f model.ImportedFolder
			fm := asMap(rf)
			if fm == nil {
				continue
			}
			f.TempID = asString(fm["id"])
			f.ParentTempID = asString(fm["parentId"])
			f.Name = asString(fm["name"])
			f.Description = asString(fm["description"])
			f.Order = asInt(fm["order"], 0)
			coll.Folders = append(coll.Folders, f)
		}
		for _, rr := range asSlice(m["requests"]) {
			var req model.Request
			if json.Unmarshal(mustJSON(rr), &req) != nil {
				continue
			}
			if req.FolderID != nil {
				// Keep folder id; app maps real ids.
			}
			coll.Requests = append(coll.Requests, req)
		}
		parsed.Collections = append(parsed.Collections, coll)
	}
	for _, re := range asSlice(doc["environments"]) {
		var env model.Environment
		if json.Unmarshal(mustJSON(re), &env) == nil {
			parsed.Environments = append(parsed.Environments, model.ImportedEnvironment{Name: env.Name, Values: env.Values})
		}
	}
	for _, rg := range asSlice(doc["globals"]) {
		var v model.Variable
		if json.Unmarshal(mustJSON(rg), &v) == nil {
			parsed.Globals = append(parsed.Globals, v)
		}
	}
	return parsed, nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func orString(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "item"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}
