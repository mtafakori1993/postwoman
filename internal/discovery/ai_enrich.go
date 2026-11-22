package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

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
	batches := chunkSuggestions(candidates, 4)
	for index, batch := range batches {
		if ctx.Err() != nil {
			return append(warnings, "AI enrichment cancelled")
		}
		progress("ai", 70+int(float64(index)/float64(len(batches))*15), "Analyzing endpoint details with AI", "")
		payload := buildAIPayload(a.profile, batch)
		reply, err := manager.Chat(ctx, settings, ai.ChatRequest{
			System: "You are an API discovery assistant. You receive route snippets from a software project. " +
				"Return ONLY a JSON object matching the requested schema. Never invent secrets or values you cannot infer. " +
				"Use empty strings or omit fields when unsure.",
			Messages:  []ai.Message{{Role: "user", Content: payload}},
			JSONMode:  true,
			MaxTokens: 3000,
		})
		if err != nil {
			warnings = append(warnings, "AI enrichment failed: "+err.Error())
			continue
		}
		content, err := ai.ExtractJSON(reply.Content)
		if err != nil {
			warnings = append(warnings, "AI returned unreadable JSON; kept static results")
			continue
		}
		var envelope aiSuggestionEnvelope
		if err := json.Unmarshal([]byte(content), &envelope); err != nil {
			warnings = append(warnings, "AI returned malformed suggestions; kept static results")
			continue
		}
		a.applySuggestions(envelope.Endpoints)
	}
	return warnings
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
	payload := map[string]any{
		"project": map[string]any{
			"languages":  profile.Languages,
			"frameworks": profile.Frameworks,
		},
		"endpoints": items,
		"instructions": "For each endpoint return: id, name (short human title), description, authRequired (boolean), " +
			"authType (bearer|basic|apikey|oauth2|session|unknown), requestBody (JSON object or null), " +
			"queryParams, pathParams, headers (arrays of {name,in,type,required,description}), " +
			"statusCodes (numbers), confidence (0-100), notes (array of strings). " +
			"Respond with {\"endpoints\":[...]}. Do not include markdown fences.",
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
