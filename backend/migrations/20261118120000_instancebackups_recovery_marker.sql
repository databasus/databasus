-- +goose Up
ALTER TABLE instancebackups_configs
    RENAME COLUMN recovery_paused_operation_id TO recovered_by_operation_id;

ALTER TABLE instancebackups_configs
    RENAME COLUMN recovery_paused_at TO recovered_at;

ALTER TABLE instancebackups_configs
    RENAME CONSTRAINT chk_instancebackups_configs_recovery_pause TO chk_instancebackups_configs_recovery_marker;

-- +goose Down
ALTER TABLE instancebackups_configs
    RENAME CONSTRAINT chk_instancebackups_configs_recovery_marker TO chk_instancebackups_configs_recovery_pause;

ALTER TABLE instancebackups_configs
    RENAME COLUMN recovered_at TO recovery_paused_at;

ALTER TABLE instancebackups_configs
    RENAME COLUMN recovered_by_operation_id TO recovery_paused_operation_id;
