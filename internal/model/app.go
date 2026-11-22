package model

// AppSettings is the persisted application configuration. It mirrors the
// AppSettings TypeScript interface so the settings screen round-trips without
// any translation layer.
type AppSettings struct {
	Theme                  string            `json:"theme"`
	FontSize               int               `json:"fontSize"`
	EditorFontSize         int               `json:"editorFontSize"`
	EditorTabSize          int               `json:"editorTabSize"`
	RequestTimeout         int               `json:"requestTimeout"`
	SSLVerification        bool              `json:"sslVerification"`
	Proxy                  *Proxy            `json:"proxy,omitempty"`
	DefaultHeaders         []KeyValue        `json:"defaultHeaders"`
	HistoryRetentionDays   int               `json:"historyRetentionDays"`
	ScriptExecutionEnabled bool              `json:"scriptExecutionEnabled"`
	MaxResponseSize        int               `json:"maxResponseSize"`
	AutoSave               bool              `json:"autoSave"`
	ShowRequestHeaders     bool              `json:"showRequestHeaders"`
	ShowResponseHeaders    bool              `json:"showResponseHeaders"`
	KeyboardShortcuts      map[string]string `json:"keyboardShortcuts"`
	FollowRedirects        bool              `json:"followRedirects"`
	SendNoCacheHeader      bool              `json:"sendNoCacheHeader"`

	// Personalisation
	AccentColor      string `json:"accentColor"`
	EditorFontFamily string `json:"editorFontFamily"`
	// UIFont selects the interface typeface: "system" (sans, default) or
	// "source-code-pro" (bundled monospace).
	UIFont       string `json:"uiFont"`
	CompactMode  bool   `json:"compactMode"`
	LayoutPreset string `json:"layoutPreset"`

	// Centralised AI configuration shared by every AI-powered feature.
	//
	// The flat fields below mirror the active provider for backwards
	// compatibility with older builds. AIProviders holds the full provider
	// list (including custom OpenAI-compatible providers) and AIProviderID
	// selects the active one.
	AIEnabled        bool               `json:"aiEnabled"`
	AIProvider       string             `json:"aiProvider"`
	AIBaseURL        string             `json:"aiBaseUrl"`
	AIAPIKey         string             `json:"aiApiKey"`
	AIModel          string             `json:"aiModel"`
	AITemperature    float64            `json:"aiTemperature"`
	AIMaxTokens      int                `json:"aiMaxTokens"`
	AITimeoutSeconds int                `json:"aiTimeoutSeconds"`
	AIProviders      []AIProviderConfig `json:"aiProviders"`
	AIProviderID     string             `json:"aiProviderId"`
	// AIAPIKeySet reports whether a key is stored without revealing it.
	AIAPIKeySet bool `json:"aiApiKeySet"`
	// AIClearAPIKey is a transient request flag: set by the UI to remove the
	// stored key. It is never persisted.
	AIClearAPIKey bool `json:"aiClearApiKey,omitempty"`
}

// DefaultAppSettings returns production defaults used on a fresh install.
func DefaultAppSettings() AppSettings {
	return AppSettings{
		Theme:                  "system",
		FontSize:               15,
		EditorFontSize:         13,
		EditorTabSize:          2,
		RequestTimeout:         30000,
		SSLVerification:        true,
		DefaultHeaders:         []KeyValue{},
		HistoryRetentionDays:   30,
		ScriptExecutionEnabled: true,
		MaxResponseSize:        50 * 1024 * 1024,
		AutoSave:               true,
		ShowRequestHeaders:     true,
		ShowResponseHeaders:    true,
		KeyboardShortcuts:      map[string]string{},
		FollowRedirects:        true,
		SendNoCacheHeader:      false,
		AccentColor:            "orange",
		EditorFontFamily:       "mono",
		UIFont:                 "system",
		CompactMode:            false,
		LayoutPreset:           "default",
		AIEnabled:              false,
		AIProvider:             "openai",
		AIBaseURL:              "https://api.openai.com/v1",
		AIModel:                "gpt-4o-mini",
		AITemperature:          0.2,
		AIMaxTokens:            2048,
		AITimeoutSeconds:       60,
		AIProviders:            []AIProviderConfig{DefaultAIProvider()},
		AIProviderID:           "openai",
	}
}

// DefaultAIProvider returns the OpenAI provider preset used on a fresh install.
func DefaultAIProvider() AIProviderConfig {
	return AIProviderConfig{
		ID:             "openai",
		Name:           "OpenAI",
		Kind:           "openai",
		BaseURL:        "https://api.openai.com/v1",
		Model:          "gpt-4o-mini",
		Temperature:    0.2,
		MaxTokens:      2048,
		TimeoutSeconds: 60,
	}
}

// ImportSummary reports what an import operation produced.
type ImportSummary struct {
	Format       string   `json:"format"`
	Collections  []string `json:"collections"`
	Environments []string `json:"environments"`
	Warnings     []string `json:"warnings"`
	Message      string   `json:"message,omitempty"`
}

// SearchResult is one hit from the global search palette.
type SearchResult struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Parent string `json:"parent,omitempty"`
	Match  string `json:"match"`
	Score  int    `json:"score"`
}

// SendRequestInput is accepted by App.SendRequest when the UI prefers a single
// structured argument over positional ones.
type SendRequestInput struct {
	Request             Request           `json:"request"`
	Environment         map[string]string `json:"environment"`
	EnvironmentID       string            `json:"environmentId,omitempty"`
	Globals             map[string]string `json:"globals"`
	CollectionVariables map[string]string `json:"collectionVariables"`
	RunScripts          bool              `json:"runScripts"`
}

// ActiveAIProvider resolves the provider the user selected, falling back to the
// legacy flat fields when the provider list is empty (older installs).
func (s AppSettings) ActiveAIProvider() AIProviderConfig {
	var active AIProviderConfig
	found := false
	for _, p := range s.AIProviders {
		if p.ID != "" && p.ID == s.AIProviderID {
			active, found = p, true
			break
		}
	}
	if !found && len(s.AIProviders) > 0 {
		// Fall back to the provider whose kind matches the legacy selector.
		for _, p := range s.AIProviders {
			if p.Kind == s.AIProvider {
				active, found = p, true
				break
			}
		}
		if !found {
			active, found = s.AIProviders[0], true
		}
	}
	if !found {
		active = AIProviderConfig{
			ID:      orDefault(s.AIProvider, "openai"),
			Name:    orDefault(s.AIProvider, "OpenAI"),
			Kind:    orDefault(s.AIProvider, "openai"),
			BaseURL: s.AIBaseURL,
			Model:   s.AIModel,
		}
	}
	if active.Kind == "" {
		active.Kind = orDefault(s.AIProvider, "openai")
	}
	if active.BaseURL == "" {
		active.BaseURL = s.AIBaseURL
	}
	if active.ID == "" {
		active.ID = active.Kind
	}
	if active.Model == "" {
		active.Model = s.AIModel
	}
	if active.APIKey == "" {
		active.APIKey = s.AIAPIKey
	}
	if active.Temperature <= 0 {
		active.Temperature = s.AITemperature
	}
	if active.MaxTokens <= 0 {
		active.MaxTokens = s.AIMaxTokens
	}
	if active.TimeoutSeconds <= 0 {
		active.TimeoutSeconds = s.AITimeoutSeconds
	}
	if active.TimeoutSeconds <= 0 {
		active.TimeoutSeconds = 60
	}
	return active
}

// SanitizedForUI strips secrets from a settings value before it crosses the
// Wails bridge. API keys are replaced with a boolean flag so the UI can show
// "key configured" without ever receiving the token.
func (s AppSettings) SanitizedForUI() AppSettings {
	out := s
	out.AIClearAPIKey = false
	anyKey := s.AIAPIKey != ""
	out.AIAPIKey = ""
	providers := make([]AIProviderConfig, len(s.AIProviders))
	for i, p := range s.AIProviders {
		if p.APIKey != "" {
			anyKey = true
		}
		p.APIKey = ""
		p.APIKeySet = s.AIProviders[i].APIKey != ""
		providers[i] = p
	}
	out.AIProviders = providers
	out.AIAPIKeySet = anyKey
	return out
}

// MergeAIKeys preserves stored credentials when an incoming settings payload
// omits them (the UI never receives keys, so it cannot echo them back) and
// honours the explicit "clear API key" flag.
func (s AppSettings) MergeAIKeys(stored AppSettings) AppSettings {
	out := s
	if !s.AIClearAPIKey && s.AIAPIKey == "" {
		out.AIAPIKey = stored.AIAPIKey
	}
	if s.AIClearAPIKey {
		out.AIAPIKey = ""
	}
	byID := map[string]string{}
	for _, p := range stored.AIProviders {
		byID[p.ID] = p.APIKey
	}
	providers := make([]AIProviderConfig, len(s.AIProviders))
	for i, p := range s.AIProviders {
		if !s.AIClearAPIKey && p.APIKey == "" {
			p.APIKey = byID[p.ID]
		}
		p.APIKeySet = false
		providers[i] = p
	}
	out.AIProviders = providers
	out.AIClearAPIKey = false
	out.AIAPIKeySet = false
	return out
}

func orDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
