package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"Postwoman/internal/model"
	"Postwoman/internal/netutil"
)

// Message is one chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the provider-agnostic chat request.
type ChatRequest struct {
	System      string
	Messages    []Message
	Model       string
	Temperature *float64
	MaxTokens   int
	JSONMode    bool
}

// ChatResponse is the provider-agnostic chat response.
type ChatResponse struct {
	Content string         `json:"content"`
	Model   string         `json:"model,omitempty"`
	Usage   map[string]int `json:"usage,omitempty"`
}

// Client sends chat requests to one configured provider. It is safe for
// concurrent use and never logs credentials.
type Client struct {
	cfg  model.AIProviderConfig
	http *http.Client
}

// NewClient validates and normalises a provider configuration and builds a
// client that respects the process proxy environment (HTTP_PROXY, HTTPS_PROXY,
// NO_PROXY) but no explicit application proxy.
func NewClient(cfg model.AIProviderConfig) (*Client, error) {
	return NewClientWithProxy(cfg, nil)
}

// NewClientWithProxy builds a client that routes AI provider traffic through
// the supplied proxy. A nil (or host-less) proxy keeps the environment default,
// so callers can pass model.AppSettings.AIProxy() directly.
func NewClientWithProxy(cfg model.AIProviderConfig, proxy *model.Proxy) (*Client, error) {
	normalized, err := Normalize(cfg)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(normalized.TimeoutSeconds) * time.Second
	httpClient := &http.Client{Timeout: timeout}
	if proxy != nil {
		proxyFunc, err := netutil.ProxyFunc(proxy)
		if err != nil {
			return nil, err
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = proxyFunc
		httpClient.Transport = transport
	}
	return &Client{
		cfg:  normalized,
		http: httpClient,
	}, nil
}

// Config returns the normalised provider configuration. The API key is included
// because the caller already owns it; it must never be logged or forwarded.
func (c *Client) Config() model.AIProviderConfig { return c.cfg }

var (
	bearerPattern = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._\-]{8,}`)
	skPattern     = regexp.MustCompile(`(?i)\b(sk-[A-Za-z0-9._\-]{8,})`)
	googlePattern = regexp.MustCompile(`\b(AIza[0-9A-Za-z._\-]{10,})`)
)

// Redact removes a known secret and common credential shapes from text that may
// be surfaced in an error message.
func Redact(text, secret string) string {
	out := text
	if secret != "" {
		out = strings.ReplaceAll(out, secret, "[redacted]")
	}
	out = bearerPattern.ReplaceAllString(out, "${1}[redacted]")
	out = skPattern.ReplaceAllString(out, "[redacted]")
	out = googlePattern.ReplaceAllString(out, "[redacted]")
	return out
}

func truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return text[:max] + "…"
}

// doJSON performs one HTTP request and returns the response body. Errors are
// redacted before they can reach a log or the UI.
func (c *Client) doJSON(ctx context.Context, method, target string, headers map[string]string, payload any) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, 0, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s request failed: %s", c.cfg.Name, Redact(err.Error(), c.cfg.APIKey))
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading %s response: %s", c.cfg.Name, Redact(readErr.Error(), c.cfg.APIKey))
	}
	if resp.StatusCode >= 400 {
		return raw, resp.StatusCode, fmt.Errorf("%s returned HTTP %d: %s", c.cfg.Name, resp.StatusCode, truncate(Redact(strings.TrimSpace(string(raw)), c.cfg.APIKey), 800))
	}
	return raw, resp.StatusCode, nil
}

// ExtractJSON pulls the first balanced JSON object/array out of a model reply,
// tolerating markdown fences and surrounding prose.
func ExtractJSON(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", fmt.Errorf("the model returned an empty response")
	}
	// Preferred: a fenced ```json block.
	if idx := strings.Index(trimmed, "```"); idx >= 0 {
		rest := trimmed[idx+3:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[nl+1:]
		}
		if end := strings.Index(rest, "```"); end >= 0 {
			candidate := strings.TrimSpace(rest[:end])
			if json.Valid([]byte(candidate)) {
				return candidate, nil
			}
			trimmed = candidate
		}
	}
	start := -1
	for i, r := range trimmed {
		if r == '{' || r == '[' {
			start = i
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("no JSON found in the model response")
	}
	open := trimmed[start]
	closeCh := byte('}')
	if open == '[' {
		closeCh = ']'
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(trimmed); i++ {
		ch := trimmed[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == open {
			depth++
		}
		if ch == closeCh {
			depth--
			if depth == 0 {
				candidate := trimmed[start : i+1]
				if json.Valid([]byte(candidate)) {
					return candidate, nil
				}
				return "", fmt.Errorf("the model returned malformed JSON")
			}
		}
	}
	return "", fmt.Errorf("the model returned incomplete JSON")
}
