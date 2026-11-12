-- +goose Up
ALTER TABLE backups_physical_wal_segments
    ADD COLUMN last_transaction_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE backups_physical_wal_segments
    DROP COLUMN last_transaction_at;
