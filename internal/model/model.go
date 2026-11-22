// Package model contains the serialisable data model shared by the backend
// packages and the frontend (via JSON over the Wails bridge).
package model

import "encoding/json"

// KeyValue is a single row in a key/value editor (headers, query params,
// path variables, form fields).
type KeyValue struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Type        string `json:"type,omitempty"` // text | file (form-data)
	Src         string `json:"src,omitempty"`  // absolute file path for uploads
	FileName    string `json:"fileName,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

// UnmarshalJSON defaults Enabled to true when the field is absent. Imported
// Postman data commonly omits it.
func (kv *KeyValue) UnmarshalJSON(data []byte) error {
	var aux struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		Description string `json:"description"`
		Enabled     *bool  `json:"enabled"`
		Type        string `json:"type"`
		Src         string `json:"src"`
		FileName    string `json:"fileName"`
		ContentType string `json:"contentType"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	kv.Key = aux.Key
	kv.Value = aux.Value
	kv.Description = aux.Description
	kv.Enabled = true
	if aux.Enabled != nil {
		kv.Enabled = *aux.Enabled
	}
	kv.Type = aux.Type
	kv.Src = aux.Src
	kv.FileName = aux.FileName
	kv.ContentType = aux.ContentType
	return nil
}

// GraphQLBody is the GraphQL request payload.
type GraphQLBody struct {
	Query     string `json:"query"`
	Variables string `json:"variables"`
}

// Body is the request body union.
type Body struct {
	Mode       string       `json:"mode"`
	Raw        string       `json:"raw,omitempty"`
	FormData   []KeyValue   `json:"formdata,omitempty"`
	URLEncoded []KeyValue   `json:"urlencoded,omitempty"`
	GraphQL    *GraphQLBody `json:"graphql,omitempty"`
	Binary     string       `json:"binary,omitempty"` // base64 (data URL accepted)
	FileName   string       `json:"fileName,omitempty"`
}

type APIKeyAuth struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	AddTo string `json:"addTo"` // header | query
}

type BearerAuth struct {
	Token string `json:"token"`
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type DigestAuth struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Realm     string `json:"realm,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	Qop       string `json:"qop,omitempty"`
}

type OAuth1Auth struct {
	ConsumerKey     string `json:"consumerKey"`
	ConsumerSecret  string `json:"consumerSecret"`
	Token           string `json:"token"`
	TokenSecret     string `json:"tokenSecret"`
	SignatureMethod string `json:"signatureMethod"`
	Version         string `json:"version"`
	Realm           string `json:"realm,omitempty"`
}

type OAuth2Auth struct {
	GrantType    string `json:"grantType"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	AuthURL      string `json:"authUrl"`
	TokenURL     string `json:"tokenUrl"`
	Scope        string `json:"scope"`
	RedirectURL  string `json:"redirectUrl"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	// ExpiresAt is the unix-millisecond expiry of AccessToken (0 = unknown).
	ExpiresAt int64  `json:"expiresAt,omitempty"`
	TokenType string `json:"tokenType,omitempty"`
	// PKCE enables the Proof Key for Code Exchange extension for the
	// authorisation-code flow.
	PKCE bool `json:"pkce,omitempty"`
	// CodeVerifier holds the transient PKCE verifier between the authorization
	// request and the token exchange.
	CodeVerifier string `json:"codeVerifier,omitempty"`
	State        string `json:"state,omitempty"`
}

type AWSV4Auth struct {
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	SessionToken string `json:"sessionToken,omitempty"`
	Region       string `json:"region"`
	Service      string `json:"service"`
}

type HawkAuth struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Algorithm string `json:"algorithm"`
	User      string `json:"user,omitempty"`
}

type NTLMAuth struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Domain      string `json:"domain,omitempty"`
	Workstation string `json:"workstation,omitempty"`
}

type JWTAuth struct {
	Token        string `json:"token"`
	Algorithm    string `json:"algorithm,omitempty"`
	Secret       string `json:"secret,omitempty"`
	Payload      string `json:"payload,omitempty"`
	HeaderPrefix string `json:"headerPrefix,omitempty"`
}

// Auth is the union of all supported authentication mechanisms.
type Auth struct {
	Type   string      `json:"type"`
	APIKey *APIKeyAuth `json:"apikey,omitempty"`
	Bearer *BearerAuth `json:"bearer,omitempty"`
	Basic  *BasicAuth  `json:"basic,omitempty"`
	Digest *DigestAuth `json:"digest,omitempty"`
	OAuth1 *OAuth1Auth `json:"oauth1,omitempty"`
	OAuth2 *OAuth2Auth `json:"oauth2,omitempty"`
	AWSV4  *AWSV4Auth  `json:"awsv4,omitempty"`
	Hawk   *HawkAuth   `json:"hawk,omitempty"`
	NTLM   *NTLMAuth   `json:"ntlm,omitempty"`
	JWT    *JWTAuth    `json:"jwt,omitempty"`
}

type Proxy struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Protocol string   `json:"protocol"` // http | https | socks5
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Bypass   []string `json:"bypass,omitempty"`
}

type TLS struct {
	Cert               string `json:"cert,omitempty"`
	Key                string `json:"key,omitempty"`
	CA                 string `json:"ca,omitempty"`
	Passphrase         string `json:"passphrase,omitempty"`
	RejectUnauthorized bool   `json:"rejectUnauthorized"`
}

type Settings struct {
	FollowRedirects      bool   `json:"followRedirects"`
	FollowOriginalMethod bool   `json:"followOriginalMethod"`
	MaxRedirects         int    `json:"maxRedirects"`
	Timeout              int    `json:"timeout"` // milliseconds
	Encoding             string `json:"encoding"`
	Proxy                *Proxy `json:"proxy,omitempty"`
	TLS                  *TLS   `json:"tls,omitempty"`
}

type Script struct {
	Type     string `json:"type"` // prerequest | test
	Script   string `json:"script"`
	Disabled bool   `json:"disabled,omitempty"`
}

// Request is a saved or unsaved HTTP request.
type Request struct {
	ID            string     `json:"id"`
	CollectionID  string     `json:"collectionId"`
	FolderID      *string    `json:"folderId,omitempty"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Method        string     `json:"method"`
	URL           string     `json:"url"`
	Headers       []KeyValue `json:"headers"`
	QueryParams   []KeyValue `json:"queryParams"`
	PathVariables []KeyValue `json:"pathVariables"`
	Body          Body       `json:"body"`
	Auth          Auth       `json:"auth"`
	Scripts       []Script   `json:"scripts"`
	Settings      Settings   `json:"settings"`
	Order         int        `json:"order"`
	CreatedAt     int64      `json:"createdAt"`
	UpdatedAt     int64      `json:"updatedAt"`
}

// CollectionInfo mirrors the Postman collection `info` object.
type CollectionInfo struct {
	PostmanID   string `json:"_postman_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
	Version     string `json:"version,omitempty"`
}

type Collection struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspaceId"`
	Info        CollectionInfo    `json:"info"`
	Variables   []Variable        `json:"variables"`
	Auth        Auth              `json:"auth"`
	Events      []CollectionEvent `json:"events"`
	CreatedAt   int64             `json:"createdAt"`
	UpdatedAt   int64             `json:"updatedAt"`
}

type CollectionEvent struct {
	Listen string           `json:"listen"`
	Script CollectionScript `json:"script"`
}

type CollectionScript struct {
	Type string   `json:"type"`
	Exec []string `json:"exec"`
}

type Folder struct {
	ID           string            `json:"id"`
	CollectionID string            `json:"collectionId"`
	ParentID     *string           `json:"parentId,omitempty"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Order        int               `json:"order"`
	Auth         *Auth             `json:"auth,omitempty"`
	Events       []CollectionEvent `json:"events,omitempty"`
	CreatedAt    int64             `json:"createdAt"`
	UpdatedAt    int64             `json:"updatedAt"`
}

type Variable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"` // default | secret
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type Environment struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	Name        string     `json:"name"`
	Values      []Variable `json:"values"`
	CreatedAt   int64      `json:"createdAt"`
	UpdatedAt   int64      `json:"updatedAt"`
}

type GlobalVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type Workspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Expires  int64  `json:"expires,omitempty"`
	MaxAge   int    `json:"maxAge,omitempty"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"httpOnly"`
	SameSite string `json:"sameSite,omitempty"`
	HostOnly bool   `json:"hostOnly,omitempty"`
	Session  bool   `json:"session,omitempty"`
}

type TestResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

type ConsoleEntry struct {
	Level string   `json:"level"`
	Args  []string `json:"args"`
}

// Response is the result of executing a request.
type Response struct {
	StatusCode   int            `json:"status"`
	StatusText   string         `json:"statusText"`
	Headers      []KeyValue     `json:"headers"`
	Cookies      []Cookie       `json:"cookies"`
	Body         string         `json:"body"`
	BodyBase64   string         `json:"bodyBase64,omitempty"`
	IsBinary     bool           `json:"isBinary,omitempty"`
	ContentType  string         `json:"contentType,omitempty"`
	BodySize     int64          `json:"bodySize"`
	HeaderSize   int64          `json:"headerSize"`
	ResponseTime int64          `json:"responseTime"` // milliseconds
	Timestamp    int64          `json:"timestamp"`
	Redirected   bool           `json:"redirected"`
	URL          string         `json:"url,omitempty"`
	Error        string         `json:"error,omitempty"`
	Tests        []TestResult   `json:"tests"`
	Console      []ConsoleEntry `json:"console"`
}

// Tab is a persisted editor tab.
type Tab struct {
	ID                  string    `json:"id"`
	RequestID           string    `json:"requestId,omitempty"`
	Title               string    `json:"title"`
	IsDirty             bool      `json:"isDirty"`
	IsActive            bool      `json:"isActive"`
	Protocol            string    `json:"protocol"`
	Request             *Request  `json:"request,omitempty"`
	Response            *Response `json:"response,omitempty"`
	SidebarWidth        int       `json:"sidebarWidth,omitempty"`
	ResponsePanelHeight int       `json:"responsePanelHeight,omitempty"`
	CreatedAt           int64     `json:"createdAt"`
	UpdatedAt           int64     `json:"updatedAt"`
}

type HistoryEntry struct {
	ID          string    `json:"id"`
	Request     Request   `json:"request"`
	Response    *Response `json:"response,omitempty"`
	Timestamp   int64     `json:"timestamp"`
	Duration    int64     `json:"duration"`
	Environment string    `json:"environment,omitempty"`
}

type MockRoute struct {
	ID         string     `json:"id"`
	Method     string     `json:"method"`
	Path       string     `json:"path"`
	StatusCode int        `json:"statusCode"`
	Headers    []KeyValue `json:"headers"`
	Body       string     `json:"body"`
	Delay      int        `json:"delay"`
	MatchQuery bool       `json:"matchQuery"`
}

type MockServer struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Port      int         `json:"port"`
	Routes    []MockRoute `json:"routes"`
	IsRunning bool        `json:"isRunning"`
	CreatedAt int64       `json:"createdAt"`
	UpdatedAt int64       `json:"updatedAt"`
}

type MockRequestLog struct {
	ID        string     `json:"id"`
	Method    string     `json:"method"`
	Path      string     `json:"path"`
	Headers   []KeyValue `json:"headers"`
	Body      string     `json:"body"`
	Status    int        `json:"status"`
	Timestamp int64      `json:"timestamp"`
}

// AppState is returned by Bootstrap and contains everything the UI needs.
type AppState struct {
	Workspace           Workspace        `json:"workspace"`
	Workspaces          []Workspace      `json:"workspaces"`
	Collections         []Collection     `json:"collections"`
	Folders             []Folder         `json:"folders"`
	Requests            []Request        `json:"requests"`
	Environments        []Environment    `json:"environments"`
	ActiveEnvironmentID string           `json:"activeEnvironmentId,omitempty"`
	GlobalVariables     []GlobalVariable `json:"globalVariables"`
	History             []HistoryEntry   `json:"history"`
	Settings            AppSettings      `json:"settings"`
	Tabs                []Tab            `json:"tabs"`
	MockServers         []MockServer     `json:"mockServers"`
	Version             string           `json:"version"`
	Platform            string           `json:"platform"`
}

// ImportPreview describes an imported document before it is persisted.
type ImportPreview struct {
	Format       string       `json:"format"`
	Name         string       `json:"name"`
	Collections  []ImportItem `json:"collections"`
	Environments []ImportEnv  `json:"environments"`
	Warnings     []string     `json:"warnings"`
	Errors       []string     `json:"errors"`
	Raw          string       `json:"raw,omitempty"`
}

type ImportItem struct {
	Name         string `json:"name"`
	FolderCount  int    `json:"folderCount"`
	RequestCount int    `json:"requestCount"`
}

type ImportEnv struct {
	Name   string `json:"name"`
	Values int    `json:"values"`
}

// ImportBundle is the parsed representation ready for persistence.
type ImportBundle struct {
	Format       string
	Collections  []ImportedCollection
	Environments []ImportedEnvironment
	Globals      []Variable
	Warnings     []string
}

type ImportedCollection struct {
	Info      CollectionInfo
	Variables []Variable
	Auth      Auth
	Events    []CollectionEvent
	Folders   []ImportedFolder
	Requests  []Request
}

type ImportedFolder struct {
	TempID       string
	ParentTempID string
	Name         string
	Description  string
	Auth         *Auth
	Events       []CollectionEvent
	Order        int
}

type ImportedEnvironment struct {
	Name   string
	Values []Variable
	Scope  string // environment | global
}

// RunOptions configures a collection run.
type RunOptions struct {
	CollectionID  string   `json:"collectionId"`
	FolderID      string   `json:"folderId,omitempty"`
	RequestIDs    []string `json:"requestIds,omitempty"`
	EnvironmentID string   `json:"environmentId,omitempty"`
	Iterations    int      `json:"iterations"`
	Delay         int      `json:"delay"` // milliseconds
	DataFile      string   `json:"dataFile,omitempty"`
	StopOnError   bool     `json:"stopOnError"`
}

type RunResult struct {
	ID            string            `json:"id"`
	CollectionID  string            `json:"collectionId"`
	StartedAt     int64             `json:"startedAt"`
	CompletedAt   int64             `json:"completedAt,omitempty"`
	Status        string            `json:"status"`
	TotalRequests int               `json:"totalRequests"`
	PassedTests   int               `json:"passedTests"`
	FailedTests   int               `json:"failedTests"`
	TotalTests    int               `json:"totalTests"`
	Iterations    []IterationResult `json:"iterations"`
	Error         string            `json:"error,omitempty"`
}

type IterationResult struct {
	Iteration   int             `json:"iteration"`
	Requests    []RequestResult `json:"requests"`
	StartedAt   int64           `json:"startedAt"`
	CompletedAt int64           `json:"completedAt"`
}

type RequestResult struct {
	RequestID   string       `json:"requestId"`
	RequestName string       `json:"requestName"`
	Status      string       `json:"status"` // passed | failed | error | skipped
	Response    *Response    `json:"response,omitempty"`
	Tests       []TestResult `json:"tests"`
	Duration    int64        `json:"duration"`
	Error       string       `json:"error,omitempty"`
}

// WSConnection is a live WebSocket session exposed to the UI.
type WSConnection struct {
	ID        string      `json:"id"`
	URL       string      `json:"url"`
	Protocols []string    `json:"protocols"`
	Headers   []KeyValue  `json:"headers"`
	Connected bool        `json:"isConnected"`
	Messages  []WSMessage `json:"messages"`
	CreatedAt int64       `json:"createdAt"`
}

// WSMessage is a WebSocket frame or system notice.
type WSMessage struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // sent | received | error | system
	Data      string `json:"data"`
	IsBinary  bool   `json:"isBinary"`
	Timestamp int64  `json:"timestamp"`
}

// StoredCookie associates a cookie with the URL/origin it belongs to. Go's
// cookiejar can only enumerate cookies per URL, so remembering the origins we
// have talked to is what makes the cookie manager possible.
type StoredCookie struct {
	URL    string `json:"url"`
	Cookie Cookie `json:"cookie"`
}

// ExtractionResult is the outcome of a JSONPath extraction from a response
// body.
type ExtractionResult struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Value   string `json:"value"`   // JSON encoding of the extracted value
	Display string `json:"display"` // human-readable form
}

// JSONDiffEntry describes one structural difference between two JSON values.
type JSONDiffEntry struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"` // added | removed | changed
	Left      string `json:"left,omitempty"`
	Right     string `json:"right,omitempty"`
	LeftType  string `json:"leftType,omitempty"`
	RightType string `json:"rightType,omitempty"`
}

// JSONDiff is the result of comparing two JSON documents.
type JSONDiff struct {
	Equal     bool            `json:"equal"`
	Added     int             `json:"added"`
	Removed   int             `json:"removed"`
	Changed   int             `json:"changed"`
	Truncated bool            `json:"truncated"`
	Entries   []JSONDiffEntry `json:"entries"`
}

// AIRequest is the payload for the optional AI assistant. The UI only sends
// request/response data after the user explicitly triggers a prompt.
type AIRequest struct {
	Prompt       string `json:"prompt"`
	RequestJSON  string `json:"requestJson,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`
	SystemPrompt string `json:"systemPrompt,omitempty"`
	// Optional per-call overrides; the central settings remain authoritative
	// for the provider, key and endpoint.
	Model       string   `json:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"maxTokens,omitempty"`
	JSONMode    bool     `json:"jsonMode,omitempty"`
	// Purpose labels the call for logging (never includes prompt contents).
	Purpose string `json:"purpose,omitempty"`
}

// OAuth2StartResult is returned when the authorization URL has been built. The
// Auth value carries the transient PKCE verifier/state that must be kept on the
// request until the code is exchanged.
type OAuth2StartResult struct {
	Auth *OAuth2Auth `json:"auth"`
	URL  string      `json:"url"`
}
