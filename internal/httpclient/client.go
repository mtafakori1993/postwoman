package httpclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"Postwoman/internal/model"
	"Postwoman/internal/variables"
)

const maxResponseBytes = 200 * 1024 * 1024 // 200 MiB hard ceiling

// Client performs HTTP requests. One client is shared by the app so cookies
// persist between requests, matching Postman's cookie-jar behaviour.
type Client struct {
	mu      sync.RWMutex
	jar     *cookiejar.Jar
	origins map[string]struct{}
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &Client{jar: jar, origins: map[string]struct{}{}}, nil
}

// jarHandle returns the current cookie jar. Reads are guarded because the jar
// can be replaced when the user clears cookies.
func (c *Client) jarHandle() *cookiejar.Jar {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jar
}

// Do executes a request and always returns a populated Response. Transport
// errors are reported through Response.Error so the UI can render them like
// any other response instead of throwing.
func (c *Client) Do(ctx context.Context, reqModel *model.Request, resolver *variables.Resolver) (*model.Response, error) {
	start := time.Now()
	response := &model.Response{
		Timestamp: start.UnixMilli(),
		Headers:   []model.KeyValue{},
		Cookies:   []model.Cookie{},
		Tests:     []model.TestResult{},
		Console:   []model.ConsoleEntry{},
	}
	if resolver == nil {
		resolver = variables.New()
	}

	settings := reqModel.Settings
	if settings.Timeout <= 0 {
		settings.Timeout = 30000
	}
	if settings.MaxRedirects <= 0 {
		settings.MaxRedirects = 10
	}
	if settings.Encoding == "" {
		settings.Encoding = "auto"
	}

	method := strings.ToUpper(strings.TrimSpace(reqModel.Method))
	if method == "" {
		method = "GET"
	}

	// Resolve explicit path variables first (they win over generic variable
	// interpolation), then run the normal {{variable}} resolver over the URL.
	rawURL := applyPathVariables(reqModel.URL, reqModel.PathVariables, resolver)
	rawURL = resolver.Resolve(rawURL)
	if strings.TrimSpace(rawURL) == "" {
		response.Error = "request URL is empty"
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		response.Error = "invalid URL: " + err.Error()
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		response.Error = fmt.Sprintf("unsupported URL protocol %q (only http and https are supported)", parsedURL.Scheme)
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}

	// The URL and the params table are kept in sync by the UI, so blindly
	// adding every enabled row would send duplicate parameters. Only add rows
	// that are not already present in the URL (counting duplicates).
	query := parsedURL.Query()
	existing := map[string]int{}
	for k, values := range query {
		for _, v := range values {
			existing[k+"\x00"+v]++
		}
	}
	for _, p := range reqModel.QueryParams {
		if !p.Enabled || strings.TrimSpace(p.Key) == "" {
			continue
		}
		key := resolver.Resolve(p.Key)
		value := resolver.Resolve(p.Value)
		token := key + "\x00" + value
		if existing[token] > 0 {
			existing[token]--
			continue
		}
		query.Add(key, value)
	}
	parsedURL.RawQuery = query.Encode()
	c.recordOrigin(parsedURL)

	bodyBytes, contentType, err := buildBody(reqModel.Body, resolver)
	if err != nil {
		response.Error = err.Error()
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}

	transport, err := c.buildTransport(&settings)
	if err != nil {
		response.Error = err.Error()
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}

	httpClient := &http.Client{
		Jar:       c.jarHandle(),
		Transport: transport,
		Timeout:   time.Duration(settings.Timeout) * time.Millisecond,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if !settings.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= settings.MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	if reqModel.Auth.Type == "oauth2" && reqModel.Auth.OAuth2 != nil {
		if err := c.prepareOAuth2(ctx, httpClient, reqModel.Auth.OAuth2, resolver); err != nil {
			response.Error = err.Error()
			response.ResponseTime = time.Since(start).Milliseconds()
			return response, nil
		}
	}

	newRequest := func() (*http.Request, error) {
		var body io.Reader
		if len(bodyBytes) > 0 {
			body = bytes.NewReader(bodyBytes)
		}
		httpReq, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), body)
		if err != nil {
			return nil, err
		}
		for _, h := range reqModel.Headers {
			if !h.Enabled || strings.TrimSpace(h.Key) == "" {
				continue
			}
			value := resolver.Resolve(h.Value)
			if strings.EqualFold(h.Key, "Host") {
				httpReq.Host = value
				continue
			}
			// Content-Type must be single-valued. Using Set keeps the last
			// enabled row and prevents conflicting duplicate headers.
			if strings.EqualFold(h.Key, "Content-Type") {
				httpReq.Header.Set(h.Key, value)
				continue
			}
			httpReq.Header.Add(h.Key, value)
		}
		if contentType != "" {
			// Multipart boundaries are generated while building the body, so a
			// user-supplied header can never be complete. Always use the value
			// that includes the actual boundary.
			if reqModel.Body.Mode == "formdata" || httpReq.Header.Get("Content-Type") == "" {
				httpReq.Header.Set("Content-Type", contentType)
			}
		}
		if httpReq.Header.Get("User-Agent") == "" {
			httpReq.Header.Set("User-Agent", "Postwoman/1.0")
		}
		if httpReq.Header.Get("Accept") == "" {
			httpReq.Header.Set("Accept", "*/*")
		}
		return httpReq, nil
	}

	httpReq, err := newRequest()
	if err != nil {
		response.Error = "failed to build request: " + err.Error()
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}

	if err := c.applyAuth(ctx, httpClient, httpReq, reqModel.Auth, resolver); err != nil {
		response.Error = err.Error()
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		response.Error = friendlyNetworkError(err)
		response.ResponseTime = time.Since(start).Milliseconds()
		return response, nil
	}

	// Digest auth requires a challenge round-trip. Retry exactly once.
	if resp.StatusCode == http.StatusUnauthorized && reqModel.Auth.Type == "digest" && reqModel.Auth.Digest != nil {
		challenge := resp.Header.Get("WWW-Authenticate")
		if strings.HasPrefix(strings.ToLower(challenge), "digest") {
			_ = resp.Body.Close()
			retry, derr := newRequest()
			if derr == nil {
				if herr := applyDigest(retry, reqModel.Auth.Digest, challenge, resolver); herr == nil {
					resp, err = httpClient.Do(retry)
					if err != nil {
						response.Error = friendlyNetworkError(err)
						response.ResponseTime = time.Since(start).Milliseconds()
						return response, nil
					}
				}
			}
		}
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, readErr := io.ReadAll(limited)
	if readErr != nil {
		response.Error = "failed to read response body: " + readErr.Error()
	}
	truncated := int64(len(body)) > maxResponseBytes
	if truncated {
		body = body[:maxResponseBytes]
	}

	response.StatusCode = resp.StatusCode
	response.StatusText = strings.TrimSpace(strings.TrimPrefix(resp.Status, fmt.Sprintf("%d", resp.StatusCode)))
	if response.StatusText == "" {
		response.StatusText = http.StatusText(resp.StatusCode)
	}
	response.Headers = headerList(resp.Header)
	response.ContentType = resp.Header.Get("Content-Type")
	response.BodySize = int64(len(body))
	response.HeaderSize = headerSize(resp.Header)
	response.ResponseTime = time.Since(start).Milliseconds()
	response.Redirected = resp.Request != nil && resp.Request.URL.String() != parsedURL.String()
	response.URL = resp.Request.URL.String()
	response.IsBinary = !isTextual(response.ContentType, body)
	if response.IsBinary {
		response.BodyBase64 = base64.StdEncoding.EncodeToString(body)
	} else {
		response.Body = string(body)
	}
	if truncated {
		response.Error = fmt.Sprintf("response truncated at %d bytes", maxResponseBytes)
	}

	if cookies := c.jarHandle().Cookies(parsedURL); len(cookies) > 0 {
		for _, ck := range cookies {
			response.Cookies = append(response.Cookies, toCookie(ck))
		}
	}
	if len(response.Cookies) == 0 {
		for _, ck := range resp.Cookies() {
			response.Cookies = append(response.Cookies, toCookie(ck))
		}
	}
	return response, nil
}

func friendlyNetworkError(err error) string {
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "context canceled"), strings.Contains(lower, "operation was canceled"):
		return "request cancelled"
	case strings.Contains(lower, "context deadline exceeded"), strings.Contains(lower, "timeout"):
		return "request timed out"
	case strings.Contains(lower, "connection refused"):
		return "connection refused by server"
	case strings.Contains(lower, "no such host"):
		return "host could not be resolved (DNS lookup failed)"
	case strings.Contains(lower, "certificate"), strings.Contains(lower, "x509"):
		return "TLS certificate verification failed: " + msg
	case strings.Contains(lower, "unsupported protocol scheme"):
		return "unsupported URL protocol"
	}
	return msg
}

func applyPathVariables(rawURL string, pathVars []model.KeyValue, resolver *variables.Resolver) string {
	for _, pv := range pathVars {
		if !pv.Enabled || strings.TrimSpace(pv.Key) == "" {
			continue
		}
		value := resolver.Resolve(pv.Value)
		if value == "" {
			continue
		}
		// Support both Postman-style :id segments and {{id}} placeholders in
		// the path. The explicit path-variable row wins over a same-named
		// environment variable.
		rawURL = replacePathToken(rawURL, ":"+pv.Key, value)
		rawURL = strings.ReplaceAll(rawURL, "{{"+pv.Key+"}}", value)
	}
	return rawURL
}

// replacePathToken replaces occurrences of a :key segment where the token is
// preceded by "/" (or starts the string) and followed by a path boundary.
// This avoids corrupting "https://" or double-prefix keys such as :id/:idx.
func replacePathToken(raw, token, value string) string {
	if !strings.Contains(raw, token) {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); {
		if strings.HasPrefix(raw[i:], token) && (i == 0 || raw[i-1] == '/') {
			end := i + len(token)
			if end == len(raw) || !isPathKeyChar(raw[end]) {
				b.WriteString(value)
				i = end
				continue
			}
		}
		b.WriteByte(raw[i])
		i++
	}
	return b.String()
}

func isPathKeyChar(c byte) bool {
	return c == '_' || c == '-' || c == '.' ||
		(c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func buildBody(body model.Body, resolver *variables.Resolver) ([]byte, string, error) {
	if body.Mode == "" || body.Mode == "none" {
		return nil, "", nil
	}
	switch body.Mode {
	case "json":
		return []byte(resolver.Resolve(body.Raw)), "application/json", nil
	case "raw":
		return []byte(resolver.Resolve(body.Raw)), "text/plain", nil
	case "xml":
		return []byte(resolver.Resolve(body.Raw)), "application/xml", nil
	case "html":
		return []byte(resolver.Resolve(body.Raw)), "text/html", nil
	case "javascript":
		return []byte(resolver.Resolve(body.Raw)), "application/javascript", nil
	case "text":
		return []byte(resolver.Resolve(body.Raw)), "text/plain", nil
	case "urlencoded":
		values := url.Values{}
		for _, kv := range body.URLEncoded {
			if kv.Enabled && strings.TrimSpace(kv.Key) != "" {
				values.Add(resolver.Resolve(kv.Key), resolver.Resolve(kv.Value))
			}
		}
		return []byte(values.Encode()), "application/x-www-form-urlencoded", nil
	case "formdata":
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		for _, kv := range body.FormData {
			if !kv.Enabled || strings.TrimSpace(kv.Key) == "" {
				continue
			}
			key := resolver.Resolve(kv.Key)
			if kv.Type == "file" {
				path := resolver.Resolve(kv.Src)
				if path == "" {
					path = resolver.Resolve(kv.Value)
				}
				content, name, err := readUploadFile(path)
				if err != nil {
					_ = writer.Close()
					return nil, "", fmt.Errorf("cannot read file %q: %w", path, err)
				}
				contentType := kv.ContentType
				if contentType == "" {
					contentType = http.DetectContentType(content)
				}
				header := textproto.MIMEHeader{}
				header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(key), escapeQuotes(name)))
				header.Set("Content-Type", contentType)
				part, err := writer.CreatePart(header)
				if err != nil {
					_ = writer.Close()
					return nil, "", err
				}
				_, _ = part.Write(content)
				continue
			}
			_ = writer.WriteField(key, resolver.Resolve(kv.Value))
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), writer.FormDataContentType(), nil
	case "graphql":
		if body.GraphQL == nil {
			return nil, "application/json", nil
		}
		query := resolver.Resolve(body.GraphQL.Query)
		var gqlVars interface{}
		rawVars := strings.TrimSpace(body.GraphQL.Variables)
		if rawVars != "" {
			if err := json.Unmarshal([]byte(rawVars), &gqlVars); err != nil {
				return nil, "", fmt.Errorf("GraphQL variables are not valid JSON: %w", err)
			}
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"query":     query,
			"variables": gqlVars,
		})
		return payload, "application/json", nil
	case "binary":
		data := body.Binary
		if idx := strings.Index(data, ","); idx >= 0 && strings.Contains(data[:idx], "base64") {
			data = data[idx+1:]
		}
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, "", fmt.Errorf("binary body is not valid base64: %w", err)
		}
		return decoded, "application/octet-stream", nil
	default:
		return []byte(resolver.Resolve(body.Raw)), "", nil
	}
}

func escapeQuotes(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(value)
}

func readUploadFile(path string) ([]byte, string, error) {
	if path == "" {
		return nil, "", fmt.Errorf("no file selected")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	if info.IsDir() {
		return nil, "", fmt.Errorf("path is a directory")
	}
	if info.Size() > 100*1024*1024 {
		return nil, "", fmt.Errorf("file exceeds 100 MiB limit")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return data, filepath.Base(path), nil
}

func (c *Client) buildTransport(settings *model.Settings) (*http.Transport, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if settings.Proxy != nil && settings.Proxy.Host != "" {
		scheme := settings.Proxy.Protocol
		if scheme == "" {
			scheme = "http"
		}
		proxyURL := &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", settings.Proxy.Host, settings.Proxy.Port),
		}
		if settings.Proxy.Username != "" {
			proxyURL.User = url.UserPassword(settings.Proxy.Username, settings.Proxy.Password)
		}
		bypass := settings.Proxy.Bypass
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			for _, pattern := range bypass {
				if hostMatches(req.URL.Hostname(), pattern) {
					return nil, nil
				}
			}
			return proxyURL, nil
		}
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	reject := true
	if settings.TLS != nil {
		reject = settings.TLS.RejectUnauthorized
	}
	tlsConfig.InsecureSkipVerify = !reject

	if settings.TLS != nil && settings.TLS.CA != "" {
		pem, err := os.ReadFile(settings.TLS.CA)
		if err != nil {
			return nil, fmt.Errorf("cannot read CA certificate: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("CA certificate is not valid PEM")
		}
		tlsConfig.RootCAs = pool
	}
	if settings.TLS != nil && settings.TLS.Cert != "" && settings.TLS.Key != "" {
		cert, err := tls.LoadX509KeyPair(settings.TLS.Cert, settings.TLS.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}

func hostMatches(host, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	pattern = strings.ReplaceAll(pattern, ".", `\.`)
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	matched, _ := regexp.MatchString("^"+pattern+"$", host)
	return matched
}

func headerList(h http.Header) []model.KeyValue {
	out := make([]model.KeyValue, 0, len(h))
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range h[k] {
			out = append(out, model.KeyValue{Key: k, Value: v, Enabled: true})
		}
	}
	return out
}

func headerSize(h http.Header) int64 {
	var size int64
	for k, values := range h {
		for _, v := range values {
			size += int64(len(k) + len(v) + 4)
		}
	}
	return size
}

func toCookie(c *http.Cookie) model.Cookie {
	sameSite := "lax"
	switch c.SameSite {
	case http.SameSiteStrictMode:
		sameSite = "strict"
	case http.SameSiteNoneMode:
		sameSite = "none"
	}
	var expires int64
	if !c.Expires.IsZero() {
		expires = c.Expires.UnixMilli()
	}
	return model.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Expires:  expires,
		MaxAge:   c.MaxAge,
		Secure:   c.Secure,
		HTTPOnly: c.HttpOnly,
		SameSite: sameSite,
		HostOnly: c.Domain == "",
		Session:  c.MaxAge == 0 && c.Expires.IsZero(),
	}
}

func isTextual(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	for _, prefix := range []string{
		"text/", "application/json", "application/xml", "application/javascript",
		"application/x-www-form-urlencoded", "application/graphql",
		"application/problem+json", "application/xhtml", "application/ld+json",
	} {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	if ct == "" {
		if bytes.IndexByte(body, 0) >= 0 {
			return false
		}
		return json.Valid(body) || utf8.Valid(body)
	}
	return false
}

func (c *Client) applyAuth(ctx context.Context, httpClient *http.Client, req *http.Request, auth model.Auth, resolver *variables.Resolver) error {
	switch auth.Type {
	case "", "noauth":
		return nil
	case "apikey":
		if auth.APIKey == nil {
			return nil
		}
		key := resolver.Resolve(auth.APIKey.Key)
		value := resolver.Resolve(auth.APIKey.Value)
		if auth.APIKey.AddTo == "query" {
			q := req.URL.Query()
			q.Set(key, value)
			req.URL.RawQuery = q.Encode()
		} else {
			req.Header.Set(key, value)
		}
	case "bearer":
		if auth.Bearer != nil {
			req.Header.Set("Authorization", "Bearer "+resolver.Resolve(auth.Bearer.Token))
		}
	case "jwt":
		if auth.JWT != nil {
			prefix := auth.JWT.HeaderPrefix
			if prefix == "" {
				prefix = "Bearer"
			}
			req.Header.Set("Authorization", prefix+" "+resolver.Resolve(auth.JWT.Token))
		}
	case "basic":
		if auth.Basic != nil {
			req.SetBasicAuth(resolver.Resolve(auth.Basic.Username), resolver.Resolve(auth.Basic.Password))
		}
	case "digest":
		return nil // challenge handled by the caller
	case "oauth1":
		if auth.OAuth1 == nil {
			return nil
		}
		return signOAuth1(req, auth.OAuth1, resolver)
	case "oauth2":
		if auth.OAuth2 == nil {
			return nil
		}
		if auth.OAuth2.AccessToken != "" {
			prefix := auth.OAuth2.TokenType
			if prefix == "" {
				prefix = "Bearer"
			}
			req.Header.Set("Authorization", prefix+" "+resolver.Resolve(auth.OAuth2.AccessToken))
		}
	case "awsv4":
		if auth.AWSV4 == nil {
			return nil
		}
		return signAWSV4(req, auth.AWSV4, resolver)
	case "hawk":
		return fmt.Errorf("Hawk authentication is not supported yet")
	case "ntlm":
		return fmt.Errorf("NTLM authentication is not supported yet")
	}
	return nil
}

func (c *Client) prepareOAuth2(ctx context.Context, httpClient *http.Client, oauth *model.OAuth2Auth, resolver *variables.Resolver) error {
	resolved := resolveOAuth2(*oauth, resolver)
	expired := resolved.ExpiresAt > 0 && time.Now().UnixMilli() >= resolved.ExpiresAt
	if resolved.AccessToken != "" && !expired {
		return nil
	}
	if resolved.TokenURL == "" {
		if resolved.AccessToken == "" {
			return fmt.Errorf("OAuth2 access token is missing; use Authorize in the Auth tab or configure a token URL")
		}
		return nil
	}
	// Prefer a refresh token whenever the provider issued one; otherwise use
	// the configured grant type.
	grant := resolved.GrantType
	if resolved.RefreshToken != "" && grant != "client_credentials" && grant != "password" {
		grant = "refresh_token"
	}
	switch grant {
	case "client_credentials", "password", "refresh_token":
		resolved.GrantType = grant
		updated, err := c.FetchOAuth2Token(ctx, resolved)
		if err != nil {
			return err
		}
		oauth.AccessToken = updated.AccessToken
		if updated.RefreshToken != "" {
			oauth.RefreshToken = updated.RefreshToken
		}
		oauth.TokenType = updated.TokenType
		oauth.ExpiresAt = updated.ExpiresAt
		return nil
	default:
		if resolved.AccessToken == "" {
			return fmt.Errorf("OAuth2 authorization code flow needs an access token; use Authorize in the Auth tab")
		}
		return nil
	}
}

func applyDigest(req *http.Request, digest *model.DigestAuth, challenge string, resolver *variables.Resolver) error {
	params := parseAuthParams(challenge)
	realm := params["realm"]
	nonce := params["nonce"]
	opaque := params["opaque"]
	algorithm := params["algorithm"]
	qopRaw := params["qop"]
	qop := "auth"
	if qopRaw != "" && !strings.Contains(qopRaw, "auth") {
		qop = ""
	}
	uri := req.URL.RequestURI()
	nc := "00000001"
	cnonce := uuidLike()

	hash := func(s string) string {
		if strings.EqualFold(algorithm, "SHA-256") {
			sum := sha256.Sum256([]byte(s))
			return hex.EncodeToString(sum[:])
		}
		sum := md5.Sum([]byte(s))
		return hex.EncodeToString(sum[:])
	}

	username := resolver.Resolve(digest.Username)
	password := resolver.Resolve(digest.Password)
	ha1 := hash(username + ":" + realm + ":" + password)
	ha2 := hash(req.Method + ":" + uri)
	var response string
	if qop != "" {
		response = hash(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)
	} else {
		response = hash(ha1 + ":" + nonce + ":" + ha2)
	}

	parts := []string{
		fmt.Sprintf(`username="%s"`, username),
		fmt.Sprintf(`realm="%s"`, realm),
		fmt.Sprintf(`nonce="%s"`, nonce),
		fmt.Sprintf(`uri="%s"`, uri),
		fmt.Sprintf(`response="%s"`, response),
	}
	if algorithm != "" {
		parts = append(parts, fmt.Sprintf(`algorithm=%s`, algorithm))
	}
	if opaque != "" {
		parts = append(parts, fmt.Sprintf(`opaque="%s"`, opaque))
	}
	if qop != "" {
		parts = append(parts, fmt.Sprintf(`qop=%s`, qop), fmt.Sprintf(`nc=%s`, nc), fmt.Sprintf(`cnonce="%s"`, cnonce))
	}
	req.Header.Set("Authorization", "Digest "+strings.Join(parts, ", "))
	return nil
}

func parseAuthParams(header string) map[string]string {
	out := map[string]string{}
	idx := strings.Index(header, " ")
	if idx < 0 {
		return out
	}
	rest := header[idx+1:]
	re := regexp.MustCompile(`([a-zA-Z0-9_-]+)\s*=\s*("([^"]*)"|[^,]+)`)
	for _, m := range re.FindAllStringSubmatch(rest, -1) {
		value := m[3]
		if value == "" {
			value = strings.TrimSpace(m[2])
		}
		out[strings.ToLower(m[1])] = value
	}
	return out
}

func signOAuth1(req *http.Request, oauth *model.OAuth1Auth, resolver *variables.Resolver) error {
	consumerKey := resolver.Resolve(oauth.ConsumerKey)
	consumerSecret := resolver.Resolve(oauth.ConsumerSecret)
	token := resolver.Resolve(oauth.Token)
	tokenSecret := resolver.Resolve(oauth.TokenSecret)
	method := oauth.SignatureMethod
	if method == "" {
		method = "HMAC-SHA1"
	}

	oauthParams := map[string]string{
		"oauth_consumer_key":     consumerKey,
		"oauth_nonce":            uuidLike(),
		"oauth_signature_method": method,
		"oauth_timestamp":        fmt.Sprintf("%d", time.Now().Unix()),
		"oauth_version":          "1.0",
	}
	if token != "" {
		oauthParams["oauth_token"] = token
	}

	all := map[string]string{}
	for k, values := range req.URL.Query() {
		all[k] = strings.Join(values, " ")
	}
	for k, v := range oauthParams {
		all[k] = v
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(all[k]))
	}
	baseURL := req.URL.Scheme + "://" + req.URL.Host + req.URL.Path
	baseString := req.Method + "&" + percentEncode(baseURL) + "&" + percentEncode(strings.Join(pairs, "&"))
	signingKey := percentEncode(consumerSecret) + "&" + percentEncode(tokenSecret)

	var signature string
	switch method {
	case "PLAINTEXT":
		signature = signingKey
	case "HMAC-SHA256":
		mac := hmac.New(sha256.New, []byte(signingKey))
		mac.Write([]byte(baseString))
		signature = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	default:
		mac := hmac.New(sha1.New, []byte(signingKey))
		mac.Write([]byte(baseString))
		signature = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	oauthParams["oauth_signature"] = signature

	hkeys := make([]string, 0, len(oauthParams))
	for k := range oauthParams {
		hkeys = append(hkeys, k)
	}
	sort.Strings(hkeys)
	header := make([]string, 0, len(hkeys))
	for _, k := range hkeys {
		header = append(header, percentEncode(k)+`="`+percentEncode(oauthParams[k])+`"`)
	}
	value := "OAuth "
	if oauth.Realm != "" {
		value += `realm="` + oauth.Realm + `", `
	}
	value += strings.Join(header, ", ")
	req.Header.Set("Authorization", value)
	return nil
}

func percentEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func signAWSV4(req *http.Request, auth *model.AWSV4Auth, resolver *variables.Resolver) error {
	accessKey := resolver.Resolve(auth.AccessKey)
	secretKey := resolver.Resolve(auth.SecretKey)
	region := resolver.Resolve(auth.Region)
	service := resolver.Resolve(auth.Service)
	if accessKey == "" || secretKey == "" || region == "" || service == "" {
		return fmt.Errorf("AWS Signature v4 requires access key, secret key, region and service")
	}
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	payloadHash := sha256Hex(bodyBytes)

	req.Header.Set("X-Amz-Date", amzDate)
	if auth.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", resolver.Resolve(auth.SessionToken))
	}
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQuery := req.URL.Query().Encode()

	headerNames := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	headerValues := map[string]string{
		"host":                 req.URL.Host,
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzDate,
	}
	if token := req.Header.Get("X-Amz-Security-Token"); token != "" {
		headerNames = append(headerNames, "x-amz-security-token")
		headerValues["x-amz-security-token"] = token
	}
	sort.Strings(headerNames)
	var canonicalHeaders strings.Builder
	for _, name := range headerNames {
		canonicalHeaders.WriteString(name + ":" + strings.TrimSpace(headerValues[name]) + "\n")
	}
	signedHeaders := strings.Join(headerNames, ";")
	canonicalRequest := strings.Join([]string{
		req.Method, canonicalURI, canonicalQuery, canonicalHeaders.String(), signedHeaders, payloadHash,
	}, "\n")
	scope := dateStamp + "/" + region + "/" + service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256", amzDate, scope, sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	kDate := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, scope, signedHeaders, signature))
	return nil
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func uuidLike() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
