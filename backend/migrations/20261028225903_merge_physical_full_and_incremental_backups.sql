-- +goose Up
-- +goose StatementBegin
CREATE TABLE backups_physical (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id               UUID NOT NULL,
    storage_id                UUID NOT NULL,
    timeline_id               INT  NOT NULL,
    status                    TEXT NOT NULL,
    error_reason              TEXT,
    file_name                 TEXT,
    start_lsn                 PG_LSN,
    stop_lsn                  PG_LSN,
    backup_size_mb            DOUBLE PRECISION,
    raw_size_mb               DOUBLE PRECISION,
    backup_duration_ms        BIGINT,
    compression               TEXT NOT NULL DEFAULT 'ZSTD',
    encryption                TEXT NOT NULL DEFAULT 'NONE',
    encryption_salt           TEXT,
    encryption_iv             TEXT,
    manifest_file_name        TEXT,
    manifest_encryption_salt  TEXT,
    manifest_encryption_iv    TEXT,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fail_message              TEXT,
    kind                      TEXT NOT NULL,
    parent_backup_id          UUID,
    root_full_backup_id        UUID,
    completed_at              TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical
    ADD CONSTRAINT fk_backups_physical_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical
    ADD CONSTRAINT fk_backups_physical_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical
    ADD CONSTRAINT fk_backups_physical_parent_backup
    FOREIGN KEY (parent_backup_id)
    REFERENCES backups_physical (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical
    ADD CONSTRAINT fk_backups_physical_root_full_backup
    FOREIGN KEY (root_full_backup_id)
    REFERENCES backups_physical (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO backups_physical (id, database_id, storage_id, timeline_id, status, error_reason, file_name, start_lsn, stop_lsn, backup_size_mb, raw_size_mb, backup_duration_ms, compression, encryption, encryption_salt, encryption_iv, manifest_file_name, manifest_encryption_salt, manifest_encryption_iv, created_at, fail_message, completed_at, kind)
SELECT id, database_id, storage_id, timeline_id, status, error_reason, file_name, start_lsn, stop_lsn, backup_size_mb, raw_size_mb, backup_duration_ms, compression, encryption, encryption_salt, encryption_iv, manifest_file_name, manifest_encryption_salt, manifest_encryption_iv, created_at, fail_message, completed_at, 'FULL' FROM backups_physical_full;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO backups_physical (id, database_id, storage_id, timeline_id, status, error_reason, file_name, start_lsn, stop_lsn, backup_size_mb, raw_size_mb, backup_duration_ms, compression, encryption, encryption_salt, encryption_iv, manifest_file_name, manifest_encryption_salt, manifest_encryption_iv, created_at, fail_message, completed_at, kind, parent_backup_id, root_full_backup_id)
SELECT id, database_id, storage_id, timeline_id, status, error_reason, file_name, start_lsn, stop_lsn, backup_size_mb, raw_size_mb, backup_duration_ms, compression, encryption, encryption_salt, encryption_iv, manifest_file_name, manifest_encryption_salt, manifest_encryption_iv, created_at, fail_message, completed_at, 'INCREMENTAL', COALESCE(parent_incremental_backup_id, root_full_backup_id), root_full_backup_id
FROM backups_physical_incremental;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical
SET completed_at = created_at
WHERE status = 'COMPLETED' AND completed_at IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_physical
SET status = 'ERROR',
    fail_message = 'The upgrade marked an incomplete backup failed: missing ' ||
        CONCAT_WS(', ',
            CASE WHEN file_name IS NULL THEN 'file_name' END,
            CASE WHEN start_lsn IS NULL THEN 'start_lsn' END,
            CASE WHEN stop_lsn IS NULL THEN 'stop_lsn' END)
WHERE status = 'COMPLETED' AND (file_name IS NULL OR start_lsn IS NULL OR stop_lsn IS NULL);
-- +goose StatementEnd

-- +goose StatementBegin
WITH running AS (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY database_id ORDER BY created_at DESC, id DESC) AS position
    FROM backups_physical
    WHERE status = 'IN_PROGRESS'
)
UPDATE backups_physical
SET status = 'ERROR',
    error_reason = 'APPLICATION_RESTART',
    completed_at = created_at,
    fail_message = 'Backup was interrupted by an application restart and marked failed. Trigger a new backup.'
WHERE id IN (SELECT id FROM running WHERE position > 1);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical
    ADD CONSTRAINT ck_backups_physical_chain
    CHECK ((kind <> 'FULL' OR (parent_backup_id IS NULL AND root_full_backup_id IS NULL))
        AND (kind <> 'INCREMENTAL' OR (parent_backup_id IS NOT NULL AND root_full_backup_id IS NOT NULL)));
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_physical
    ADD CONSTRAINT ck_backups_physical_completed
    CHECK (status <> 'COMPLETED' OR (file_name IS NOT NULL AND start_lsn IS NOT NULL AND stop_lsn IS NOT NULL AND completed_at IS NOT NULL));
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_physical_database_id_status_created_at
    ON backups_physical (database_id, status, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_physical_database_id_timeline_id
    ON backups_physical (database_id, timeline_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_physical_root_full_start_lsn
    ON backups_physical (root_full_backup_id, start_lsn);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_physical_parent_backup_id
    ON backups_physical (parent_backup_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_physical_database_id_kind_created_at
    ON backups_physical (database_id, kind, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX uq_backups_physical_database_in_progress
    ON backups_physical (database_id)
    WHERE status = 'IN_PROGRESS';
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE backups_physical_incremental;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE backups_physical_full;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE backups_physical_in_flight;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'The physical backup merge is one-way. To roll back, restore a dump taken before the upgrade.';
END $$;
-- +goose StatementEnd
