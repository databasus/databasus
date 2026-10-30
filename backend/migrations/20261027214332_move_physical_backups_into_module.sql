-- +goose Up
-- +goose StatementBegin
ALTER TABLE physical_full_backups
    RENAME TO backups_physical_full;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    RENAME CONSTRAINT physical_full_backups_pkey TO backups_physical_full_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    RENAME CONSTRAINT fk_physical_full_backups_storage_id TO fk_backups_physical_full_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_full_backups_database_id_status_created_at
    RENAME TO idx_backups_physical_full_database_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_full_backups_database_id_timeline_id
    RENAME TO idx_backups_physical_full_database_timeline_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_incremental_backups
    RENAME TO backups_physical_incremental;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT physical_incremental_backups_pkey TO backups_physical_incremental_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT fk_physical_incremental_backups_storage_id TO fk_backups_physical_incremental_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT fk_physical_incremental_backups_root_full_backup_id TO fk_backups_physical_incremental_root_full_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT fk_physical_incremental_backups_parent_incremental_backup_id TO fk_backups_physical_incremental_parent_incremental_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_incremental_backups_root_full_start_lsn
    RENAME TO idx_backups_physical_incremental_root_full_start_lsn;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_incremental_backups_database_id_status_created_at
    RENAME TO idx_backups_physical_incremental_database_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_incremental_backups_parent_incremental_backup_id
    RENAME TO idx_backups_physical_incremental_parent_incremental_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_in_flight_backups
    RENAME TO backups_physical_in_flight;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    RENAME CONSTRAINT physical_in_flight_backups_pkey TO backups_physical_in_flight_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    RENAME CONSTRAINT chk_physical_in_flight_backups_backup_type TO ck_backups_physical_in_flight_backup_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_segments
    RENAME TO backups_physical_wal_segments;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    RENAME CONSTRAINT physical_wal_segments_pkey TO backups_physical_wal_segments_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    RENAME CONSTRAINT fk_physical_wal_segments_storage_id TO fk_backups_physical_wal_segments_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    RENAME CONSTRAINT uk_physical_wal_segments_database_id_timeline_id_start_lsn TO uq_backups_physical_wal_segments_database_timeline_start_lsn;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_wal_segments_database_id_received_at
    RENAME TO idx_backups_physical_wal_segments_database_received_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_history_files
    RENAME TO backups_physical_wal_history_files;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    RENAME CONSTRAINT physical_wal_history_files_pkey TO backups_physical_wal_history_files_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    RENAME CONSTRAINT fk_physical_wal_history_files_storage_id TO fk_backups_physical_wal_history_files_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    RENAME CONSTRAINT uk_physical_wal_history_files_database_id_timeline_id TO uq_backups_physical_wal_history_files_database_timeline;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_streamers
    RENAME TO backups_physical_wal_streamers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_streamers
    RENAME CONSTRAINT physical_wal_streamers_pkey TO backups_physical_wal_streamers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_stream_samples
    RENAME TO backups_physical_wal_stream_samples;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_stream_samples
    RENAME CONSTRAINT physical_wal_stream_samples_pkey TO backups_physical_wal_stream_samples_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_wal_stream_samples_database_id_sampled_at
    RENAME TO idx_backups_physical_wal_stream_samples_database_sampled_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_physical_wal_stream_samples_sampled_at
    RENAME TO idx_backups_physical_wal_stream_samples_sampled_at;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_full
SET status = 'ERROR',
    fail_message = 'The upgrade replaced the unknown status ' || status
WHERE status NOT IN ('IN_PROGRESS', 'COMPLETED', 'ERROR', 'CHAIN_BROKEN', 'CANCELED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_full
SET status = 'ERROR',
    fail_message = 'The upgrade replaced the unknown compression ' || compression,
    compression = 'ZSTD'
WHERE compression NOT IN ('ZSTD', 'GZIP', 'NONE');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_incremental
SET status = 'ERROR',
    fail_message = 'The upgrade replaced the unknown status ' || status
WHERE status NOT IN ('IN_PROGRESS', 'COMPLETED', 'ERROR', 'CHAIN_BROKEN', 'CANCELED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_incremental
SET status = 'ERROR',
    fail_message = 'The upgrade replaced the unknown compression ' || compression,
    compression = 'ZSTD'
WHERE compression NOT IN ('ZSTD', 'GZIP', 'NONE');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_full
SET encryption = CASE WHEN encryption_salt IS NULL THEN 'NONE' ELSE 'ENCRYPTED' END
WHERE encryption NOT IN ('NONE', 'ENCRYPTED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_incremental
SET encryption = CASE WHEN encryption_salt IS NULL THEN 'NONE' ELSE 'ENCRYPTED' END
WHERE encryption NOT IN ('NONE', 'ENCRYPTED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_wal_segments
SET encryption = CASE WHEN encryption_salt IS NULL THEN 'NONE' ELSE 'ENCRYPTED' END
WHERE encryption NOT IN ('NONE', 'ENCRYPTED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_full
SET fail_message = CASE WHEN COALESCE(fail_message, '') = '' THEN '' ELSE fail_message || '; ' END
        || 'The upgrade cleared the unknown error reason ' || error_reason,
    error_reason = NULL
WHERE error_reason NOT IN ('PG_BASEBACKUP_FAILED', 'STORAGE_UPLOAD_FAILED', 'NETWORK_FAILURE', 'NETWORK_STALL_TIMEOUT', 'APPLICATION_RESTART', 'SYSTEM_IDENTIFIER_MISMATCH', 'TIMELINE_REGRESSION', 'TIMELINE_SWITCH_DETECTED', 'FAILOVER_DURING_BACKUP', 'MANIFEST_CORRUPTED', 'START_LSN_OUTSIDE_TIMELINE_RANGE', 'CANCELED_BY_USER', 'CANCELED_BY_DB_REMOVAL', 'SUMMARIES_EXPIRED', 'SUMMARIZER_OFF', 'PARENT_MANIFEST_MISSING', 'SUMMARIZER_FALLING_BEHIND');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_incremental
SET fail_message = CASE WHEN COALESCE(fail_message, '') = '' THEN '' ELSE fail_message || '; ' END
        || 'The upgrade cleared the unknown error reason ' || error_reason,
    error_reason = NULL
WHERE error_reason NOT IN ('PG_BASEBACKUP_FAILED', 'STORAGE_UPLOAD_FAILED', 'NETWORK_FAILURE', 'NETWORK_STALL_TIMEOUT', 'APPLICATION_RESTART', 'SYSTEM_IDENTIFIER_MISMATCH', 'TIMELINE_REGRESSION', 'TIMELINE_SWITCH_DETECTED', 'FAILOVER_DURING_BACKUP', 'MANIFEST_CORRUPTED', 'START_LSN_OUTSIDE_TIMELINE_RANGE', 'CANCELED_BY_USER', 'CANCELED_BY_DB_REMOVAL', 'SUMMARIES_EXPIRED', 'SUMMARIZER_OFF', 'PARENT_MANIFEST_MISSING', 'SUMMARIZER_FALLING_BEHIND');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical_wal_streamers
SET status = 'FAILED'
WHERE status NOT IN ('RUNNING', 'FAILED');
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM backups_physical_wal_stream_samples
WHERE source_state NOT IN ('REACHABLE', 'UNREACHABLE', 'SLOT_MISSING');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    DROP CONSTRAINT fk_physical_full_backups_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    ADD CONSTRAINT fk_backups_physical_full_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    DROP CONSTRAINT fk_physical_incremental_backups_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    ADD CONSTRAINT fk_backups_physical_incremental_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    DROP CONSTRAINT fk_physical_in_flight_backups_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    ADD CONSTRAINT fk_backups_physical_in_flight_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    DROP CONSTRAINT fk_physical_wal_segments_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    ADD CONSTRAINT fk_backups_physical_wal_segments_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    DROP CONSTRAINT fk_physical_wal_history_files_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    ADD CONSTRAINT fk_backups_physical_wal_history_files_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_streamers
    DROP CONSTRAINT fk_physical_wal_streamers_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_streamers
    ADD CONSTRAINT fk_backups_physical_wal_streamers_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_stream_samples
    DROP CONSTRAINT fk_physical_wal_stream_samples_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_stream_samples
    ADD CONSTRAINT fk_backups_physical_wal_stream_samples_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE backups_physical_wal_stream_samples
    DROP CONSTRAINT fk_backups_physical_wal_stream_samples_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_stream_samples
    ADD CONSTRAINT fk_physical_wal_stream_samples_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_streamers
    DROP CONSTRAINT fk_backups_physical_wal_streamers_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_streamers
    ADD CONSTRAINT fk_physical_wal_streamers_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    DROP CONSTRAINT fk_backups_physical_wal_history_files_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    ADD CONSTRAINT fk_physical_wal_history_files_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    DROP CONSTRAINT fk_backups_physical_wal_segments_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    ADD CONSTRAINT fk_physical_wal_segments_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    DROP CONSTRAINT fk_backups_physical_in_flight_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    ADD CONSTRAINT fk_physical_in_flight_backups_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    DROP CONSTRAINT fk_backups_physical_incremental_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    ADD CONSTRAINT fk_physical_incremental_backups_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    DROP CONSTRAINT fk_backups_physical_full_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    ADD CONSTRAINT fk_physical_full_backups_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_wal_stream_samples_sampled_at
    RENAME TO idx_physical_wal_stream_samples_sampled_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_wal_stream_samples_database_sampled_at
    RENAME TO idx_physical_wal_stream_samples_database_id_sampled_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_stream_samples
    RENAME CONSTRAINT backups_physical_wal_stream_samples_pkey TO physical_wal_stream_samples_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_stream_samples
    RENAME TO physical_wal_stream_samples;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_streamers
    RENAME CONSTRAINT backups_physical_wal_streamers_pkey TO physical_wal_streamers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_streamers
    RENAME TO physical_wal_streamers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    RENAME CONSTRAINT uq_backups_physical_wal_history_files_database_timeline TO uk_physical_wal_history_files_database_id_timeline_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    RENAME CONSTRAINT fk_backups_physical_wal_history_files_storage TO fk_physical_wal_history_files_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    RENAME CONSTRAINT backups_physical_wal_history_files_pkey TO physical_wal_history_files_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_history_files
    RENAME TO physical_wal_history_files;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_wal_segments_database_received_at
    RENAME TO idx_physical_wal_segments_database_id_received_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    RENAME CONSTRAINT uq_backups_physical_wal_segments_database_timeline_start_lsn TO uk_physical_wal_segments_database_id_timeline_id_start_lsn;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    RENAME CONSTRAINT fk_backups_physical_wal_segments_storage TO fk_physical_wal_segments_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    RENAME CONSTRAINT backups_physical_wal_segments_pkey TO physical_wal_segments_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_wal_segments
    RENAME TO physical_wal_segments;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    RENAME CONSTRAINT ck_backups_physical_in_flight_backup_type TO chk_physical_in_flight_backups_backup_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    RENAME CONSTRAINT backups_physical_in_flight_pkey TO physical_in_flight_backups_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_in_flight
    RENAME TO physical_in_flight_backups;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_incremental_parent_incremental_backup
    RENAME TO idx_physical_incremental_backups_parent_incremental_backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_incremental_database_status_created_at
    RENAME TO idx_physical_incremental_backups_database_id_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_incremental_root_full_start_lsn
    RENAME TO idx_physical_incremental_backups_root_full_start_lsn;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT fk_backups_physical_incremental_parent_incremental_backup TO fk_physical_incremental_backups_parent_incremental_backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT fk_backups_physical_incremental_root_full_backup TO fk_physical_incremental_backups_root_full_backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT fk_backups_physical_incremental_storage TO fk_physical_incremental_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME CONSTRAINT backups_physical_incremental_pkey TO physical_incremental_backups_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_incremental
    RENAME TO physical_incremental_backups;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_full_database_timeline_id
    RENAME TO idx_physical_full_backups_database_id_timeline_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_physical_full_database_status_created_at
    RENAME TO idx_physical_full_backups_database_id_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    RENAME CONSTRAINT fk_backups_physical_full_storage TO fk_physical_full_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    RENAME CONSTRAINT backups_physical_full_pkey TO physical_full_backups_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical_full
    RENAME TO physical_full_backups;
-- +goose StatementEnd
