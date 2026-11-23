package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Chat sends a provider-agnostic chat request and returns the assistant reply.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if len(req.Messages) == 0 && strings.TrimSpace(req.System) == "" {
		return nil, fmt.Errorf("a message is required")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.cfg.Model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = c.cfg.MaxTokens
	}
	temperature := c.cfg.Temperature
	if req.Temperature != nil {
		temperature = *req.Temperature
	}
	switch c.cfg.Kind {
	case "anthropic":
		return c.chatAnthropic(ctx, req, model, maxTokens, temperature)
	case "gemini":
		return c.chatGemini(ctx, req, model, maxTokens, temperature)
	case "ollama":
		return c.chatOllama(ctx, req, model, maxTokens, temperature)
	default:
		return c.chatOpenAI(ctx, req, model, maxTokens, temperature)
	}
}

func (c *Client) chatOpenAI(ctx context.Context, req ChatRequest, model string, maxTokens int, temperature float64) (*ChatResponse, error) {
	messages := make([]Message, 0, len(req.Messages)+1)
	if system := strings.TrimSpace(req.System); system != "" {
		messages = append(messages, Message{Role: "system", Content: system})
	}
	messages = append(messages, req.Messages...)
	payload := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}
	if req.JSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}
	ep := chatEndpoint(c.cfg)
	raw, status, err := c.doJSON(ctx, "POST", ep.URL, ep.Header, payload)
	if err != nil && status == 400 {
		// OpenAI-compatible gateways differ in which optional parameters they
		// accept: o-series models reject max_tokens, others reject
		// response_format or temperature. Retry once with only the required
		// fields so a single unsupported parameter does not fail the request.
		retry := map[string]any{"model": model, "messages": messages}
		if retryRaw, _, retryErr := c.doJSON(ctx, "POST", ep.URL, ep.Header, retry); retryErr == nil {
			raw, err = retryRaw, nil
		}
		// If the retry also fails, keep the original error: it carries the
		// provider's response body, which is more useful than the second error.
	}
	if err != nil {
		return nil, err
	}
	var parsed openAIChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("%s returned an unreadable response: %s", c.cfg.Name, responseSnippet(raw, c.cfg.APIKey))
	}
	if message := errorMessage(parsed.Error); message != "" {
		return nil, fmt.Errorf("%s: %s", c.cfg.Name, Redact(message, c.cfg.APIKey))
	}
	content := strings.TrimSpace(parsed.content())
	if content == "" {
		return nil, fmt.Errorf("%s returned no content: %s", c.cfg.Name, responseSnippet(raw, c.cfg.APIKey))
	}
	return &ChatResponse{Content: content, Model: parsed.Model, Usage: parseUsage(parsed.Usage), FinishReason: parsed.finishReason()}, nil
}

func (c *Client) chatAnthropic(ctx context.Context, req ChatRequest, model string, maxTokens int, temperature float64) (*ChatResponse, error) {
	messages := make([]Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		role := m.Role
		if role != "assistant" {
			role = "user"
		}
		messages = append(messages, Message{Role: role, Content: m.Content})
	}
	payload := map[string]any{
		"model":       model,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"messages":    messages,
	}
	if system := strings.TrimSpace(req.System); system != "" {
		payload["system"] = system
	}
	ep := chatEndpoint(c.cfg)
	raw, _, err := c.doJSON(ctx, "POST", ep.URL, ep.Header, payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string          `json:"stop_reason"`
		Usage      json.RawMessage `json:"usage"`
		Error      json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("Anthropic returned an unreadable response: %s", responseSnippet(raw, c.cfg.APIKey))
	}
	if message := errorMessage(parsed.Error); message != "" {
		return nil, fmt.Errorf("Anthropic: %s", Redact(message, c.cfg.APIKey))
	}
	var text strings.Builder
	for _, part := range parsed.Content {
		if part.Type == "text" || part.Type == "" {
			text.WriteString(part.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return nil, fmt.Errorf("Anthropic returned no content: %s", responseSnippet(raw, c.cfg.APIKey))
	}
	return &ChatResponse{Content: text.String(), Model: parsed.Model, Usage: parseUsage(parsed.Usage), FinishReason: normalizeFinishReason(parsed.StopReason)}, nil
}
