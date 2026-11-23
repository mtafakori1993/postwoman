package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"Postwoman/internal/model"
)

// ListModels returns the model identifiers advertised by the provider.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	ep := modelsEndpoint(c.cfg)
	raw, _, err := c.doJSON(ctx, "GET", ep.URL, ep.Header, nil)
	if err != nil {
		return nil, err
	}
	models := []string{}
	switch c.cfg.Kind {
	case "anthropic":
		var parsed struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("Anthropic returned an unexpected model list")
		}
		for _, entry := range parsed.Data {
			models = append(models, entry.ID)
		}
	case "gemini":
		var parsed struct {
			Models []struct {
				Name                       string   `json:"name"`
				SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("Gemini returned an unexpected model list")
		}
		for _, entry := range parsed.Models {
			if len(entry.SupportedGenerationMethods) > 0 {
				supported := false
				for _, m := range entry.SupportedGenerationMethods {
					if m == "generateContent" {
						supported = true
						break
					}
				}
				if !supported {
					continue
				}
			}
			models = append(models, strings.TrimPrefix(entry.Name, "models/"))
		}
	case "ollama":
		var parsed struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("Ollama returned an unexpected model list")
		}
		for _, entry := range parsed.Models {
			models = append(models, entry.Name)
		}
	default:
		var parsed struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("%s returned an unexpected model list", c.cfg.Name)
		}
		for _, entry := range parsed.Data {
			models = append(models, entry.ID)
		}
	}
	sort.Strings(models)
	return models, nil
}

// TestConnection verifies credentials and connectivity. It first tries the
// model list and falls back to a tiny chat completion, which works for
// deployments that do not expose /models.
func (c *Client) TestConnection(ctx context.Context) model.AIConnectionTest {
	result := model.AIConnectionTest{Provider: c.cfg.Name, Model: c.cfg.Model}
	started := time.Now()
	models, err := c.ListModels(ctx)
	if err == nil && len(models) > 0 {
		result.OK = true
		result.LatencyMs = time.Since(started).Milliseconds()
		result.Models = models
		result.Message = fmt.Sprintf("Connected to %s — %d model(s) available.", c.cfg.Name, len(models))
		return result
	}
	listErr := err
	zero := 0.0
	reply, chatErr := c.Chat(ctx, ChatRequest{
		Messages:    []Message{{Role: "user", Content: "Reply with the single word OK."}},
		MaxTokens:   16,
		Temperature: &zero,
	})
	result.LatencyMs = time.Since(started).Milliseconds()
	if chatErr != nil {
		result.OK = false
		if listErr != nil {
			result.Message = "Connection failed: " + listErr.Error()
		} else {
			result.Message = "Connection failed: " + chatErr.Error()
		}
		return result
	}
	result.OK = true
	result.Message = fmt.Sprintf("Connected to %s (model responded).", c.cfg.Name)
	if len(models) > 0 {
		result.Models = models
	}
	result.Details = truncate(strings.TrimSpace(reply.Content), 200)
	return result
}

// Manager is the shared AI service used by every feature.
type Manager struct{}

// NewManager creates the shared AI service.
func NewManager() *Manager { return &Manager{} }

// ClientFor resolves the globally configured provider and returns a client that
// honours the AI proxy scope. AI must be enabled centrally; features never read
// keys themselves.
func (m *Manager) ClientFor(settings model.AppSettings) (*Client, error) {
	if !settings.AIEnabled {
		return nil, fmt.Errorf("AI is disabled; enable it in Settings → AI")
	}
	cfg := settings.ActiveAIProvider()
	client, err := NewClientWithProxy(cfg, settings.AIProxy())
	if err != nil {
		return nil, fmt.Errorf("AI provider is not configured: %w", err)
	}
	return client, nil
}

// Chat runs a chat completion with the centrally configured provider.
func (m *Manager) Chat(ctx context.Context, settings model.AppSettings, req ChatRequest) (*ChatResponse, error) {
	client, err := m.ClientFor(settings)
	if err != nil {
		return nil, err
	}
	return client.Chat(ctx, req)
}

// MergeProviderSecrets copies stored credentials into an incoming provider
// config when the UI omits them (the UI never receives keys).
func MergeProviderSecrets(cfg model.AIProviderConfig, settings model.AppSettings) model.AIProviderConfig {
	if cfg.APIKey != "" {
		return cfg
	}
	if cfg.ID != "" && cfg.ID == settings.AIProviderID || cfg.ID == "" {
		cfg.APIKey = settings.ActiveAIProvider().APIKey
		return cfg
	}
	for _, p := range settings.AIProviders {
		if p.ID == cfg.ID {
			cfg.APIKey = p.APIKey
			return cfg
		}
	}
	cfg.APIKey = settings.AIAPIKey
	return cfg
}

// NormalizeProvider is exposed for callers that need the resolved config (e.g.
// the discovery agent deciding whether AI is usable) without issuing a request.
func NormalizeProvider(cfg model.AIProviderConfig) (model.AIProviderConfig, error) {
	return Normalize(cfg)
}
