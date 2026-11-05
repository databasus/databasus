-- +goose Up
-- +goose StatementBegin
ALTER TABLE local_storages
    RENAME TO storages_local;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    RENAME CONSTRAINT local_storages_pkey TO storages_local_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE s3_storages
    RENAME TO storages_s3;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    RENAME CONSTRAINT s3_storages_pkey TO storages_s3_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE google_drive_storages
    RENAME TO storages_google_drive;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    RENAME CONSTRAINT google_drive_storages_pkey TO storages_google_drive_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE nas_storages
    RENAME TO storages_nas;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    RENAME CONSTRAINT nas_storages_pkey TO storages_nas_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE azure_blob_storages
    RENAME TO storages_azure_blob;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    RENAME CONSTRAINT azure_blob_storages_pkey TO storages_azure_blob_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE ftp_storages
    RENAME TO storages_ftp;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    RENAME CONSTRAINT ftp_storages_pkey TO storages_ftp_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE sftp_storages
    RENAME TO storages_sftp;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    RENAME CONSTRAINT sftp_storages_pkey TO storages_sftp_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE rclone_storages
    RENAME TO storages_rclone;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    RENAME CONSTRAINT rclone_storages_pkey TO storages_rclone_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storage_pending_deletions
    RENAME TO storages_pending_deletions;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT storage_pending_deletions_pkey TO storages_pending_deletions_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT uq_storage_pending_deletions_storage_file TO uq_storages_pending_deletions_storage_file;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT ck_storage_pending_deletions_attempt_count_not_negative TO ck_storages_pending_deletions_attempt_count_not_negative;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT ck_storage_pending_deletions_file_name_not_blank TO ck_storages_pending_deletions_file_name_not_blank;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT ck_storage_pending_deletions_generation_not_negative TO ck_storages_pending_deletions_generation_not_negative;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT fk_storage_pending_deletions_storage_id TO fk_storages_pending_deletions_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_storage_pending_deletions_not_before
    RENAME TO idx_storages_pending_deletions_not_before;
-- +goose StatementEnd

-- A provider row whose storage has another type is never read.
-- +goose StatementBegin
DELETE FROM storages_local p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'LOCAL';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM storages_s3 p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'S3';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM storages_google_drive p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'GOOGLE_DRIVE';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM storages_nas p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'NAS';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM storages_azure_blob p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'AZURE_BLOB';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM storages_ftp p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'FTP';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM storages_sftp p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'SFTP';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM storages_rclone p
USING storages s
WHERE s.id = p.storage_id
    AND s.type <> 'RCLONE';
-- +goose StatementEnd

-- A storage of an unknown type or without its provider row cannot work. It goes unless a backup
-- or a backup config still references it; such a storage keeps its parent row.
-- +goose StatementBegin
DELETE FROM storages s
WHERE NOT (
        (s.type = 'LOCAL' AND EXISTS (SELECT 1 FROM storages_local p WHERE p.storage_id = s.id))
        OR (s.type = 'S3' AND EXISTS (SELECT 1 FROM storages_s3 p WHERE p.storage_id = s.id))
        OR (s.type = 'GOOGLE_DRIVE' AND EXISTS (SELECT 1 FROM storages_google_drive p WHERE p.storage_id = s.id))
        OR (s.type = 'NAS' AND EXISTS (SELECT 1 FROM storages_nas p WHERE p.storage_id = s.id))
        OR (s.type = 'AZURE_BLOB' AND EXISTS (SELECT 1 FROM storages_azure_blob p WHERE p.storage_id = s.id))
        OR (s.type = 'FTP' AND EXISTS (SELECT 1 FROM storages_ftp p WHERE p.storage_id = s.id))
        OR (s.type = 'SFTP' AND EXISTS (SELECT 1 FROM storages_sftp p WHERE p.storage_id = s.id))
        OR (s.type = 'RCLONE' AND EXISTS (SELECT 1 FROM storages_rclone p WHERE p.storage_id = s.id))
    )
    AND NOT (
        EXISTS (SELECT 1 FROM logical_backup_configs r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM logical_backups r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_backup_configs r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_full_backups r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_incremental_backups r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_wal_segments r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_wal_history_files r WHERE r.storage_id = s.id)
    );
-- +goose StatementEnd

-- A storage without a workspace joins the workspace of the earliest database whose backups use it.
-- +goose StatementBegin
UPDATE storages s
SET workspace_id = (
    SELECT d.workspace_id
    FROM databases d
    WHERE d.workspace_id IS NOT NULL
        AND d.id IN (
        SELECT r.database_id FROM logical_backup_configs r WHERE r.storage_id = s.id
        UNION ALL
        SELECT r.database_id FROM logical_backups r WHERE r.storage_id = s.id
        UNION ALL
        SELECT r.database_id FROM physical_backup_configs r WHERE r.storage_id = s.id
        UNION ALL
        SELECT r.database_id FROM physical_full_backups r WHERE r.storage_id = s.id
        UNION ALL
        SELECT r.database_id FROM physical_incremental_backups r WHERE r.storage_id = s.id
        UNION ALL
        SELECT r.database_id FROM physical_wal_segments r WHERE r.storage_id = s.id
        UNION ALL
        SELECT r.database_id FROM physical_wal_history_files r WHERE r.storage_id = s.id
        )
    ORDER BY d.id
    LIMIT 1
)
WHERE s.workspace_id IS NULL;
-- +goose StatementEnd

-- Without such a database it joins the oldest workspace, as storages did when the column appeared.
-- +goose StatementBegin
UPDATE storages
SET workspace_id = (SELECT id FROM workspaces ORDER BY created_at, id LIMIT 1)
WHERE workspace_id IS NULL;
-- +goose StatementEnd

-- An instance without any workspace has nowhere to put the storage.
-- +goose StatementBegin
DELETE FROM storages s
WHERE s.workspace_id IS NULL
    AND NOT (
        EXISTS (SELECT 1 FROM logical_backup_configs r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM logical_backups r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_backup_configs r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_full_backups r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_incremental_backups r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_wal_segments r WHERE r.storage_id = s.id)
        OR EXISTS (SELECT 1 FROM physical_wal_history_files r WHERE r.storage_id = s.id)
    );
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    ALTER COLUMN workspace_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    ADD CONSTRAINT uq_storages_id_type
    UNIQUE (id, type);
-- +goose StatementEnd

-- The list index starts with the workspace, so it serves every query this one served.
-- +goose StatementBegin
DROP INDEX idx_storages_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_storages_workspace_id_name_id
    ON storages (workspace_id, name, id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    DROP COLUMN last_save_error;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    DROP CONSTRAINT fk_local_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    ADD COLUMN type TEXT NOT NULL DEFAULT 'LOCAL';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    ADD CONSTRAINT fk_storages_local_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    DROP CONSTRAINT fk_s3_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    ADD COLUMN type TEXT NOT NULL DEFAULT 'S3';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    ADD CONSTRAINT fk_storages_s3_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    DROP CONSTRAINT fk_google_drive_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    ADD COLUMN type TEXT NOT NULL DEFAULT 'GOOGLE_DRIVE';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    ADD CONSTRAINT fk_storages_google_drive_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    DROP CONSTRAINT fk_nas_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    ADD COLUMN type TEXT NOT NULL DEFAULT 'NAS';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    ADD CONSTRAINT fk_storages_nas_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    DROP CONSTRAINT fk_azure_blob_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    ADD COLUMN type TEXT NOT NULL DEFAULT 'AZURE_BLOB';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    ADD CONSTRAINT fk_storages_azure_blob_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    DROP CONSTRAINT fk_ftp_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    ADD COLUMN type TEXT NOT NULL DEFAULT 'FTP';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    ADD CONSTRAINT fk_storages_ftp_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    DROP CONSTRAINT fk_sftp_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    ADD COLUMN type TEXT NOT NULL DEFAULT 'SFTP';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    ADD CONSTRAINT fk_storages_sftp_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    DROP CONSTRAINT fk_rclone_storages_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    ADD COLUMN type TEXT NOT NULL DEFAULT 'RCLONE';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    ADD CONSTRAINT fk_storages_rclone_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE storages_rclone
    DROP CONSTRAINT fk_storages_rclone_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    ADD CONSTRAINT fk_rclone_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    DROP CONSTRAINT fk_storages_sftp_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    ADD CONSTRAINT fk_sftp_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    DROP CONSTRAINT fk_storages_ftp_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    ADD CONSTRAINT fk_ftp_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    DROP CONSTRAINT fk_storages_azure_blob_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    ADD CONSTRAINT fk_azure_blob_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    DROP CONSTRAINT fk_storages_nas_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    ADD CONSTRAINT fk_nas_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    DROP CONSTRAINT fk_storages_google_drive_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    ADD CONSTRAINT fk_google_drive_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    DROP CONSTRAINT fk_storages_s3_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    ADD CONSTRAINT fk_s3_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    DROP CONSTRAINT fk_storages_local_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    ADD CONSTRAINT fk_local_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    ADD COLUMN last_save_error TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_storages_workspace_id_name_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_storages_workspace_id
    ON storages (workspace_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    DROP CONSTRAINT uq_storages_id_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    ALTER COLUMN workspace_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_storages_pending_deletions_not_before
    RENAME TO idx_storage_pending_deletions_not_before;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT storages_pending_deletions_pkey TO storage_pending_deletions_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT uq_storages_pending_deletions_storage_file TO uq_storage_pending_deletions_storage_file;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT ck_storages_pending_deletions_attempt_count_not_negative TO ck_storage_pending_deletions_attempt_count_not_negative;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT ck_storages_pending_deletions_file_name_not_blank TO ck_storage_pending_deletions_file_name_not_blank;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT ck_storages_pending_deletions_generation_not_negative TO ck_storage_pending_deletions_generation_not_negative;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME CONSTRAINT fk_storages_pending_deletions_storage_id TO fk_storage_pending_deletions_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_pending_deletions
    RENAME TO storage_pending_deletions;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    RENAME CONSTRAINT storages_rclone_pkey TO rclone_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_rclone
    RENAME TO rclone_storages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    RENAME CONSTRAINT storages_sftp_pkey TO sftp_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_sftp
    RENAME TO sftp_storages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    RENAME CONSTRAINT storages_ftp_pkey TO ftp_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_ftp
    RENAME TO ftp_storages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    RENAME CONSTRAINT storages_azure_blob_pkey TO azure_blob_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_azure_blob
    RENAME TO azure_blob_storages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    RENAME CONSTRAINT storages_nas_pkey TO nas_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_nas
    RENAME TO nas_storages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    RENAME CONSTRAINT storages_google_drive_pkey TO google_drive_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_google_drive
    RENAME TO google_drive_storages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    RENAME CONSTRAINT storages_s3_pkey TO s3_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_s3
    RENAME TO s3_storages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    RENAME CONSTRAINT storages_local_pkey TO local_storages_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages_local
    RENAME TO local_storages;
-- +goose StatementEnd
