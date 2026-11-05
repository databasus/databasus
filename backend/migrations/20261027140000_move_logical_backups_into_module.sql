-- +goose Up
-- +goose StatementBegin
ALTER TABLE logical_backups
    RENAME TO backups_logical;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    RENAME CONSTRAINT backups_pkey TO backups_logical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE restores
    RENAME TO backups_restores;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    RENAME CONSTRAINT restores_pkey TO backups_restores_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE download_tokens
    RENAME TO backups_download_tokens;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME CONSTRAINT download_tokens_pkey TO backups_download_tokens_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME CONSTRAINT download_tokens_token_key TO uq_backups_download_tokens_token;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME COLUMN used TO is_used;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    DROP COLUMN is_skip_retry;
-- +goose StatementEnd

-- Unknown lifecycle states cannot be resumed by the application. Keep their stored value in the failure message.
-- +goose StatementBegin
UPDATE backups_logical
SET status       = 'FAILED',
    fail_message = 'The upgrade replaced the unknown status ' || status
WHERE status NOT IN ('IN_PROGRESS', 'COMPLETED', 'FAILED', 'CANCELED');
-- +goose StatementEnd

-- An encryption salt is written only for an encrypted file, so a row with a salt keeps its file readable
-- as ENCRYPTED and a row without one was never encrypted.
-- +goose StatementBegin
UPDATE backups_logical
SET encryption = CASE WHEN encryption_salt IS NULL THEN 'NONE' ELSE 'ENCRYPTED' END
WHERE encryption NOT IN ('NONE', 'ENCRYPTED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_logical
SET restore_verification_status = 'NOT_VERIFIED'
WHERE restore_verification_status NOT IN ('NOT_VERIFIED', 'VERIFIED_SUCCESSFUL', 'VERIFICATION_FAILED');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_restores
SET status       = 'FAILED',
    fail_message = 'The upgrade replaced the unknown status ' || status
WHERE status NOT IN ('IN_PROGRESS', 'COMPLETED', 'FAILED', 'CANCELED');
-- +goose StatementEnd

-- Only one backup of a database may run, so all but the newest in-progress backup are failed, as the
-- next start would fail them.
-- +goose StatementBegin
UPDATE backups_logical
SET status       = 'FAILED',
    fail_message = 'Backup failed due to application restart'
WHERE status = 'IN_PROGRESS'
  AND id NOT IN (
      SELECT DISTINCT ON (database_id) id
      FROM backups_logical
      WHERE status = 'IN_PROGRESS'
      ORDER BY database_id, created_at DESC, id DESC
  );
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    ADD COLUMN database_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_restores
SET database_id = backups_logical.database_id
FROM backups_logical
WHERE backups_logical.id = backups_restores.backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    ALTER COLUMN database_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE backups_restores
SET status       = 'FAILED',
    fail_message = 'Restore failed due to application restart'
WHERE status = 'IN_PROGRESS'
  AND id NOT IN (
      SELECT DISTINCT ON (database_id) id
      FROM backups_restores
      WHERE status = 'IN_PROGRESS'
      ORDER BY database_id, created_at DESC, id DESC
  );
-- +goose StatementEnd

-- A restore keeps its backup's database, so the index below allows one running restore per database;
-- the composite reference keeps the copy equal to the backup's database.
-- +goose StatementBegin
ALTER TABLE backups_logical
    ADD CONSTRAINT uq_backups_logical_id_database
    UNIQUE (id, database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    DROP CONSTRAINT fk_restores_backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    ADD CONSTRAINT fk_backups_restores_backup
    FOREIGN KEY (backup_id, database_id)
    REFERENCES backups_logical (id, database_id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX uq_backups_logical_in_progress_database
    ON backups_logical (database_id)
    WHERE status = 'IN_PROGRESS';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX uq_backups_restores_in_progress_database
    ON backups_restores (database_id)
    WHERE status = 'IN_PROGRESS';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    DROP CONSTRAINT fk_backups_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    ADD CONSTRAINT fk_backups_logical_database
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    RENAME CONSTRAINT fk_backups_storage_id TO fk_backups_logical_storage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME CONSTRAINT fk_download_tokens_backup_id TO fk_backups_download_tokens_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    DROP CONSTRAINT fk_download_tokens_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    ADD CONSTRAINT fk_backups_download_tokens_user
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- The lists page newest first by (created_at, id).
-- +goose StatementBegin
DROP INDEX idx_backups_database_id_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_logical_database_created_at
    ON backups_logical (database_id, created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_status_created_at
    RENAME TO idx_backups_logical_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_logical_storage_id
    ON backups_logical (storage_id);
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_restores_backup_id_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_restores_backup_created_at
    ON backups_restores (backup_id, created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_restores_status_created_at
    RENAME TO idx_backups_restores_status_created_at;
-- +goose StatementEnd

-- The unique constraint on the token already indexes it.
-- +goose StatementBegin
DROP INDEX idx_download_tokens_token;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_download_tokens_backup_id
    RENAME TO idx_backups_download_tokens_backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_download_tokens_expires_at
    RENAME TO idx_backups_download_tokens_expires_at;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_download_tokens_user_id
    ON backups_download_tokens (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_backups_download_tokens_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_download_tokens_expires_at
    RENAME TO idx_download_tokens_expires_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_download_tokens_backup_id
    RENAME TO idx_download_tokens_backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_download_tokens_token
    ON backups_download_tokens (token);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_restores_status_created_at
    RENAME TO idx_restores_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_backups_restores_backup_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_restores_backup_id_created_at
    ON backups_restores (backup_id, created_at);
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_backups_logical_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_backups_logical_status_created_at
    RENAME TO idx_backups_status_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_backups_logical_database_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_backups_database_id_created_at
    ON backups_logical (database_id, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    DROP CONSTRAINT fk_backups_download_tokens_user;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    ADD CONSTRAINT fk_download_tokens_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME CONSTRAINT fk_backups_download_tokens_backup TO fk_download_tokens_backup_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    RENAME CONSTRAINT fk_backups_logical_storage TO fk_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    DROP CONSTRAINT fk_backups_logical_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    ADD CONSTRAINT fk_backups_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX uq_backups_restores_in_progress_database;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX uq_backups_logical_in_progress_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    DROP CONSTRAINT fk_backups_restores_backup;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    ADD CONSTRAINT fk_restores_backup_id
    FOREIGN KEY (backup_id)
    REFERENCES backups_logical (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    DROP CONSTRAINT uq_backups_logical_id_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    DROP COLUMN database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    ADD COLUMN is_skip_retry BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME COLUMN is_used TO used;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME CONSTRAINT uq_backups_download_tokens_token TO download_tokens_token_key;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME CONSTRAINT backups_download_tokens_pkey TO download_tokens_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_download_tokens
    RENAME TO download_tokens;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    RENAME CONSTRAINT backups_restores_pkey TO restores_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_restores
    RENAME TO restores;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    RENAME CONSTRAINT backups_logical_pkey TO backups_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE backups_logical
    RENAME TO logical_backups;
-- +goose StatementEnd
