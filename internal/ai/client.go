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
	// FinishReason is normalised to "stop", "length", "content_filter" or the
	// provider's own value. "length" means the reply may be incomplete, which
	// callers that expect JSON must treat as a retryable truncation.
	FinishReason string `json:"finishReason,omitempty"`
}

// Truncated reports whether the provider stopped because it hit the completion
// token limit. A truncated reply may contain only part of the requested JSON.
func (r *ChatResponse) Truncated() bool {
	return r != nil && (r.FinishReason == "length" || r.FinishReason == "max_tokens")
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

// ExtractJSON pulls the first complete JSON object/array out of a model reply.
// Real providers rarely return pure JSON: some wrap the payload in markdown
// fences, prepend a sentence of prose, or (for reasoning models) emit a long
// explanation first. All of that is tolerated here. When a reply is cut off at
// the token limit we salvage the last complete portion instead of discarding the
// whole batch.
func ExtractJSON(text string) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	if trimmed == "" {
		return "", fmt.Errorf("the model returned an empty response")
	}
	if candidate, ok := normalizeJSONCandidate(trimmed); ok {
		return candidate, nil
	}

	// Fenced ```json blocks. Scan every fence because a model may include a
	// short example before the real answer.
	for _, block := range fencedBlocks(trimmed) {
		if candidate, ok := normalizeJSONCandidate(block); ok {
			return candidate, nil
		}
	}

	// Scan for a balanced value. If the first opening bracket belongs to a
	// truncated payload, repair it before considering nested objects (otherwise
	// a nested endpoint object would be returned instead of the envelope).
	for start := 0; start < len(trimmed); start++ {
		if trimmed[start] != '{' && trimmed[start] != '[' {
			continue
		}
		candidate, complete, matched := balancedValue(trimmed[start:])
		if !matched {
			continue
		}
		if complete {
			if normalized, ok := normalizeJSONCandidate(candidate); ok {
				return normalized, nil
			}
			continue
		}
		if repaired, ok := repairTruncatedJSON(candidate); ok {
			return repaired, nil
		}
	}
	return "", fmt.Errorf("no complete JSON object or array found in the model response")
}

// fencedBlocks returns the contents of every ``` fenced block in text,
// regardless of the optional language tag on the opening line.
func fencedBlocks(text string) []string {
	blocks := []string{}
	for {
		start := strings.Index(text, "```")
		if start < 0 {
			break
		}
		rest := text[start+3:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[nl+1:]
		} else if brace := strings.IndexAny(rest, "{["); brace >= 0 {
			// Support one-line fences such as ```json {"endpoints":[]} ```.
			rest = rest[brace:]
		} else {
			rest = ""
		}
		end := strings.Index(rest, "```")
		if end < 0 {
			blocks = append(blocks, strings.TrimSpace(rest))
			break
		}
		blocks = append(blocks, strings.TrimSpace(rest[:end]))
		text = rest[end+3:]
	}
	return blocks
}

// normalizeJSONCandidate accepts a candidate that is already valid JSON, has
// harmless trailing commas, or was truncated and can be closed safely.
func normalizeJSONCandidate(text string) (string, bool) {
	candidate := strings.TrimSpace(text)
	if candidate == "" {
		return "", false
	}
	if json.Valid([]byte(candidate)) {
		return candidate, true
	}
	stripped := stripTrailingCommas(candidate)
	if stripped != candidate && json.Valid([]byte(stripped)) {
		return stripped, true
	}
	if repaired, ok := repairTruncatedJSON(candidate); ok {
		return repaired, true
	}
	return "", false
}

// balancedValue returns the prefix of s that starts with the given opening
// bracket and ends where its matching bracket closes. complete reports whether
// that closing bracket was found; matched reports whether brackets nested
// correctly up to that point.
func balancedValue(s string) (candidate string, complete, matched bool) {
	stack := make([]byte, 0, 8)
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
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
		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, ch)
		case '}', ']':
			if len(stack) == 0 {
				return s[:i+1], false, false
			}
			open := stack[len(stack)-1]
			if (ch == '}' && open != '{') || (ch == ']' && open != '[') {
				return s[:i+1], false, false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return s[:i+1], true, true
			}
		}
	}
	return s, false, true
}

// repairTruncatedJSON salvages a reply cut off at the token limit by dropping
// the incomplete trailing value and closing the still-open objects/arrays. It
// only accepts the result when the repaired text is valid JSON.
func repairTruncatedJSON(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}
	// Collect element boundaries outside strings. Repair is tried from the end
	// backwards, so the incomplete tail is discarded while everything before it
	// is preserved. A bounded number of attempts keeps this cheap for big replies.
	cuts := []int{len(trimmed)}
	inString := false
	escaped := false
	for i := 0; i < len(trimmed); i++ {
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
		switch ch {
		case '"':
			inString = true
		case ',':
			cuts = append(cuts, i)
		}
	}
	// Each attempt scans the prefix. Keep the worst case bounded for unusually
	// large provider replies while still trying the most recent boundaries.
	maxAttempts := 512
	if len(trimmed) > 256*1024 {
		maxAttempts = 32
	}
	if len(cuts) > maxAttempts {
		cuts = cuts[len(cuts)-maxAttempts:]
	}
	for i := len(cuts) - 1; i >= 0; i-- {
		prefix := strings.TrimRight(trimmed[:cuts[i]], " \t\r\n")
		if repaired, ok := closeJSON(prefix); ok {
			return repaired, true
		}
	}
	return "", false
}

// closeJSON appends the missing closers to a syntactically complete prefix. It
// refuses prefixes that end inside a string or after a key separator because
// those cannot be repaired without inventing data.
func closeJSON(prefix string) (string, bool) {
	stack := make([]byte, 0, 8)
	inString := false
	escaped := false
	for i := 0; i < len(prefix); i++ {
		ch := prefix[i]
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
		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, ch)
		case '}', ']':
			if len(stack) == 0 {
				return "", false
			}
			open := stack[len(stack)-1]
			if (ch == '}' && open != '{') || (ch == ']' && open != '[') {
				return "", false
			}
			stack = stack[:len(stack)-1]
		}
	}
	if inString || escaped {
		return "", false
	}
	prefix = strings.TrimRight(prefix, " \t\r\n")
	if strings.HasSuffix(prefix, ",") {
		prefix = strings.TrimRight(strings.TrimSuffix(prefix, ","), " \t\r\n")
	}
	if strings.HasSuffix(prefix, ":") {
		return "", false
	}
	var builder strings.Builder
	builder.WriteString(prefix)
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == '{' {
			builder.WriteByte('}')
		} else {
			builder.WriteByte(']')
		}
	}
	candidate := strings.TrimSpace(builder.String())
	if !json.Valid([]byte(candidate)) {
		return "", false
	}
	return candidate, true
}

// stripTrailingCommas removes commas that appear immediately before a closing
// brace/bracket. Several models emit those even in JSON mode.
func stripTrailingCommas(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	inString := false
	escaped := false
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inString {
			builder.WriteByte(ch)
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
		switch ch {
		case '"':
			inString = true
			builder.WriteByte(ch)
		case ',':
			next := i + 1
			for next < len(text) {
				switch text[next] {
				case ' ', '\t', '\r', '\n':
					next++
					continue
				}
				break
			}
			if next < len(text) && (text[next] == '}' || text[next] == ']') {
				continue
			}
			builder.WriteByte(ch)
		default:
			builder.WriteByte(ch)
		}
	}
	return builder.String()
}
