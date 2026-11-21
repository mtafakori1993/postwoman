-- +goose Up
-- +goose StatementBegin

-- Workspaces
CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Collections
CREATE TABLE collections (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    info_json TEXT NOT NULL, -- JSON encoded CollectionInfo
    variables_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded CollectionVariable[]
    auth_json TEXT, -- JSON encoded AuthConfig
    events_json TEXT, -- JSON encoded CollectionEvent[]
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Folders
CREATE TABLE folders (
    id TEXT PRIMARY KEY,
    collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    "order" INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Requests
CREATE TABLE requests (
    id TEXT PRIMARY KEY,
    collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    folder_id TEXT REFERENCES folders(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    method TEXT NOT NULL,
    url TEXT NOT NULL,
    headers_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded Header[]
    query_params_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded QueryParam[]
    path_variables_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded KeyValuePair[]
    body_json TEXT NOT NULL, -- JSON encoded RequestBody
    auth_json TEXT NOT NULL, -- JSON encoded RequestAuth
    scripts_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded RequestScript[]
    settings_json TEXT NOT NULL, -- JSON encoded RequestSettings
    "order" INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Environments
CREATE TABLE environments (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    values_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded EnvironmentVariable[]
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Global Variables
CREATE TABLE global_variables (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'default', -- 'default' or 'secret'
    description TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    updated_at INTEGER NOT NULL
);

-- Request History
CREATE TABLE request_history (
    id TEXT PRIMARY KEY,
    request_json TEXT NOT NULL, -- JSON encoded Request
    response_json TEXT, -- JSON encoded Response (nullable for failed requests)
    timestamp INTEGER NOT NULL,
    duration INTEGER NOT NULL
);

CREATE INDEX idx_request_history_timestamp ON request_history(timestamp DESC);

-- Tabs
CREATE TABLE tabs (
    id TEXT PRIMARY KEY,
    request_id TEXT REFERENCES requests(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    is_dirty INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 0,
    protocol TEXT NOT NULL DEFAULT 'http',
    request_json TEXT, -- JSON encoded Request (for unsaved tabs)
    response_json TEXT, -- JSON encoded Response
    sidebar_width INTEGER,
    response_panel_height INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- App Settings
CREATE TABLE app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL, -- JSON encoded value
    updated_at INTEGER NOT NULL
);

-- Mock Servers
CREATE TABLE mock_servers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    port INTEGER NOT NULL,
    routes_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded MockRoute[]
    is_running INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Monitors
CREATE TABLE monitors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    request_id TEXT NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    environment_id TEXT REFERENCES environments(id) ON DELETE SET NULL,
    interval INTEGER NOT NULL, -- minutes
    assertions_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded AssertionConfig[]
    is_active INTEGER NOT NULL DEFAULT 1,
    last_run INTEGER,
    next_run INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- API Specifications
CREATE TABLE api_specs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    type TEXT NOT NULL, -- 'openapi', 'swagger', 'graphql', 'asyncapi'
    specification_json TEXT NOT NULL, -- JSON encoded specification
    endpoints_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded APIEndpoint[]
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- WebSocket Connections
CREATE TABLE ws_connections (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    protocols_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded string[]
    headers_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded Header[]
    is_connected INTEGER NOT NULL DEFAULT 0,
    messages_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded WebSocketMessage[]
    created_at INTEGER NOT NULL
);

-- gRPC Connections
CREATE TABLE grpc_connections (
    id TEXT PRIMARY KEY,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    proto_files_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded string[]
    services_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded GRPCService[]
    is_connected INTEGER NOT NULL DEFAULT 0,
    metadata_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded Header[]
    tls_json TEXT, -- JSON encoded TLSConfig
    created_at INTEGER NOT NULL
);

-- MQTT Connections
CREATE TABLE mqtt_connections (
    id TEXT PRIMARY KEY,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    client_id TEXT NOT NULL,
    username TEXT,
    password TEXT, -- Encrypted
    tls_json TEXT, -- JSON encoded TLSConfig
    topics_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded MQTTTopic[]
    is_connected INTEGER NOT NULL DEFAULT 0,
    messages_json TEXT NOT NULL DEFAULT '[]', -- JSON encoded MQTTMessage[]
    created_at INTEGER NOT NULL
);

-- Active Workspace
CREATE TABLE active_workspace (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    active_environment_id TEXT REFERENCES environments(id) ON DELETE SET NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS active_workspace;
DROP TABLE IF EXISTS mqtt_connections;
DROP TABLE IF EXISTS grpc_connections;
DROP TABLE IF EXISTS ws_connections;
DROP TABLE IF EXISTS api_specs;
DROP TABLE IF EXISTS monitors;
DROP TABLE IF EXISTS mock_servers;
DROP TABLE IF EXISTS app_settings;
DROP TABLE IF EXISTS tabs;
DROP TABLE IF EXISTS request_history;
DROP TABLE IF EXISTS global_variables;
DROP TABLE IF EXISTS environments;
DROP TABLE IF EXISTS requests;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS workspaces;

-- +goose StatementEnd