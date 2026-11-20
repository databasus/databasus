-- +goose Up
ALTER TABLE storages
    DROP COLUMN recovery_validation_status;

-- +goose Down
ALTER TABLE storages
    ADD COLUMN recovery_validation_status TEXT NOT NULL DEFAULT 'AVAILABLE';
