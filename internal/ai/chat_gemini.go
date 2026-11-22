package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) chatGemini(ctx context.Context, req ChatRequest, model string, maxTokens int, temperature float64) (*ChatResponse, error) {
	contents := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]string{{"text": m.Content}},
		})
	}
	gen := map[string]any{
		"temperature":     temperature,
		"maxOutputTokens": maxTokens,
	}
	if req.JSONMode {
		gen["responseMimeType"] = "application/json"
	}
	payload := map[string]any{
		"contents":         contents,
		"generationConfig": gen,
	}
	if system := strings.TrimSpace(req.System); system != "" {
		payload["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": system}},
		}
	}
	ep := chatEndpoint(c.cfg)
	raw, _, err := c.doJSON(ctx, "POST", ep.URL, ep.Header, payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback *struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("Gemini returned an unexpected response")
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("Gemini: %s", Redact(parsed.Error.Message, c.cfg.APIKey))
	}
	if parsed.PromptFeedback != nil && parsed.PromptFeedback.BlockReason != "" {
		return nil, fmt.Errorf("Gemini blocked the request (%s)", parsed.PromptFeedback.BlockReason)
	}
	if len(parsed.Candidates) == 0 {
		return nil, fmt.Errorf("Gemini returned no candidates")
	}
	var text strings.Builder
	for _, part := range parsed.Candidates[0].Content.Parts {
		text.WriteString(part.Text)
	}
	if strings.TrimSpace(text.String()) == "" {
		return nil, fmt.Errorf("Gemini returned no content")
	}
	return &ChatResponse{Content: text.String(), Model: model}, nil
}

func (c *Client) chatOllama(ctx context.Context, req ChatRequest, model string, maxTokens int, temperature float64) (*ChatResponse, error) {
	messages := make([]Message, 0, len(req.Messages)+1)
	if system := strings.TrimSpace(req.System); system != "" {
		messages = append(messages, Message{Role: "system", Content: system})
	}
	messages = append(messages, req.Messages...)
	payload := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
		"options": map[string]any{
			"temperature": temperature,
			"num_predict": maxTokens,
		},
	}
	if req.JSONMode {
		payload["format"] = "json"
	}
	ep := chatEndpoint(c.cfg)
	raw, _, err := c.doJSON(ctx, "POST", ep.URL, ep.Header, payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Model   string `json:"model"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("Ollama returned an unexpected response")
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("Ollama: %s", Redact(parsed.Error, c.cfg.APIKey))
	}
	if strings.TrimSpace(parsed.Message.Content) == "" {
		return nil, fmt.Errorf("Ollama returned no content")
	}
	return &ChatResponse{Content: parsed.Message.Content, Model: parsed.Model}, nil
}
