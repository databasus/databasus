-- +goose Up
-- +goose StatementBegin
ALTER TABLE logical_backup_configs
    RENAME TO backups_configs_logical;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    RENAME CONSTRAINT backup_configs_pkey TO backups_configs_logical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_backup_configs
    RENAME TO backups_configs_physical;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    RENAME CONSTRAINT physical_backup_configs_pkey TO backups_configs_physical_pkey;
-- +goose StatementEnd

-- A database keeps only the config of its own method; no code path reads the other one.
-- +goose StatementBegin
DELETE FROM backups_configs_logical
USING databases
WHERE databases.id = backups_configs_logical.database_id
  AND databases.type = 'POSTGRES_PHYSICAL';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM backups_configs_physical
USING databases
WHERE databases.id = backups_configs_physical.database_id
  AND databases.type <> 'POSTGRES_PHYSICAL';
-- +goose StatementEnd

-- The repairs replace only values that no validation accepts with the defaults of
-- internal/modules/backups/internal/config/defaults.go.
-- +goose StatementBegin
UPDATE backups_configs_logical
SET retention_policy_type = 'TIME_PERIOD'
WHERE retention_policy_type NOT IN ('TIME_PERIOD', 'COUNT', 'GFS');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_logical
SET retention_time_period = '3_MONTH'
WHERE retention_time_period NOT IN (
    'DAY', 'WEEK', 'MONTH', '3_MONTH', '6_MONTH', 'YEAR', '2_YEARS', '3_YEARS', '4_YEARS', '5_YEARS', 'FOREVER'
);
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_logical
SET interval_type = 'DAILY',
    time_of_day   = COALESCE(time_of_day, '04:00')
WHERE interval_type NOT IN ('HOURLY', 'DAILY', 'WEEKLY', 'MONTHLY', 'CRON');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_logical
SET encryption = 'NONE'
WHERE encryption NOT IN ('NONE', 'ENCRYPTED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_physical
SET full_interval_type = 'DAILY',
    full_time_of_day   = COALESCE(full_time_of_day, '04:00')
WHERE full_interval_type NOT IN ('HOURLY', 'DAILY', 'WEEKLY', 'MONTHLY', 'CRON');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_physical
SET incremental_interval_type = ''
WHERE incremental_interval_type NOT IN ('', 'HOURLY', 'DAILY', 'WEEKLY', 'MONTHLY', 'CRON');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_physical
SET retention = 'FULL_BACKUPS'
WHERE retention NOT IN ('CHAINS', 'FULL_BACKUPS', 'CHAINS_AND_FULL_BACKUPS');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_physical
SET full_backups_retention_policy = 'LAST_N'
WHERE full_backups_retention_policy NOT IN ('', 'LAST_N', 'GFS');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_physical
SET encryption = 'NONE'
WHERE encryption NOT IN ('NONE', 'ENCRYPTED');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE backups_configs (
    database_id           UUID    NOT NULL,
    method                TEXT    NOT NULL,
    is_backups_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    storage_id            UUID,
    encryption            TEXT    NOT NULL DEFAULT 'NONE',
    notification_triggers TEXT[]  NOT NULL DEFAULT '{}'
);
-- +goose StatementEnd

-- The comma-joined triggers keep the order of their first appearance, without empty, duplicate and
-- unknown entries.
-- +goose StatementBegin
INSERT INTO backups_configs (database_id, method, is_backups_enabled, storage_id, encryption, notification_triggers)
SELECT
    database_id,
    'LOGICAL',
    is_backups_enabled,
    storage_id,
    encryption,
    ARRAY(
        SELECT entries.trigger
        FROM (
            SELECT btrim(entry) AS trigger, min(position) AS first_position
            FROM unnest(string_to_array(send_notifications_on, ',')) WITH ORDINALITY AS split(entry, position)
            WHERE btrim(entry) IN ('BACKUP_FAILED', 'BACKUP_SUCCESS')
            GROUP BY btrim(entry)
        ) AS entries
        ORDER BY entries.first_position
    )
FROM backups_configs_logical;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO backups_configs (database_id, method, is_backups_enabled, storage_id, encryption, notification_triggers)
SELECT
    database_id,
    'PHYSICAL',
    is_backups_enabled,
    storage_id,
    encryption,
    ARRAY(
        SELECT entries.trigger
        FROM (
            SELECT btrim(entry) AS trigger, min(position) AS first_position
            FROM unnest(string_to_array(send_notifications_on, ',')) WITH ORDINALITY AS split(entry, position)
            WHERE btrim(entry) IN ('BACKUP_FAILED', 'BACKUP_SUCCESS', 'CHAIN_BROKEN', 'WAL_GAP')
            GROUP BY btrim(entry)
        ) AS entries
        ORDER BY entries.first_position
    )
FROM backups_configs_physical;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs
    ADD CONSTRAINT backups_configs_pkey
    PRIMARY KEY (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs
    ADD CONSTRAINT uq_backups_configs_database_method
    UNIQUE (database_id, method);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs
    ADD CONSTRAINT fk_backups_configs_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs
    ADD CONSTRAINT fk_backups_configs_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_configs_storage_id
    ON backups_configs (storage_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    DROP CONSTRAINT fk_backup_config_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    DROP CONSTRAINT fk_backup_config_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    DROP COLUMN is_backups_enabled,
    DROP COLUMN storage_id,
    DROP COLUMN encryption,
    DROP COLUMN send_notifications_on;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    ADD COLUMN method TEXT NOT NULL DEFAULT 'LOGICAL';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    ADD CONSTRAINT fk_backups_configs_logical_config
    FOREIGN KEY (database_id, method)
    REFERENCES backups_configs (database_id, method)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    ALTER COLUMN retention_time_period SET DEFAULT '3_MONTH';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    DROP CONSTRAINT fk_physical_backup_configs_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    DROP CONSTRAINT fk_physical_backup_configs_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    DROP COLUMN is_backups_enabled,
    DROP COLUMN storage_id,
    DROP COLUMN encryption,
    DROP COLUMN send_notifications_on;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    ADD COLUMN method TEXT NOT NULL DEFAULT 'PHYSICAL';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    ADD CONSTRAINT fk_backups_configs_physical_config
    FOREIGN KEY (database_id, method)
    REFERENCES backups_configs (database_id, method)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    ALTER COLUMN full_interval_type DROP DEFAULT;
-- +goose StatementEnd

-- An empty incremental type is the schedule of a database without incremental backups.
-- An empty full-backups policy is the retention of a config that keeps only chains.
-- +goose Down
-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    ALTER COLUMN full_interval_type SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    ALTER COLUMN retention_time_period SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    DROP CONSTRAINT fk_backups_configs_physical_config,
    DROP COLUMN method;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    DROP CONSTRAINT fk_backups_configs_logical_config,
    DROP COLUMN method;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    ADD COLUMN is_backups_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN storage_id            UUID,
    ADD COLUMN encryption            TEXT    NOT NULL DEFAULT 'NONE',
    ADD COLUMN send_notifications_on TEXT    NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    ADD COLUMN is_backups_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN storage_id            UUID,
    ADD COLUMN encryption            TEXT    NOT NULL DEFAULT 'NONE',
    ADD COLUMN send_notifications_on TEXT    NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_logical
SET is_backups_enabled    = backups_configs.is_backups_enabled,
    storage_id            = backups_configs.storage_id,
    encryption            = backups_configs.encryption,
    send_notifications_on = array_to_string(backups_configs.notification_triggers, ',')
FROM backups_configs
WHERE backups_configs.database_id = backups_configs_logical.database_id;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_configs_physical
SET is_backups_enabled    = backups_configs.is_backups_enabled,
    storage_id            = backups_configs.storage_id,
    encryption            = backups_configs.encryption,
    send_notifications_on = array_to_string(backups_configs.notification_triggers, ',')
FROM backups_configs
WHERE backups_configs.database_id = backups_configs_physical.database_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE backups_configs;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    ADD CONSTRAINT fk_backup_config_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    ADD CONSTRAINT fk_backup_config_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    ADD CONSTRAINT fk_physical_backup_configs_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    ADD CONSTRAINT fk_physical_backup_configs_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    RENAME CONSTRAINT backups_configs_physical_pkey TO physical_backup_configs_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_physical
    RENAME TO physical_backup_configs;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    RENAME CONSTRAINT backups_configs_logical_pkey TO backup_configs_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_configs_logical
    RENAME TO logical_backup_configs;
-- +goose StatementEnd
