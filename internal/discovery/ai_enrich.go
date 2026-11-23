package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"Postwoman/internal/ai"
	"Postwoman/internal/model"
)

// secretPatterns are redacted from snippets before anything is sent to an AI
// provider. The scanner already skips secret files, but tokens can also appear
// in source or configuration files.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|passwd|pwd|authorization)\s*[:=]\s*["']?[A-Za-z0-9_\-\.\/+=]{8,}["']?`),
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9_\-\.]{12,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9]{16,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
}

type aiSuggestion struct {
	ID           string                 `json:"id"`
	Method       string                 `json:"method,omitempty"`
	Path         string                 `json:"path,omitempty"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	AuthRequired *bool                  `json:"authRequired"`
	AuthType     string                 `json:"authType"`
	RequestBody  any                    `json:"requestBody"`
	QueryParams  []model.DiscoveryParam `json:"queryParams"`
	PathParams   []model.DiscoveryParam `json:"pathParams"`
	Headers      []model.DiscoveryParam `json:"headers"`
	StatusCodes  []int                  `json:"statusCodes"`
	Confidence   int                    `json:"confidence"`
	Notes        []string               `json:"notes"`
}

type aiSuggestionEnvelope struct {
	Endpoints []aiSuggestion `json:"endpoints"`
}

const (
	aiSuggestionBatchSize = 2
	aiInitialMaxTokens    = 8000
	aiRetryMaxTokens      = 16000
)

// aiDiscoverySystemPrompt explicitly forbids the prose/reasoning preamble that
// reasoning models otherwise emit. A verbose preamble is what exhausted the
// completion budget in the reported failure, leaving no room for the JSON.
const aiDiscoverySystemPrompt = "You are an API discovery assistant. You receive route snippets from a software project. " +
	"Reply with one concise JSON object of the form {\"endpoints\":[...]} and nothing else. " +
	"Begin your reply with the character { and end it with }. " +
	"Never invent secrets or values you cannot infer: use null, empty strings or empty arrays instead. " +
	"Do not explain your reasoning, do not think step by step, do not restate the instructions, and do not use markdown fences."

// enrichWithAI asks the configured AI provider to resolve details the static
// analyser could not determine. It sends only small, secret-redacted snippets
// and never the whole project. Failures degrade gracefully to static results.
func (a *analyzer) enrichWithAI(ctx context.Context, settings model.AppSettings, manager *ai.Manager, opts model.DiscoveryOptions, progress func(stage string, percent int, message, current string)) []string {
	warnings := []string{}
	if manager == nil || !settings.AIEnabled {
		return append(warnings, "AI enrichment skipped: AI is disabled in Settings → AI")
	}
	candidates := a.aiCandidates(opts.MaxAIFiles)
	if len(candidates) == 0 {
		return warnings
	}
	progress("ai", 70, fmt.Sprintf("Consulting AI for %d unresolved endpoint(s)", len(candidates)), "")
	batches := chunkSuggestions(candidates, aiSuggestionBatchSize)
	for index, batch := range batches {
		if ctx.Err() != nil {
			return append(warnings, "AI enrichment cancelled")
		}
		progress("ai", 70+int(float64(index)/float64(len(batches))*15), "Analyzing endpoint details with AI", "")
		payload := buildAIPayload(a.profile, batch)
		suggestions, err := requestAISuggestions(ctx, manager, settings, payload, batch)
		if err != nil {
			warnings = append(warnings, "AI enrichment kept static results: "+err.Error())
			continue
		}
		if len(suggestions) == 0 {
			warnings = append(warnings, "AI returned no endpoint suggestions; kept static results")
			continue
		}
		a.applySuggestions(suggestions)
		if len(suggestions) < len(batch) {
			warnings = append(warnings, fmt.Sprintf("AI returned details for %d of %d endpoint(s); the rest kept static data", len(suggestions), len(batch)))
		}
	}
	return warnings
}

// requestAISuggestions asks for one batch and retries once with a larger token
// budget when the provider truncated the reply or the JSON could not be parsed.
// Reasoning models sometimes spend the whole first budget on prose; the retry
// makes that recoverable instead of silently falling back to static data.
func requestAISuggestions(ctx context.Context, manager *ai.Manager, settings model.AppSettings, payload string, expected []*model.DiscoveredEndpoint) ([]aiSuggestion, error) {
	zero := 0.0
	best := []aiSuggestion{}
	var lastErr error
	for attempt, maxTokens := range []int{aiInitialMaxTokens, aiRetryMaxTokens} {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		system := aiDiscoverySystemPrompt
		if attempt > 0 {
			system += " Your previous reply was cut off or was not valid JSON. Respond now with the complete JSON object and no prose."
		}
		reply, err := manager.Chat(ctx, settings, ai.ChatRequest{
			System:      system,
			Messages:    []ai.Message{{Role: "user", Content: payload}},
			JSONMode:    true,
			MaxTokens:   maxTokens,
			Temperature: &zero,
		})
		if err != nil {
			// A partial repaired reply is still useful; otherwise the provider
			// failure will not be fixed by resending.
			if len(best) > 0 {
				return best, nil
			}
			return nil, err
		}
		suggestions, decodeErr := decodeAISuggestions(reply.Content)
		if decodeErr == nil && len(suggestions) > 0 {
			suggestions = matchingSuggestions(suggestions, expected)
			if len(suggestions) == 0 {
				// The model may echo the responseSchema example instead of
				// answering. Treat that as a failed attempt so the retry gets a
				// chance.
				decodeErr = fmt.Errorf("the reply did not reference any requested endpoint")
			}
		}
		if decodeErr == nil && len(suggestions) > 0 {
			if !reply.Truncated() {
				if len(best) > len(suggestions) {
					return best, nil
				}
				return suggestions, nil
			}
			// The reply was cut off but repaired into usable suggestions. Keep
			// the best partial result and retry so the next endpoints survive too.
			if len(suggestions) > len(best) {
				best = suggestions
			}
			lastErr = fmt.Errorf("the reply was cut off at the token limit")
			continue
		}
		if decodeErr == nil {
			decodeErr = fmt.Errorf("the reply contained no endpoint suggestions")
		}
		if reply.Truncated() {
			decodeErr = fmt.Errorf("%w (the reply was cut off at the token limit)", decodeErr)
		}
		lastErr = decodeErr
	}
	if len(best) > 0 {
		return best, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("the AI returned no usable suggestions")
	}
	return nil, lastErr
}

// matchesExpectedEndpoints reports whether at least one suggestion refers to an
// endpoint in the current batch. It accepts either the exact id or a matching
// method+path pair, because models occasionally rewrite ids.
func matchesExpectedEndpoints(suggestions []aiSuggestion, expected []*model.DiscoveredEndpoint) bool {
	return len(matchingSuggestions(suggestions, expected)) > 0
}

// matchingSuggestions drops suggestions that do not refer to any endpoint in the
// batch (for example a repeated responseSchema example).
func matchingSuggestions(suggestions []aiSuggestion, expected []*model.DiscoveredEndpoint) []aiSuggestion {
	if len(expected) == 0 {
		return suggestions
	}
	out := make([]aiSuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		for _, ep := range expected {
			if ep == nil {
				continue
			}
			if suggestion.ID != "" && (strings.EqualFold(suggestion.ID, ep.ID) ||
				strings.EqualFold(suggestion.ID, ep.Method+" "+normalizeRoutePath(ep.Path))) {
				out = append(out, suggestion)
				break
			}
			if suggestion.Method != "" && suggestion.Path != "" &&
				strings.EqualFold(suggestion.Method, ep.Method) &&
				normalizeRoutePath(suggestion.Path) == normalizeRoutePath(ep.Path) {
				out = append(out, suggestion)
				break
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *analyzer) aiCandidates(limit int) []*model.DiscoveredEndpoint {
	if limit <= 0 {
		limit = 40
	}
	candidates := []*model.DiscoveredEndpoint{}
	for _, key := range a.order {
		ep := a.endpoints[key]
		if ep == nil {
			continue
		}
		if ep.Confidence >= 90 && ep.Description != "" && ep.Handler != "" {
			continue
		}
		if ep.Origin == "openapi" {
			continue
		}
		candidates = append(candidates, ep)
		if len(candidates) >= limit {
			break
		}
	}
	return candidates
}

func chunkSuggestions(endpoints []*model.DiscoveredEndpoint, size int) [][]*model.DiscoveredEndpoint {
	if size <= 0 {
		size = 4
	}
	batches := [][]*model.DiscoveredEndpoint{}
	for start := 0; start < len(endpoints); start += size {
		end := start + size
		if end > len(endpoints) {
			end = len(endpoints)
		}
		batches = append(batches, endpoints[start:end])
	}
	return batches
}

func buildAIPayload(profile *projectProfile, endpoints []*model.DiscoveredEndpoint) string {
	type snippet struct {
		ID         string `json:"id"`
		Method     string `json:"method"`
		Path       string `json:"path"`
		Handler    string `json:"handler,omitempty"`
		SourceFile string `json:"sourceFile,omitempty"`
		Framework  string `json:"framework,omitempty"`
		Code       string `json:"code,omitempty"`
	}
	items := make([]snippet, 0, len(endpoints))
	for _, ep := range endpoints {
		items = append(items, snippet{
			ID: ep.ID, Method: ep.Method, Path: ep.Path, Handler: ep.Handler,
			SourceFile: ep.SourceFile, Framework: ep.Framework,
			Code: redactSecrets(ep.HandlerSnippet),
		})
	}
	responseSchema := map[string]any{
		"endpoints": []any{
			map[string]any{
				"id":           "<echo the input id unchanged>",
				"name":         "Short human title",
				"description":  "One or two sentences.",
				"authRequired": false,
				"authType":     "session",
				"requestBody":  nil,
				"queryParams": []any{
					map[string]any{"name": "email", "in": "query", "type": "string", "required": true, "description": "…"},
				},
				"pathParams":  []any{},
				"headers":     []any{},
				"statusCodes": []int{200},
				"confidence":  70,
				"notes":       []string{"Short note."},
			},
		},
	}
	payload := map[string]any{
		"project": map[string]any{
			"languages":  profile.Languages,
			"frameworks": profile.Frameworks,
		},
		"endpoints": items,
		"instructions": "For each input endpoint return: id, name (short human title), description, authRequired (boolean), " +
			"authType (bearer|basic|apikey|oauth2|session|unknown), requestBody (JSON object or null), " +
			"queryParams, pathParams, headers (arrays of {name,in,type,required,description}), " +
			"statusCodes (numbers), confidence (0-100), notes (array of strings). " +
			"Respond with {\"endpoints\":[...]}. Do not include markdown fences.",
		"responseSchema": responseSchema,
		"rules": []string{
			"Return exactly one result for every input endpoint, in the same order.",
			"Copy each input id into the result id unchanged.",
			"Use null for requestBody when there is no body.",
			"Use [] for arrays you cannot determine and null/empty strings for unknown values.",
			"Use at most 3 short notes per endpoint.",
			"Output only JSON. Do not explain your reasoning or restate these instructions.",
		},
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

func redactSecrets(text string) string {
	redacted := text
	for _, pattern := range secretPatterns {
		redacted = pattern.ReplaceAllString(redacted, "[redacted]")
	}
	return redacted
}
