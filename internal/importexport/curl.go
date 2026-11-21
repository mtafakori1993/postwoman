package importexport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"Postwoman/internal/model"
)

// parseCurl converts a cURL command into a single-request collection.
func parseCurl(command string) (*Parsed, error) {
	tokens, err := splitShell(command)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 || !strings.HasPrefix(strings.ToLower(tokens[0]), "curl") {
		return nil, fmt.Errorf("not a cURL command")
	}
	req := &model.Request{
		Name:     "cURL Request",
		Method:   "GET",
		Body:     model.Body{Mode: "none"},
		Auth:     model.Auth{Type: "noauth"},
		Settings: model.Settings{FollowRedirects: true, MaxRedirects: 10, Timeout: 30000, Encoding: "auto"},
	}
	var dataParts []string
	formParts := []model.KeyValue{}
	explicitMethod := ""
	hasData := false
	hasJSON := false

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		next := func() (string, bool) {
			if i+1 < len(tokens) {
				i++
				return tokens[i], true
			}
			return "", false
		}
		switch tok {
		case "-X", "--request":
			if v, ok := next(); ok {
				explicitMethod = strings.ToUpper(v)
			}
		case "-H", "--header":
			if v, ok := next(); ok {
				if k, val, found := strings.Cut(v, ":"); found {
					key := strings.TrimSpace(k)
					req.Headers = append(req.Headers, model.KeyValue{Key: key, Value: strings.TrimSpace(val), Enabled: true})
					if strings.EqualFold(key, "Content-Type") && strings.Contains(strings.ToLower(val), "json") {
						hasJSON = true
					}
				}
			}
		case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii":
			if v, ok := next(); ok {
				dataParts = append(dataParts, v)
				hasData = true
			}
		case "--data-urlencode":
			if v, ok := next(); ok {
				dataParts = append(dataParts, v)
				hasData = true
			}
		case "-F", "--form":
			if v, ok := next(); ok {
				if k, val, found := strings.Cut(v, "="); found {
					field := model.KeyValue{Key: k, Value: val, Enabled: true, Type: "text"}
					if strings.HasPrefix(val, "@") {
						field.Type = "file"
						field.Src = strings.TrimPrefix(val, "@")
						field.FileName = pathBase(field.Src)
						field.Value = ""
					}
					formParts = append(formParts, field)
					hasData = true
				}
			}
		case "-u", "--user":
			if v, ok := next(); ok {
				user, pass, _ := strings.Cut(v, ":")
				req.Auth = model.Auth{Type: "basic", Basic: &model.BasicAuth{Username: user, Password: pass}}
			}
		case "-A", "--user-agent":
			if v, ok := next(); ok {
				req.Headers = append(req.Headers, model.KeyValue{Key: "User-Agent", Value: v, Enabled: true})
			}
		case "-b", "--cookie":
			if v, ok := next(); ok {
				req.Headers = append(req.Headers, model.KeyValue{Key: "Cookie", Value: v, Enabled: true})
			}
		case "-e", "--referer":
			if v, ok := next(); ok {
				req.Headers = append(req.Headers, model.KeyValue{Key: "Referer", Value: v, Enabled: true})
			}
		case "--url":
			if v, ok := next(); ok && req.URL == "" {
				req.URL = v
			}
		case "-I", "--head":
			explicitMethod = "HEAD"
		case "-G", "--get":
			explicitMethod = "GET"
		case "--compressed", "-s", "--silent", "-k", "--insecure", "-L", "--location", "-i", "--include", "-v", "--verbose", "--globoff":
			// Flags with no request representation.
		case "-o", "--output", "-m", "--max-time", "--connect-timeout", "-x", "--proxy", "--cacert", "--cert", "--key":
			// Consume the argument so it is not mistaken for the URL.
			_, _ = next()
		default:
			if strings.HasPrefix(tok, "-") {
				continue
			}
			if req.URL == "" {
				req.URL = tok
			}
		}
	}
	if req.URL == "" {
		return nil, fmt.Errorf("cURL command does not contain a URL")
	}

	// Move the query string into the params table for visibility while keeping
	// the raw URL intact.
	if u, err := url.Parse(req.URL); err == nil {
		for k, values := range u.Query() {
			for _, v := range values {
				req.QueryParams = append(req.QueryParams, model.KeyValue{Key: k, Value: v, Enabled: true})
			}
		}
		// Common case: cURL URLs already include the query string. Removing it
		// from the URL avoids sending every parameter twice.
		if len(req.QueryParams) > 0 {
			u.RawQuery = ""
			req.URL = u.String()
		}
	}

	if explicitMethod != "" {
		req.Method = explicitMethod
	} else if hasData && req.Method == "GET" {
		req.Method = "POST"
	}

	if len(formParts) > 0 {
		req.Body = model.Body{Mode: "formdata", FormData: formParts}
	} else if hasData {
		data := strings.Join(dataParts, "&")
		contentType := ""
		for _, h := range req.Headers {
			if strings.EqualFold(h.Key, "Content-Type") {
				contentType = strings.ToLower(h.Value)
			}
		}
		switch {
		case strings.Contains(contentType, "json") || hasJSON || looksLikeJSON(data):
			req.Body = model.Body{Mode: "json", Raw: data}
		case strings.Contains(contentType, "x-www-form-urlencoded"):
			req.Body = model.Body{Mode: "urlencoded", URLEncoded: parseFormPairs(data)}
		default:
			req.Body = model.Body{Mode: "text", Raw: data}
		}
	}

	parsed := &Parsed{Format: "curl", Name: "Imported cURL", Raw: command}
	parsed.Collections = []*model.ImportedCollection{{
		Info: model.CollectionInfo{
			Name:   "Imported cURL",
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Auth:     model.Auth{Type: "noauth"},
		Requests: []model.Request{*req},
	}}
	return parsed, nil
}

func looksLikeJSON(s string) bool {
	t := strings.TrimSpace(s)
	return (strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}")) ||
		(strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]"))
}

func parseFormPairs(data string) []model.KeyValue {
	values, err := url.ParseQuery(data)
	if err != nil {
		return []model.KeyValue{}
	}
	out := []model.KeyValue{}
	for k, v := range values {
		for _, vv := range v {
			out = append(out, model.KeyValue{Key: k, Value: vv, Enabled: true})
		}
	}
	return out
}

func pathBase(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// splitShell tokenises a shell command, honouring single quotes, double quotes,
// backslash escapes and line continuations.
func splitShell(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			if quote == '"' && r == '\\' {
				escaped = true
				continue
			}
			cur.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '\\':
			escaped = true
		case ' ', '\t', '\r', '\n':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in cURL command")
	}
	flush()
	return out, nil
}

// ExportAsCurl renders a request as a cURL command.
func ExportAsCurl(req *model.Request) string {
	parts := []string{"curl", "-X", strings.ToUpper(orString(req.Method, "GET"))}
	for _, h := range codeHeaders(req) {
		if strings.EqualFold(h.Key, "Content-Type") && req.Body.Mode == "formdata" {
			// curl generates its own multipart boundary.
			continue
		}
		parts = append(parts, "-H", shellQuote(h.Key+": "+h.Value))
	}
	switch req.Body.Mode {
	case "json", "xml", "text", "javascript", "html", "raw":
		if req.Body.Raw != "" {
			parts = append(parts, "--data-raw", shellQuote(req.Body.Raw))
		}
	case "urlencoded":
		values := url.Values{}
		for _, kv := range req.Body.URLEncoded {
			if kv.Enabled && kv.Key != "" {
				values.Add(kv.Key, kv.Value)
			}
		}
		if enc := values.Encode(); enc != "" {
			parts = append(parts, "--data-raw", shellQuote(enc))
		}
	case "graphql":
		if req.Body.GraphQL != nil {
			variables := json.RawMessage(orString(req.Body.GraphQL.Variables, "{}"))
			if !json.Valid(variables) {
				variables = json.RawMessage("{}")
			}
			payload, _ := json.Marshal(map[string]any{
				"query": req.Body.GraphQL.Query, "variables": variables,
			})
			parts = append(parts, "--data-raw", shellQuote(string(payload)))
		}
	case "formdata":
		for _, kv := range req.Body.FormData {
			if !kv.Enabled || kv.Key == "" {
				continue
			}
			if kv.Type == "file" {
				path := orString(kv.Src, kv.Value)
				if kv.ContentType != "" {
					parts = append(parts, "-F", shellQuote(kv.Key+"=@"+path+";type="+kv.ContentType))
				} else {
					parts = append(parts, "-F", shellQuote(kv.Key+"=@"+path))
				}
			} else {
				parts = append(parts, "-F", shellQuote(kv.Key+"="+kv.Value))
			}
		}
	case "binary":
		if req.Body.FileName != "" {
			parts = append(parts, "--data-binary", shellQuote("@"+req.Body.FileName))
		}
	}
	parts = append(parts, shellQuote(finalURL(req)))
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ParseCurlAsRequest converts a cURL command into a single request draft. It
// is used by the "Import cURL" dialog to open the command in an editable tab.
func ParseCurlAsRequest(command string) (*model.Request, error) {
	parsed, err := parseCurl(command)
	if err != nil {
		return nil, err
	}
	if len(parsed.Collections) == 0 || len(parsed.Collections[0].Requests) == 0 {
		return nil, fmt.Errorf("cURL command did not produce a request")
	}
	req := parsed.Collections[0].Requests[0]
	req.ID = ""
	req.CollectionID = ""
	req.FolderID = nil
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "Imported cURL"
	}
	return &req, nil
}
