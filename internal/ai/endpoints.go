package ai

import (
	"strings"

	"Postwoman/internal/model"
)

// endpoint describes how a provider's HTTP endpoint is reached.
type endpoint struct {
	URL    string
	Header map[string]string
}

// chatEndpoint builds the chat URL and protocol headers for a provider.
func chatEndpoint(cfg model.AIProviderConfig) endpoint {
	base := strings.TrimRight(cfg.BaseURL, "/")
	headers := map[string]string{"Content-Type": "application/json"}
	switch cfg.Kind {
	case "anthropic":
		headers["x-api-key"] = cfg.APIKey
		headers["anthropic-version"] = "2023-06-01"
		url := base
		if !strings.HasSuffix(url, "/messages") {
			url += "/messages"
		}
		return endpoint{URL: url, Header: headers}
	case "gemini":
		url := base
		if !strings.Contains(url, ":generateContent") {
			if strings.Contains(url, "/models/") {
				url += ":generateContent"
			} else {
				url += "/models/" + cfg.Model + ":generateContent"
			}
		}
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		return endpoint{URL: url + sep + "key=" + cfg.APIKey, Header: headers}
	case "azure-openai":
		version := cfg.APIVersion
		if version == "" {
			version = "2024-06-01"
		}
		url := base
		if !strings.Contains(url, "/chat/completions") {
			if strings.Contains(url, "/openai/deployments/") {
				url += "/chat/completions"
			} else {
				url += "/openai/deployments/" + cfg.Model + "/chat/completions"
			}
		}
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		headers["api-key"] = cfg.APIKey
		return endpoint{URL: url + sep + "api-version=" + version, Header: headers}
	case "ollama":
		url := base
		if !strings.HasSuffix(url, "/api/chat") {
			url += "/api/chat"
		}
		return endpoint{URL: url, Header: headers}
	default: // openai, custom and any OpenAI-compatible gateway
		url := base
		if !strings.HasSuffix(url, "/chat/completions") {
			url += "/chat/completions"
		}
		if cfg.APIKey != "" {
			headers["Authorization"] = "Bearer " + cfg.APIKey
		}
		for _, h := range cfg.ExtraHeaders {
			if h.Enabled && h.Key != "" {
				headers[h.Key] = h.Value
			}
		}
		return endpoint{URL: url, Header: headers}
	}
}

// modelsEndpoint returns the model-listing URL and headers for a provider.
func modelsEndpoint(cfg model.AIProviderConfig) endpoint {
	base := strings.TrimRight(cfg.BaseURL, "/")
	headers := map[string]string{"Content-Type": "application/json"}
	switch cfg.Kind {
	case "anthropic":
		headers["x-api-key"] = cfg.APIKey
		headers["anthropic-version"] = "2023-06-01"
		url := base
		if !strings.HasSuffix(url, "/models") {
			url += "/models"
		}
		return endpoint{URL: url, Header: headers}
	case "gemini":
		url := base
		if !strings.HasSuffix(url, "/models") {
			url += "/models"
		}
		return endpoint{URL: url + "?key=" + cfg.APIKey, Header: headers}
	case "azure-openai":
		version := cfg.APIVersion
		if version == "" {
			version = "2024-06-01"
		}
		url := base
		if !strings.HasSuffix(url, "/models") {
			url += "/openai/models"
		}
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		headers["api-key"] = cfg.APIKey
		return endpoint{URL: url + sep + "api-version=" + version, Header: headers}
	case "ollama":
		url := base
		if !strings.HasSuffix(url, "/api/tags") {
			url += "/api/tags"
		}
		return endpoint{URL: url, Header: headers}
	default:
		url := base
		if !strings.HasSuffix(url, "/models") {
			url += "/models"
		}
		if cfg.APIKey != "" {
			headers["Authorization"] = "Bearer " + cfg.APIKey
		}
		for _, h := range cfg.ExtraHeaders {
			if h.Enabled && h.Key != "" {
				headers[h.Key] = h.Value
			}
		}
		return endpoint{URL: url, Header: headers}
	}
}
