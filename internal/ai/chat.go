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
	if err != nil && req.JSONMode && status == 400 {
		// Some OpenAI-compatible gateways reject response_format; retry without.
		delete(payload, "response_format")
		raw, _, err = c.doJSON(ctx, "POST", ep.URL, ep.Header, payload)
	}
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage map[string]int `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("%s returned an unexpected response", c.cfg.Name)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("%s: %s", c.cfg.Name, Redact(parsed.Error.Message, c.cfg.APIKey))
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return nil, fmt.Errorf("%s returned no content", c.cfg.Name)
	}
	return &ChatResponse{Content: parsed.Choices[0].Message.Content, Model: parsed.Model, Usage: parsed.Usage}, nil
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
		Usage map[string]int `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("Anthropic returned an unexpected response")
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("Anthropic: %s", Redact(parsed.Error.Message, c.cfg.APIKey))
	}
	var text strings.Builder
	for _, part := range parsed.Content {
		if part.Type == "text" || part.Type == "" {
			text.WriteString(part.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return nil, fmt.Errorf("Anthropic returned no content")
	}
	return &ChatResponse{Content: text.String(), Model: parsed.Model, Usage: parsed.Usage}, nil
}
