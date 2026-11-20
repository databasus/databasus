-- +goose Up
ALTER TABLE instancebackups_configs
    DROP COLUMN setup_deferred_until;

-- +goose Down
ALTER TABLE instancebackups_configs
    ADD COLUMN setup_deferred_until TIMESTAMPTZ;
