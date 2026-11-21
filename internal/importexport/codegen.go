package importexport

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"Postwoman/internal/model"
)

// ExportAsCode renders a request as runnable code for the given target.
// Supported targets: curl, http, fetch, node, python, go, php.
//
// Variable placeholders ({{baseUrl}}) are preserved verbatim so generated
// snippets stay honest about the request configuration. The caller can resolve
// them separately when that is desirable.
func ExportAsCode(req *model.Request, target string) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is required")
	}
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "curl":
		return ExportAsCurl(req), nil
	case "http":
		return exportHTTP(req), nil
	case "fetch", "javascript", "js":
		return exportFetch(req), nil
	case "node", "nodejs":
		return exportNode(req), nil
	case "python", "requests":
		return exportPython(req), nil
	case "go", "golang":
		return exportGo(req), nil
	case "php":
		return exportPHP(req), nil
	default:
		return "", fmt.Errorf("unsupported code target %q", target)
	}
}

type bodyInfo struct {
	mode    string
	raw     string
	form    []model.KeyValue
	encoded []model.KeyValue
	binary  string
	file    string
}

func describeBody(req *model.Request) bodyInfo {
	info := bodyInfo{mode: req.Body.Mode}
	switch req.Body.Mode {
	case "json", "xml", "text", "javascript", "html", "raw":
		info.raw = req.Body.Raw
	case "formdata":
		info.form = req.Body.FormData
	case "urlencoded":
		info.encoded = req.Body.URLEncoded
	case "binary":
		info.binary = req.Body.Binary
		info.file = req.Body.FileName
	case "graphql":
		if req.Body.GraphQL != nil {
			vars := json.RawMessage(strings.TrimSpace(req.Body.GraphQL.Variables))
			if len(vars) == 0 || !json.Valid(vars) {
				vars = json.RawMessage("{}")
			}
			payload, _ := json.Marshal(map[string]any{"query": req.Body.GraphQL.Query, "variables": vars})
			info.raw = string(payload)
			info.mode = "json"
		}
	}
	return info
}

// codeHeaders returns enabled headers with authentication applied.
func codeHeaders(req *model.Request) []model.KeyValue {
	out := []model.KeyValue{}
	hasContentType := false
	for _, h := range req.Headers {
		if !h.Enabled || strings.TrimSpace(h.Key) == "" {
			continue
		}
		if strings.EqualFold(h.Key, "Content-Type") {
			if hasContentType {
				continue
			}
			hasContentType = true
		}
		out = append(out, h)
	}
	add := func(key, value string) {
		out = append(out, model.KeyValue{Key: key, Value: value, Enabled: true})
	}
	auth := req.Auth
	switch auth.Type {
	case "bearer":
		if auth.Bearer != nil {
			add("Authorization", "Bearer "+auth.Bearer.Token)
		}
	case "jwt":
		if auth.JWT != nil {
			prefix := auth.JWT.HeaderPrefix
			if prefix == "" {
				prefix = "Bearer"
			}
			add("Authorization", prefix+" "+auth.JWT.Token)
		}
	case "oauth2":
		if auth.OAuth2 != nil && auth.OAuth2.AccessToken != "" {
			prefix := auth.OAuth2.TokenType
			if prefix == "" {
				prefix = "Bearer"
			}
			add("Authorization", prefix+" "+auth.OAuth2.AccessToken)
		}
	case "basic":
		if auth.Basic != nil {
			raw := auth.Basic.Username + ":" + auth.Basic.Password
			add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(raw)))
		}
	case "apikey":
		if auth.APIKey != nil && auth.APIKey.AddTo != "query" {
			add(auth.APIKey.Key, auth.APIKey.Value)
		}
	}
	if !hasContentType {
		if ct := contentTypeForMode(req.Body.Mode); ct != "" {
			out = append(out, model.KeyValue{Key: "Content-Type", Value: ct, Enabled: true})
		}
	}
	return out
}

func contentTypeForMode(mode string) string {
	switch mode {
	case "json", "graphql":
		return "application/json"
	case "xml":
		return "application/xml"
	case "html":
		return "text/html"
	case "javascript":
		return "application/javascript"
	case "urlencoded":
		return "application/x-www-form-urlencoded"
	case "text", "raw":
		return "text/plain"
	case "binary":
		return "application/octet-stream"
	default:
		return "" // multipart sets its own boundary
	}
}

// finalURL merges the params table into the URL (deduplicating what is already
// present) and applies query-string API keys.
func finalURL(req *model.Request) string {
	raw := req.URL
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	existing := map[string]int{}
	for k, values := range query {
		for _, v := range values {
			existing[k+"\x00"+v]++
		}
	}
	for _, p := range req.QueryParams {
		if !p.Enabled || strings.TrimSpace(p.Key) == "" {
			continue
		}
		token := p.Key + "\x00" + p.Value
		if existing[token] > 0 {
			existing[token]--
			continue
		}
		query.Add(p.Key, p.Value)
	}
	if req.Auth.Type == "apikey" && req.Auth.APIKey != nil && req.Auth.APIKey.AddTo == "query" {
		query.Set(req.Auth.APIKey.Key, req.Auth.APIKey.Value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func sortedHeaders(headers []model.KeyValue) []model.KeyValue {
	out := append([]model.KeyValue{}, headers...)
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Key) < strings.ToLower(out[j].Key)
	})
	return out
}

func jsQuote(s string) string { return strconv.Quote(s) }

func pyQuote(s string) string { return strconv.Quote(s) }

func phpQuote(s string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "'", "\\'") + "'"
}
func exportHTTP(req *model.Request) string {
	method := strings.ToUpper(orString(req.Method, "GET"))
	rawURL := finalURL(req)
	parsed, err := url.Parse(rawURL)
	host, target := rawURL, "/"
	if err == nil {
		host = parsed.Host
		target = parsed.RequestURI()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s HTTP/1.1\r\n", method, target)
	fmt.Fprintf(&b, "Host: %s\r\n", host)
	for _, h := range codeHeaders(req) {
		fmt.Fprintf(&b, "%s: %s\r\n", h.Key, h.Value)
	}
	body := bodyString(req)
	if body != "" {
		fmt.Fprintf(&b, "Content-Length: %d\r\n", len(body))
	}
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

// bodyString renders the body for textual targets. Multipart and binary bodies
// need target-specific handling and are returned empty here.
func bodyString(req *model.Request) string {
	info := describeBody(req)
	switch req.Body.Mode {
	case "json", "xml", "text", "javascript", "html", "raw":
		return info.raw
	case "urlencoded":
		values := url.Values{}
		for _, kv := range info.encoded {
			if kv.Enabled && strings.TrimSpace(kv.Key) != "" {
				values.Add(kv.Key, kv.Value)
			}
		}
		return values.Encode()
	case "graphql":
		return info.raw
	default:
		return ""
	}
}

func headersLiteral(headers []model.KeyValue, indent string) string {
	if len(headers) == 0 {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{\n")
	for _, h := range sortedHeaders(headers) {
		fmt.Fprintf(&b, "%s  %s: %s,\n", indent, jsQuote(h.Key), jsQuote(h.Value))
	}
	fmt.Fprintf(&b, "%s}", indent)
	return b.String()
}

func fetchBodyLines(req *model.Request, indent string) (string, string) {
	// Returns prelude statements and the body expression.
	info := describeBody(req)
	switch req.Body.Mode {
	case "formdata":
		var pre strings.Builder
		pre.WriteString("const form = new FormData();\n")
		for _, kv := range info.form {
			if !kv.Enabled || strings.TrimSpace(kv.Key) == "" {
				continue
			}
			if kv.Type == "file" {
				pre.WriteString("// File fields need a File object (for example from <input type=\"file\">).\n")
				fmt.Fprintf(&pre, "form.append(%s, fileInput.files[0], %s);\n", jsQuote(kv.Key), jsQuote(orString(kv.FileName, "file")))
			} else {
				fmt.Fprintf(&pre, "form.append(%s, %s);\n", jsQuote(kv.Key), jsQuote(kv.Value))
			}
		}
		return pre.String() + indent, "form"
	case "binary":
		return fmt.Sprintf("// Binary payload %s: load the bytes into a Blob or ArrayBuffer.\n%s", jsQuote(orString(info.file, "payload")), indent), "binaryPayload"
	default:
		body := bodyString(req)
		if body == "" {
			return "", ""
		}
		return "", jsQuote(body)
	}
}

func exportFetch(req *model.Request) string {
	method := strings.ToUpper(orString(req.Method, "GET"))
	prelude, bodyExpr := fetchBodyLines(req, "  ")
	var b strings.Builder
	if prelude != "" {
		b.WriteString(strings.TrimRight(prelude, "  "))
	}
	b.WriteString("const response = await fetch(")
	b.WriteString(jsQuote(finalURL(req)))
	b.WriteString(", {\n")
	fmt.Fprintf(&b, "  method: %s,\n", jsQuote(method))
	fmt.Fprintf(&b, "  headers: %s,\n", headersLiteral(codeHeaders(req), "  "))
	if bodyExpr != "" {
		fmt.Fprintf(&b, "  body: %s,\n", bodyExpr)
	}
	b.WriteString("});\n\n")
	b.WriteString("const text = await response.text();\n")
	b.WriteString("console.log(response.status, text);\n")
	return b.String()
}

func nodeBodyLines(req *model.Request) (string, string) {
	info := describeBody(req)
	switch req.Body.Mode {
	case "formdata":
		var pre strings.Builder
		pre.WriteString("import { readFile } from \"node:fs/promises\";\n\n")
		pre.WriteString("const form = new FormData();\n")
		for _, kv := range info.form {
			if !kv.Enabled || strings.TrimSpace(kv.Key) == "" {
				continue
			}
			if kv.Type == "file" {
				path := orString(kv.Src, kv.Value)
				fmt.Fprintf(&pre, "form.append(%s, new Blob([await readFile(%s)]), %s);\n",
					jsQuote(kv.Key), jsQuote(path), jsQuote(orString(kv.FileName, pathBase(path))))
			} else {
				fmt.Fprintf(&pre, "form.append(%s, %s);\n", jsQuote(kv.Key), jsQuote(kv.Value))
			}
		}
		return pre.String(), "form"
	case "binary":
		return "import { readFile } from \"node:fs/promises\";\n", fmt.Sprintf("await readFile(%s)", jsQuote(orString(info.file, "payload.bin")))
	default:
		body := bodyString(req)
		if body == "" {
			return "", ""
		}
		return "", jsQuote(body)
	}
}

func exportNode(req *model.Request) string {
	method := strings.ToUpper(orString(req.Method, "GET"))
	prelude, bodyExpr := nodeBodyLines(req)
	var b strings.Builder
	b.WriteString("// Node.js 18+ (global fetch)\n")
	if prelude != "" {
		b.WriteString(prelude)
	}
	fmt.Fprintf(&b, "const response = await fetch(%s, {\n", jsQuote(finalURL(req)))
	fmt.Fprintf(&b, "  method: %s,\n", jsQuote(method))
	fmt.Fprintf(&b, "  headers: %s,\n", headersLiteral(codeHeaders(req), "  "))
	if bodyExpr != "" {
		fmt.Fprintf(&b, "  body: %s,\n", bodyExpr)
	}
	b.WriteString("});\n\n")
	b.WriteString("console.log(response.status, await response.text());\n")
	return b.String()
}

func pyDict(items []model.KeyValue) string {
	rows := []string{}
	for _, kv := range items {
		if !kv.Enabled || strings.TrimSpace(kv.Key) == "" {
			continue
		}
		rows = append(rows, fmt.Sprintf("    %s: %s,", pyQuote(kv.Key), pyQuote(kv.Value)))
	}
	if len(rows) == 0 {
		return "{}"
	}
	return "{\n" + strings.Join(rows, "\n") + "\n}"
}

func exportPython(req *model.Request) string {
	method := strings.ToUpper(orString(req.Method, "GET"))
	info := describeBody(req)
	var b strings.Builder
	b.WriteString("import requests\n\n")
	fmt.Fprintf(&b, "url = %s\n", pyQuote(finalURL(req)))
	b.WriteString("headers = {")
	headers := sortedHeaders(codeHeaders(req))
	if len(headers) > 0 {
		b.WriteString("\n")
		for _, h := range headers {
			fmt.Fprintf(&b, "    %s: %s,\n", pyQuote(h.Key), pyQuote(h.Value))
		}
		b.WriteString("}")
	} else {
		b.WriteString("}")
	}
	b.WriteString("\n\n")

	args := []string{"url", "headers=headers"}
	switch req.Body.Mode {
	case "formdata":
		data := []model.KeyValue{}
		files := []string{}
		for _, kv := range info.form {
			if !kv.Enabled || strings.TrimSpace(kv.Key) == "" {
				continue
			}
			if kv.Type == "file" {
				path := orString(kv.Src, kv.Value)
				files = append(files, fmt.Sprintf("    %s: open(%s, \"rb\"),", pyQuote(kv.Key), pyQuote(path)))
			} else {
				data = append(data, kv)
			}
		}
		if len(data) > 0 {
			fmt.Fprintf(&b, "data = %s\n", pyDict(data))
			args = append(args, "data=data")
		}
		if len(files) > 0 {
			b.WriteString("files = {\n")
			b.WriteString(strings.Join(files, "\n"))
			b.WriteString("\n}\n")
			args = append(args, "files=files")
		}
	case "urlencoded":
		fmt.Fprintf(&b, "data = %s\n", pyDict(info.encoded))
		args = append(args, "data=data")
	case "binary":
		fmt.Fprintf(&b, "payload = open(%s, \"rb\")\n", pyQuote(orString(info.file, "payload.bin")))
		args = append(args, "data=payload")
	default:
		if body := bodyString(req); body != "" {
			fmt.Fprintf(&b, "payload = %s\n", pyQuote(body))
			args = append(args, "data=payload")
		}
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "response = requests.request(%s, %s, %s)\n", pyQuote(method), pyQuote(finalURL(req)), strings.Join(args, ", "))
	b.WriteString("print(response.status_code)\n")
	b.WriteString("print(response.text)\n")
	return b.String()
}
func exportGo(req *model.Request) string {
	method := strings.ToUpper(orString(req.Method, "GET"))
	info := describeBody(req)
	var imports []string
	var setup strings.Builder

	switch req.Body.Mode {
	case "formdata":
		imports = append(imports, "bytes", "mime/multipart", "os")
		setup.WriteString("\tvar buf bytes.Buffer\n\twriter := multipart.NewWriter(&buf)\n")
		for _, kv := range info.form {
			if !kv.Enabled || strings.TrimSpace(kv.Key) == "" {
				continue
			}
			if kv.Type == "file" {
				path := orString(kv.Src, kv.Value)
				fmt.Fprintf(&setup, "\tfilePart, err := os.Open(%s)\n\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n", strconv.Quote(path))
				fmt.Fprintf(&setup, "\tpart, err := writer.CreateFormFile(%s, %s)\n\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n",
					strconv.Quote(kv.Key), strconv.Quote(orString(kv.FileName, pathBase(path))))
				setup.WriteString("\tif _, err := io.Copy(part, filePart); err != nil {\n\t\tlog.Fatal(err)\n\t}\n\tfilePart.Close()\n")
			} else {
				fmt.Fprintf(&setup, "\t_ = writer.WriteField(%s, %s)\n", strconv.Quote(kv.Key), strconv.Quote(kv.Value))
			}
		}
		setup.WriteString("\t_ = writer.Close()\n")
		imports = append(imports, "io")
		setup.WriteString("\tpayload := &buf\n")
	case "binary":
		imports = append(imports, "os")
		fmt.Fprintf(&setup, "\tpayload, err := os.Open(%s)\n\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n\tdefer payload.Close()\n", strconv.Quote(orString(info.file, "payload.bin")))
	default:
		if body := bodyString(req); body != "" {
			imports = append(imports, "strings")
			fmt.Fprintf(&setup, "\tpayload := strings.NewReader(%s)\n", strconv.Quote(body))
		} else {
			setup.WriteString("\tvar payload io.Reader\n")
			imports = append(imports, "io")
		}
	}

	imports = append(imports, "fmt", "io", "log", "net/http")
	seen := map[string]bool{}
	uniq := []string{}
	for _, imp := range imports {
		if !seen[imp] {
			seen[imp] = true
			uniq = append(uniq, imp)
		}
	}
	sort.Strings(uniq)

	var b strings.Builder
	b.WriteString("package main\n\nimport (\n")
	for _, imp := range uniq {
		fmt.Fprintf(&b, "\t%s\n", strconv.Quote(imp))
	}
	b.WriteString(")\n\nfunc main() {\n")
	b.WriteString(setup.String())
	fmt.Fprintf(&b, "\n\treq, err := http.NewRequest(%s, %s, payload)\n", strconv.Quote(method), strconv.Quote(finalURL(req)))
	b.WriteString("\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n\n")
	for _, h := range sortedHeaders(codeHeaders(req)) {
		fmt.Fprintf(&b, "\treq.Header.Set(%s, %s)\n", strconv.Quote(h.Key), strconv.Quote(h.Value))
	}
	if req.Body.Mode == "formdata" {
		b.WriteString("\treq.Header.Set(\"Content-Type\", writer.FormDataContentType())\n")
	}
	b.WriteString("\n\tresp, err := http.DefaultClient.Do(req)\n")
	b.WriteString("\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n\tdefer resp.Body.Close()\n\n")
	b.WriteString("\tbody, _ := io.ReadAll(resp.Body)\n")
	b.WriteString("\tfmt.Println(resp.Status)\n")
	b.WriteString("\tfmt.Println(string(body))\n}\n")
	return b.String()
}

func phpArray(items []model.KeyValue, fileKey bool) string {
	rows := []string{}
	for _, kv := range items {
		if !kv.Enabled || strings.TrimSpace(kv.Key) == "" {
			continue
		}
		if fileKey && kv.Type == "file" {
			path := orString(kv.Src, kv.Value)
			rows = append(rows, fmt.Sprintf("        %s => new CURLFile(%s, %s, %s),",
				phpQuote(kv.Key), phpQuote(path), phpQuote(orString(kv.ContentType, "application/octet-stream")), phpQuote(orString(kv.FileName, pathBase(path)))))
		} else {
			rows = append(rows, fmt.Sprintf("        %s => %s,", phpQuote(kv.Key), phpQuote(kv.Value)))
		}
	}
	if len(rows) == 0 {
		return "[]"
	}
	return "[\n" + strings.Join(rows, "\n") + "\n    ]"
}

func exportPHP(req *model.Request) string {
	method := strings.ToUpper(orString(req.Method, "GET"))
	info := describeBody(req)
	var b strings.Builder
	b.WriteString("<?php\n\n$ch = curl_init();\n\n$headers = [\n")
	for _, h := range sortedHeaders(codeHeaders(req)) {
		fmt.Fprintf(&b, "    %s,\n", phpQuote(h.Key+": "+h.Value))
	}
	b.WriteString("];\n\n")
	b.WriteString("curl_setopt_array($ch, [\n")
	fmt.Fprintf(&b, "    CURLOPT_URL => %s,\n", phpQuote(finalURL(req)))
	b.WriteString("    CURLOPT_RETURNTRANSFER => true,\n")
	fmt.Fprintf(&b, "    CURLOPT_CUSTOMREQUEST => %s,\n", phpQuote(method))
	b.WriteString("    CURLOPT_HTTPHEADER => $headers,\n")
	switch req.Body.Mode {
	case "formdata":
		fmt.Fprintf(&b, "    CURLOPT_POSTFIELDS => %s,\n", phpArray(info.form, true))
	case "binary":
		fmt.Fprintf(&b, "    CURLOPT_POSTFIELDS => file_get_contents(%s),\n", phpQuote(orString(info.file, "payload.bin")))
	case "graphql":
		if body := bodyString(req); body != "" {
			fmt.Fprintf(&b, "    CURLOPT_POSTFIELDS => %s,\n", phpQuote(body))
		}
	case "urlencoded":
		fmt.Fprintf(&b, "    CURLOPT_POSTFIELDS => http_build_query(%s),\n", phpArray(info.encoded, false))
	default:
		if body := bodyString(req); body != "" {
			fmt.Fprintf(&b, "    CURLOPT_POSTFIELDS => %s,\n", phpQuote(body))
		}
	}
	b.WriteString("]);\n\n")
	b.WriteString("$response = curl_exec($ch);\n")
	b.WriteString("$status = curl_getinfo($ch, CURLINFO_HTTP_CODE);\n")
	b.WriteString("curl_close($ch);\n\n")
	b.WriteString("echo $status . \"\\n\" . $response;\n")
	return b.String()
}
