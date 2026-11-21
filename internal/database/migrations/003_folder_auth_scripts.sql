-- +goose Up
-- +goose StatementBegin
ALTER TABLE folders ADD COLUMN auth_json TEXT;
ALTER TABLE folders ADD COLUMN events_json TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE folders DROP COLUMN events_json;
ALTER TABLE folders DROP COLUMN auth_json;
-- +goose StatementEnd
