-- +goose Up
-- +goose StatementBegin
ALTER TABLE backup_verification_configs
    RENAME TO verification_configs;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE restore_verifications
    RENAME TO verification_runs;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE restore_verification_table_stats
    RENAME TO verification_run_table_stats;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_run_table_stats
    RENAME COLUMN restore_verification_id TO run_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    RENAME CONSTRAINT backup_verification_configs_pkey TO verification_configs_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    RENAME CONSTRAINT restore_verifications_pkey TO verification_runs_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_run_table_stats
    RENAME CONSTRAINT restore_verification_table_stats_pkey TO verification_run_table_stats_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    RENAME CONSTRAINT fk_restore_verifications_agent TO fk_verification_runs_agent;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_run_table_stats
    RENAME CONSTRAINT fk_restore_verification_table_stats_verification TO fk_verification_run_table_stats_run;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_verification_agents_token_hash
    RENAME TO uq_verification_agents_token_hash;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_restore_verifications_database_created
    RENAME TO idx_verification_runs_database_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_restore_verifications_status_created
    RENAME TO idx_verification_runs_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_restore_verifications_agent_status
    RENAME TO idx_verification_runs_agent_status;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_restore_verification_table_stats_verification
    RENAME TO idx_verification_run_table_stats_run;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_verification_runs_backup
    ON verification_runs (backup_id);
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE verification_runs
SET status       = 'FAILED',
    fail_message = 'The upgrade replaced the unknown status ' || status
WHERE status NOT IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE verification_runs
SET trigger = 'MANUAL'
WHERE trigger NOT IN ('MANUAL', 'SCHEDULED', 'AFTER_BACKUP');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE verification_configs
SET schedule_type = 'AFTER_BACKUP'
WHERE schedule_type NOT IN ('INTERVAL', 'AFTER_BACKUP');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE verification_configs
SET interval_type = 'WEEKLY',
    time_of_day   = COALESCE(NULLIF(time_of_day, ''), '04:00'),
    weekday       = COALESCE(weekday, 0)
WHERE interval_type NOT IN ('HOURLY', 'DAILY', 'WEEKLY', 'MONTHLY', 'CRON');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ALTER COLUMN send_notifications_on DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ALTER COLUMN send_notifications_on TYPE TEXT[]
    USING string_to_array(send_notifications_on, ',');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE verification_configs
SET send_notifications_on = ARRAY(
    SELECT DISTINCT notification
    FROM unnest(send_notifications_on) AS notification
    WHERE notification IN ('VERIFICATION_SUCCESS', 'VERIFICATION_FAILED')
    ORDER BY notification
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ALTER COLUMN send_notifications_on SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    DROP CONSTRAINT fk_backup_verification_configs_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ADD CONSTRAINT fk_verification_configs_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    DROP CONSTRAINT fk_restore_verifications_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    ADD CONSTRAINT fk_verification_runs_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    DROP CONSTRAINT fk_restore_verifications_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    ADD CONSTRAINT fk_verification_runs_backup
    FOREIGN KEY (backup_id)
    REFERENCES backups_logical (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE verification_configs
    DROP CONSTRAINT fk_verification_configs_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ADD CONSTRAINT fk_backup_verification_configs_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    DROP CONSTRAINT fk_verification_runs_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    ADD CONSTRAINT fk_restore_verifications_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    DROP CONSTRAINT fk_verification_runs_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    ADD CONSTRAINT fk_restore_verifications_backup
    FOREIGN KEY (backup_id)
    REFERENCES backups_logical (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ALTER COLUMN send_notifications_on DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ALTER COLUMN send_notifications_on TYPE TEXT
    USING array_to_string(send_notifications_on, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    ALTER COLUMN send_notifications_on SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_verification_runs_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_verification_run_table_stats_run
    RENAME TO idx_restore_verification_table_stats_verification;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_verification_runs_agent_status
    RENAME TO idx_restore_verifications_agent_status;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_verification_runs_status_created_at
    RENAME TO idx_restore_verifications_status_created;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_verification_runs_database_created_at
    RENAME TO idx_restore_verifications_database_created;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX uq_verification_agents_token_hash
    RENAME TO idx_verification_agents_token_hash;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_run_table_stats
    RENAME CONSTRAINT fk_verification_run_table_stats_run TO fk_restore_verification_table_stats_verification;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    RENAME CONSTRAINT fk_verification_runs_agent TO fk_restore_verifications_agent;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_run_table_stats
    RENAME CONSTRAINT verification_run_table_stats_pkey TO restore_verification_table_stats_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    RENAME CONSTRAINT verification_runs_pkey TO restore_verifications_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    RENAME CONSTRAINT verification_configs_pkey TO backup_verification_configs_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_run_table_stats
    RENAME COLUMN run_id TO restore_verification_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_run_table_stats
    RENAME TO restore_verification_table_stats;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_runs
    RENAME TO restore_verifications;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE verification_configs
    RENAME TO backup_verification_configs;
-- +goose StatementEnd
