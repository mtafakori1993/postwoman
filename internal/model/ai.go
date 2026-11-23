package model

// AIProviderConfig is a single configurable AI provider. Providers are stored
// in the global application settings so every AI-powered feature (assistant,
// discovery agent, documentation, generation, …) shares one configuration
// instead of asking the user to configure AI per feature.
type AIProviderConfig struct {
	// ID is a stable identifier ("openai", "anthropic", or a generated id for
	// custom providers). Kind selects the wire protocol.
	ID   string `json:"id"`
	Name string `json:"name"`
	// Kind is the provider protocol: openai | anthropic | gemini |
	// azure-openai | ollama | custom.
	Kind string `json:"kind"`
	// BaseURL is the API root (without the chat/completions suffix).
	BaseURL string `json:"baseUrl"`
	// APIKey is only ever populated on the way in (SaveSettings). Reads from
	// the backend return it empty and set APIKeySet instead so tokens never
	// round-trip through the UI.
	APIKey    string `json:"apiKey,omitempty"`
	APIKeySet bool   `json:"apiKeySet,omitempty"`
	Model     string `json:"model"`
	// Models caches the models discovered from the provider (optional).
	Models         []string   `json:"models,omitempty"`
	Temperature    float64    `json:"temperature"`
	MaxTokens      int        `json:"maxTokens"`
	TimeoutSeconds int        `json:"timeoutSeconds"`
	Custom         bool       `json:"custom,omitempty"`
	ExtraHeaders   []KeyValue `json:"extraHeaders,omitempty"`
	// APIVersion is used by Azure OpenAI deployments.
	APIVersion string `json:"apiVersion,omitempty"`
}

// AIProviderDescriptor describes a provider protocol so the UI can render the
// right fields and defaults without hardcoding provider knowledge.
type AIProviderDescriptor struct {
	Kind                string   `json:"kind"`
	Label               string   `json:"label"`
	Description         string   `json:"description,omitempty"`
	DefaultBaseURL      string   `json:"defaultBaseUrl"`
	DefaultModel        string   `json:"defaultModel"`
	SuggestedModels     []string `json:"suggestedModels"`
	RequiresAPIKey      bool     `json:"requiresApiKey"`
	SupportsModelList   bool     `json:"supportsModelList"`
	SupportsTemperature bool     `json:"supportsTemperature"`
	SupportsMaxTokens   bool     `json:"supportsMaxTokens"`
	DocsURL             string   `json:"docsUrl,omitempty"`
}

// AIConnectionTest is the result of probing a provider with the configured
// credentials. It never contains the API key.
type AIConnectionTest struct {
	OK        bool     `json:"ok"`
	Message   string   `json:"message"`
	LatencyMs int64    `json:"latencyMs"`
	Provider  string   `json:"provider,omitempty"`
	Model     string   `json:"model,omitempty"`
	Models    []string `json:"models,omitempty"`
	Details   string   `json:"details,omitempty"`
}

// AIModelList is returned when a provider supports listing models.
type AIModelList struct {
	Provider string   `json:"provider"`
	Models   []string `json:"models"`
	Warning  string   `json:"warning,omitempty"`
}

// AIStatus reports whether the central AI configuration is usable without
// issuing a network request. Features use it to decide whether to call AI.
type AIStatus struct {
	Enabled  bool   `json:"enabled"`
	Ready    bool   `json:"ready"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Kind     string `json:"kind,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Message  string `json:"message,omitempty"`
}

// ProxyTestResult reports the outcome of probing a proxy host/port from the
// Network settings screen. It never contains proxy credentials.
type ProxyTestResult struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Host      string `json:"host,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	LatencyMs int64  `json:"latencyMs"`
}
