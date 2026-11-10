-- +goose Up
ALTER TABLE backups_physical_wal_streamers
    ADD COLUMN error_reason TEXT;

-- +goose Down
ALTER TABLE backups_physical_wal_streamers
    DROP COLUMN error_reason;
