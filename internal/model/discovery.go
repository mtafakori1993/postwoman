package model

// DiscoveryOptions configures a project API discovery run. The zero value is
// filled in by discovery.Defaults().
type DiscoveryOptions struct {
	Directory        string   `json:"directory"`
	CollectionName   string   `json:"collectionName"`
	BaseURL          string   `json:"baseUrl"`
	ExcludePatterns  []string `json:"excludePatterns"`
	IncludeTests     bool     `json:"includeTests"`
	RespectGitignore bool     `json:"respectGitignore"`
	UseAI            bool     `json:"useAi"`
	MaxFiles         int      `json:"maxFiles"`
	MaxFileSizeKB    int      `json:"maxFileSizeKb"`
	MaxAIFiles       int      `json:"maxAiFiles"`
	LanguageHint     string   `json:"languageHint,omitempty"`
	FrameworkHint    string   `json:"frameworkHint,omitempty"`
	FollowSymlinks   bool     `json:"followSymlinks"`
	IncludeHidden    bool     `json:"includeHidden"`
}

// DiscoveryProgress is emitted while a run is in flight so the UI can render an
// agent-style progress log instead of a blocking spinner.
type DiscoveryProgress struct {
	SessionID string `json:"sessionId"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	Percent   int    `json:"percent"`
	Current   string `json:"current,omitempty"`
	FilesSeen int    `json:"filesSeen"`
	Endpoints int    `json:"endpoints"`
	Done      bool   `json:"done"`
	Error     string `json:"error,omitempty"`
}

// DiscoveryFileInfo describes one project file the agent considered.
type DiscoveryFileInfo struct {
	Path      string `json:"path"`
	Language  string `json:"language"`
	Category  string `json:"category"` // source|route|model|config|schema|doc|test|other
	Bytes     int64  `json:"bytes"`
	Lines     int    `json:"lines"`
	HasRoutes bool   `json:"hasRoutes"`
	Analyzed  bool   `json:"analyzed"`
	SentToAI  bool   `json:"sentToAi"`
	Skipped   string `json:"skipped,omitempty"`
}

// DiscoverySchemaField is one field of an inferred request/response schema.
type DiscoverySchemaField struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Required    bool                   `json:"required"`
	Example     string                 `json:"example,omitempty"`
	Description string                 `json:"description,omitempty"`
	Children    []DiscoverySchemaField `json:"children,omitempty"`
}

// DiscoveryParam is a path/query/header parameter of a discovered endpoint.
type DiscoveryParam struct {
	Name        string `json:"name"`
	In          string `json:"in"` // path|query|header|cookie|body
	Type        string `json:"type,omitempty"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

// DiscoveredEndpoint is a single API endpoint discovered by the agent. Fields
// that could not be determined are left empty rather than guessed.
type DiscoveredEndpoint struct {
	ID             string                 `json:"id"`
	Method         string                 `json:"method"`
	Path           string                 `json:"path"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	SourceFile     string                 `json:"sourceFile,omitempty"`
	Line           int                    `json:"line,omitempty"`
	Handler        string                 `json:"handler,omitempty"`
	HandlerFile    string                 `json:"handlerFile,omitempty"`
	Controller     string                 `json:"controller,omitempty"`
	Folder         string                 `json:"folder"`
	Tags           []string               `json:"tags,omitempty"`
	AuthRequired   bool                   `json:"authRequired"`
	AuthType       string                 `json:"authType,omitempty"`
	AuthNotes      string                 `json:"authNotes,omitempty"`
	ContentType    string                 `json:"contentType,omitempty"`
	Headers        []DiscoveryParam       `json:"headers,omitempty"`
	PathParams     []DiscoveryParam       `json:"pathParams,omitempty"`
	QueryParams    []DiscoveryParam       `json:"queryParams,omitempty"`
	RequestBody    string                 `json:"requestBody,omitempty"`
	RequestSchema  []DiscoverySchemaField `json:"requestSchema,omitempty"`
	ResponseSchema []DiscoverySchemaField `json:"responseSchema,omitempty"`
	StatusCodes    []int                  `json:"statusCodes,omitempty"`
	Related        []string               `json:"related,omitempty"`
	Confidence     int                    `json:"confidence"`
	Notes          []string               `json:"notes,omitempty"`
	Warnings       []string               `json:"warnings,omitempty"`
	Origin         string                 `json:"origin"` // static|openapi|ai|merged
	Framework      string                 `json:"framework,omitempty"`
	Selected       bool                   `json:"selected"`
	// HandlerSnippet is the local source excerpt used for AI enrichment. It is
	// never serialised to the UI or persisted in the generated collection.
	HandlerSnippet string `json:"-"`
}

// DiscoveryStats summarises a run.
type DiscoveryStats struct {
	FilesSeen      int   `json:"filesSeen"`
	FilesAnalyzed  int   `json:"filesAnalyzed"`
	FilesSkipped   int   `json:"filesSkipped"`
	BytesScanned   int64 `json:"bytesScanned"`
	RouteFiles     int   `json:"routeFiles"`
	Endpoints      int   `json:"endpoints"`
	HighConfidence int   `json:"highConfidence"`
	LowConfidence  int   `json:"lowConfidence"`
	AIFilesSent    int   `json:"aiFilesSent"`
	OpenAPISources int   `json:"openApiSources"`
}

// DiscoverySession is the full state of a discovery run, returned to the UI for
// review before anything is imported.
type DiscoverySession struct {
	ID          string               `json:"id"`
	Status      string               `json:"status"` // running|completed|failed|cancelled
	Options     DiscoveryOptions     `json:"options"`
	Root        string               `json:"root"`
	ProjectName string               `json:"projectName"`
	Languages   []string             `json:"languages"`
	Frameworks  []string             `json:"frameworks"`
	Stats       DiscoveryStats       `json:"stats"`
	Files       []DiscoveryFileInfo  `json:"files"`
	Endpoints   []DiscoveredEndpoint `json:"endpoints"`
	Folders     []string             `json:"folders"`
	Warnings    []string             `json:"warnings"`
	Progress    DiscoveryProgress    `json:"progress"`
	StartedAt   int64                `json:"startedAt"`
	FinishedAt  int64                `json:"finishedAt,omitempty"`
	Error       string               `json:"error,omitempty"`
	AISummary   string               `json:"aiSummary,omitempty"`
	Collection  string               `json:"collectionJson,omitempty"`
}

// DiscoveryDirEntry is one row of the project directory browser.
type DiscoveryDirEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size"`
	Excluded bool   `json:"excluded"`
	Reason   string `json:"reason,omitempty"`
}

// DiscoveryDirListing is a single directory level for the browser.
type DiscoveryDirListing struct {
	Path    string              `json:"path"`
	Parent  string              `json:"parent,omitempty"`
	Entries []DiscoveryDirEntry `json:"entries"`
}

// DiscoveryEndpointEdit carries the user's review decisions back to the
// generator (rename, re-describe, move folder, include/exclude).
type DiscoveryEndpointEdit struct {
	ID          string `json:"id"`
	Selected    bool   `json:"selected"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Folder      string `json:"folder"`
}

// DiscoveryGenerateRequest asks the backend to render the reviewed endpoints as
// an import-ready Postwoman/Postman v2.1 collection.
type DiscoveryGenerateRequest struct {
	SessionID      string                  `json:"sessionId"`
	CollectionName string                  `json:"collectionName"`
	BaseURL        string                  `json:"baseUrl"`
	Endpoints      []DiscoveryEndpointEdit `json:"endpoints"`
}

// DiscoverySourceFile is a window into a project file for the review screen.
type DiscoverySourceFile struct {
	Path       string `json:"path"`
	Language   string `json:"language"`
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
	TotalLines int    `json:"totalLines"`
	Content    string `json:"content"`
	Truncated  bool   `json:"truncated"`
}
