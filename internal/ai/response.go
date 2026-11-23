package ai

import (
	"encoding/json"
	"strconv"
	"strings"
)

// flexString accepts the several shapes providers use for message text: a plain
// string, null, a number, or an array of content parts ({"type":"text","text":…}
// or bare strings) that some gateways return instead of a single string. Any of
// these would otherwise make a strict `string` field reject the whole response.
type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*f = ""
		return nil
	}
	switch trimmed[0] {
	case '"':
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*f = flexString(value)
		return nil
	case '[':
		var parts []json.RawMessage
		if err := json.Unmarshal(data, &parts); err != nil {
			return err
		}
		var builder strings.Builder
		for _, part := range parts {
			var value string
			if err := json.Unmarshal(part, &value); err == nil {
				builder.WriteString(value)
				continue
			}
			var object struct {
				Text    flexString `json:"text"`
				Content flexString `json:"content"`
			}
			if err := json.Unmarshal(part, &object); err == nil {
				builder.WriteString(firstNonEmpty(string(object.Text), string(object.Content)))
			}
		}
		*f = flexString(builder.String())
		return nil
	default:
		*f = flexString(trimmed)
		return nil
	}
}

// openAIMessage holds the several fields different providers use to carry the
// assistant text. `content` is the standard one; some gateways put the text in
// `reasoning_content`/`reasoning` (reasoning models) instead.
type openAIMessage struct {
	Content          flexString `json:"content"`
	ReasoningContent flexString `json:"reasoning_content"`
	Reasoning        flexString `json:"reasoning"`
}

func (m openAIMessage) text() string {
	return firstNonEmpty(string(m.Content), string(m.ReasoningContent), string(m.Reasoning))
}

// openAIChatResponse is a deliberately permissive view of an OpenAI-compatible
// chat response. `Usage` and `Error` are kept as raw JSON because providers
// return wildly different shapes for them (nested token-detail objects, string
// errors, `{"error": {...}}`, …), and a strict struct would make the whole
// response unparsable.
type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      openAIMessage `json:"message"`
		Delta        openAIMessage `json:"delta"`
		Text         flexString    `json:"text"`
		FinishReason string        `json:"finishReason"`
	} `json:"choices"`
	// Some non-OpenAI endpoints (Ollama, Copilot-style gateways) return a single
	// top-level message instead of a choices array.
	Message openAIMessage `json:"message"`
	// Legacy completions and some proxies expose the text at the top level.
	Content flexString `json:"content"`
	Text    flexString `json:"text"`

	Usage json.RawMessage `json:"usage"`
	Error json.RawMessage `json:"error"`
}

// content returns the best available assistant text for one response, checking
// the standard choices array first and then the alternative single-message
// shapes.
func (r openAIChatResponse) content() string {
	for _, choice := range r.Choices {
		if text := firstNonEmpty(choice.Message.text(), choice.Delta.text(), string(choice.Text)); text != "" {
			return text
		}
	}
	return firstNonEmpty(r.Message.text(), string(r.Content), string(r.Text))
}

// errorMessage extracts a human-readable message from the many shapes the
// `error` field takes: a string, {"message": …}, {"detail": …}, {"error": …} or
// null. It returns "" when there is no error.
func errorMessage(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	var asObject struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(raw, &asObject); err == nil {
		return firstNonEmpty(asObject.Message, asObject.Detail, asObject.Error)
	}
	return ""
}

// parseUsage keeps the top-level integer counters and ignores nested detail
// objects (prompt_tokens_details, completion_tokens_details, …) that would make
// a strict map[string]int decode fail.
func parseUsage(raw json.RawMessage) map[string]int {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil
	}
	usage := map[string]int{}
	for key, value := range generic {
		switch v := value.(type) {
		case float64:
			usage[key] = int(v)
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				usage[key] = n
			}
		}
	}
	if len(usage) == 0 {
		return nil
	}
	return usage
}

// responseSnippet returns a short, redacted view of a raw response body so an
// error explains what actually came back instead of a bare "unexpected
// response".
func responseSnippet(raw []byte, secret string) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "the response body was empty"
	}
	return truncate(Redact(trimmed, secret), 300)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
