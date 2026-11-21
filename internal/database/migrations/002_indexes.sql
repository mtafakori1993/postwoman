-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_collections_workspace ON collections(workspace_id);
CREATE INDEX IF NOT EXISTS idx_folders_collection ON folders(collection_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_requests_collection ON requests(collection_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_requests_folder ON requests(folder_id);
CREATE INDEX IF NOT EXISTS idx_environments_workspace ON environments(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tabs_created ON tabs(created_at);
CREATE INDEX IF NOT EXISTS idx_history_timestamp ON request_history(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_mock_servers_port ON mock_servers(port);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_mock_servers_port;
DROP INDEX IF EXISTS idx_history_timestamp;
DROP INDEX IF EXISTS idx_tabs_created;
DROP INDEX IF EXISTS idx_environments_workspace;
DROP INDEX IF EXISTS idx_requests_folder;
DROP INDEX IF EXISTS idx_requests_collection;
DROP INDEX IF EXISTS idx_folders_collection;
DROP INDEX IF EXISTS idx_collections_workspace;
-- +goose StatementEnd
