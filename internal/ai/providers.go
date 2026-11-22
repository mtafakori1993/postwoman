// Package ai is the single AI service layer used by every AI-powered feature in
// Postwoman. It owns provider protocol details (endpoints, authentication
// headers, request/response shapes), so UI components and feature packages never
// talk to a provider directly or handle API keys themselves.
package ai

import (
	"errors"
	"fmt"
	"strings"

	"Postwoman/internal/model"
)

// Descriptors returns the built-in provider protocols. The UI renders the
// provider picker from this list and "custom" lets users add any
// OpenAI-compatible endpoint.
func Descriptors() []model.AIProviderDescriptor {
	return []model.AIProviderDescriptor{
		{
			Kind:                "openai",
			Label:               "OpenAI",
			Description:         "Official OpenAI chat-completions API.",
			DefaultBaseURL:      "https://api.openai.com/v1",
			DefaultModel:        "gpt-4o-mini",
			SuggestedModels:     []string{"gpt-4o-mini", "gpt-4o", "gpt-4.1", "gpt-4.1-mini", "o3-mini"},
			RequiresAPIKey:      true,
			SupportsModelList:   true,
			SupportsTemperature: true,
			SupportsMaxTokens:   true,
			DocsURL:             "https://platform.openai.com/docs/api-reference",
		},
		{
			Kind:                "anthropic",
			Label:               "Anthropic",
			Description:         "Claude models via the Messages API.",
			DefaultBaseURL:      "https://api.anthropic.com/v1",
			DefaultModel:        "claude-3-5-sonnet-latest",
			SuggestedModels:     []string{"claude-3-5-sonnet-latest", "claude-3-5-haiku-latest", "claude-3-opus-latest"},
			RequiresAPIKey:      true,
			SupportsModelList:   true,
			SupportsTemperature: true,
			SupportsMaxTokens:   true,
			DocsURL:             "https://docs.anthropic.com/en/api",
		},
		{
			Kind:                "gemini",
			Label:               "Google Gemini",
			Description:         "Google Generative Language API.",
			DefaultBaseURL:      "https://generativelanguage.googleapis.com/v1beta",
			DefaultModel:        "gemini-2.0-flash",
			SuggestedModels:     []string{"gemini-2.0-flash", "gemini-1.5-flash", "gemini-1.5-pro"},
			RequiresAPIKey:      true,
			SupportsModelList:   true,
			SupportsTemperature: true,
			SupportsMaxTokens:   true,
			DocsURL:             "https://ai.google.dev/api",
		},
		{
			Kind:                "azure-openai",
			Label:               "Azure OpenAI",
			Description:         "Azure-hosted OpenAI deployments (uses the deployment name as the model).",
			DefaultBaseURL:      "https://YOUR-RESOURCE.openai.azure.com",
			DefaultModel:        "gpt-4o-mini",
			SuggestedModels:     []string{"gpt-4o-mini", "gpt-4o", "gpt-35-turbo"},
			RequiresAPIKey:      true,
			SupportsModelList:   true,
			SupportsTemperature: true,
			SupportsMaxTokens:   true,
			DocsURL:             "https://learn.microsoft.com/azure/ai-services/openai/",
		},
		{
			Kind:                "ollama",
			Label:               "Ollama (local)",
			Description:         "Local models served by Ollama. No API key required by default.",
			DefaultBaseURL:      "http://localhost:11434",
			DefaultModel:        "llama3.1",
			SuggestedModels:     []string{"llama3.1", "qwen2.5-coder", "codellama", "mistral"},
			RequiresAPIKey:      false,
			SupportsModelList:   true,
			SupportsTemperature: true,
			SupportsMaxTokens:   true,
			DocsURL:             "https://github.com/ollama/ollama/blob/main/docs/api.md",
		},
		{
			Kind:                "custom",
			Label:               "Custom OpenAI-compatible",
			Description:         "Any endpoint that implements /chat/completions (Groq, OpenRouter, vLLM, LM Studio, …).",
			DefaultBaseURL:      "",
			DefaultModel:        "",
			RequiresAPIKey:      false,
			SupportsModelList:   true,
			SupportsTemperature: true,
			SupportsMaxTokens:   true,
		},
	}
}

// Descriptor returns the descriptor for a provider kind (or the custom
// descriptor when the kind is unknown).
func Descriptor(kind string) model.AIProviderDescriptor {
	kind = strings.ToLower(strings.TrimSpace(kind))
	for _, d := range Descriptors() {
		if d.Kind == kind {
			return d
		}
	}
	if kind == "" {
		return Descriptors()[0]
	}
	return Descriptors()[len(Descriptors())-1]
}

// Normalize fills provider defaults, clamps numeric settings and validates that
// enough information is present to make a request. It never inspects or logs the
// API key.
func Normalize(cfg model.AIProviderConfig) (model.AIProviderConfig, error) {
	cfg.Kind = strings.ToLower(strings.TrimSpace(cfg.Kind))
	if cfg.Kind == "" {
		cfg.Kind = "openai"
	}
	d := Descriptor(cfg.Kind)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.BaseURL == "" {
		cfg.BaseURL = d.DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Model == "" {
		cfg.Model = d.DefaultModel
	}
	if cfg.Temperature < 0 {
		cfg.Temperature = 0
	}
	if cfg.Temperature > 2 {
		cfg.Temperature = 2
	}
	if cfg.Temperature == 0 && cfg.MaxTokens == 0 && cfg.TimeoutSeconds == 0 {
		// A completely unconfigured provider: apply friendly defaults.
		cfg.Temperature = 0.2
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 2048
	}
	if cfg.MaxTokens > 200000 {
		cfg.MaxTokens = 200000
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 60
	}
	if cfg.TimeoutSeconds > 600 {
		cfg.TimeoutSeconds = 600
	}
	if cfg.ID == "" {
		cfg.ID = cfg.Kind
	}
	if cfg.Name == "" {
		cfg.Name = d.Label
	}
	if d.RequiresAPIKey && strings.TrimSpace(cfg.APIKey) == "" {
		return cfg, fmt.Errorf("%s requires an API key", d.Label)
	}
	if cfg.Kind == "custom" && cfg.BaseURL == "" {
		return cfg, errors.New("a custom provider needs a base URL")
	}
	if cfg.Model == "" {
		return cfg, fmt.Errorf("%s needs a model name", d.Label)
	}
	return cfg, nil
}
