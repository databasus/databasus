-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lib_audit_events_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lib_audit_events_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lib_audit_events_author_user_id;
-- +goose StatementEnd

-- The global view and the hourly deletion of expired events read this index.
-- +goose StatementBegin
CREATE INDEX idx_lib_audit_events_created_at_id
    ON lib_audit_events (created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_lib_audit_events_workspace_id_created_at_id
    ON lib_audit_events (workspace_id, created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_lib_audit_events_author_user_id_created_at_id
    ON lib_audit_events (author_user_id, created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lib_audit_events_author_user_id_created_at_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lib_audit_events_workspace_id_created_at_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_lib_audit_events_created_at_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_lib_audit_events_created_at
    ON lib_audit_events (created_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_lib_audit_events_workspace_id
    ON lib_audit_events (workspace_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_lib_audit_events_author_user_id
    ON lib_audit_events (author_user_id);
-- +goose StatementEnd
