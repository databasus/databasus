-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_backups_physical_storage_id
    ON backups_physical (storage_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_physical_wal_segments_storage_id
    ON backups_physical_wal_segments (storage_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_physical_wal_history_files_storage_id
    ON backups_physical_wal_history_files (storage_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_databases_notifiers_notifier_id
    ON databases_notifiers (notifier_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_databases_notifiers_notifier_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_backups_physical_wal_history_files_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_backups_physical_wal_segments_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_backups_physical_storage_id;
-- +goose StatementEnd
