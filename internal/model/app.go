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

	// Optional AI assistant (OpenAI-compatible chat-completions endpoint).
	AIEnabled  bool   `json:"aiEnabled"`
	AIProvider string `json:"aiProvider"`
	AIBaseURL  string `json:"aiBaseUrl"`
	AIAPIKey   string `json:"aiApiKey"`
	AIModel    string `json:"aiModel"`
}

// DefaultAppSettings returns production defaults used on a fresh install.
func DefaultAppSettings() AppSettings {
	return AppSettings{
		Theme:                  "system",
		FontSize:               16,
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
