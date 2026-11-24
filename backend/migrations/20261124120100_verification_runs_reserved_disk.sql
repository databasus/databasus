-- +goose Up
ALTER TABLE verification_runs
    ADD COLUMN reserved_disk_mb BIGINT NOT NULL DEFAULT 0;

ALTER TABLE verification_runs
    ADD CONSTRAINT chk_verification_runs_reserved_disk_mb
    CHECK (reserved_disk_mb >= 0);

-- A run that crosses the upgrade keeps holding the disk its agent was planned with: the archive,
-- the estimated database size and a gap of half that size clamped to 1536-5120 MB, or 5120 MB
-- when the size is unknown.
UPDATE verification_runs
SET reserved_disk_mb = CEIL(
        GREATEST(backups_logical.backup_size_mb, 0)
        + GREATEST(backups_logical.backup_raw_db_size_mb, 0)
        + CASE
            WHEN backups_logical.backup_raw_db_size_mb > 0
                THEN LEAST(5120, GREATEST(1536, backups_logical.backup_raw_db_size_mb * 0.5))
            ELSE 5120
          END
    )
FROM backups_logical
WHERE backups_logical.id = verification_runs.backup_id
  AND verification_runs.status = 'RUNNING';

-- +goose Down
ALTER TABLE verification_runs
    DROP COLUMN reserved_disk_mb;
