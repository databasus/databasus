-- +goose Up
DROP TABLE backups_storage_switches;

DROP TABLE storages_deletion_batch_items;

DROP TABLE storages_deletion_batches;

TRUNCATE TABLE storages_pending_deletions;

ALTER TABLE storages_pending_deletions
    DROP COLUMN is_ordinary;

ALTER TABLE storages_pending_deletions
    RENAME TO storages_pending_file_deletions;

ALTER TABLE storages_pending_file_deletions
    RENAME CONSTRAINT storages_pending_deletions_pkey TO storages_pending_file_deletions_pkey;

ALTER TABLE storages_pending_file_deletions
    RENAME CONSTRAINT uq_storages_pending_deletions_storage_file TO uq_storages_pending_file_deletions_storage_file;

ALTER TABLE storages_pending_file_deletions
    RENAME CONSTRAINT ck_storages_pending_deletions_attempt_count_not_negative TO ck_storages_pending_file_deletions_attempt_count_not_negative;

ALTER TABLE storages_pending_file_deletions
    RENAME CONSTRAINT ck_storages_pending_deletions_file_name_not_blank TO ck_storages_pending_file_deletions_file_name_not_blank;

ALTER TABLE storages_pending_file_deletions
    RENAME CONSTRAINT ck_storages_pending_deletions_generation_not_negative TO ck_storages_pending_file_deletions_generation_not_negative;

ALTER TABLE storages_pending_file_deletions
    RENAME CONSTRAINT fk_storages_pending_deletions_storage_id TO fk_storages_pending_file_deletions_storage_id;

ALTER INDEX idx_storages_pending_deletions_not_before
    RENAME TO idx_storages_pending_file_deletions_not_before;

ALTER TABLE storages_pending_file_deletions
    ADD COLUMN cleanup_database_id UUID;

ALTER TABLE storages_pending_file_deletions
    ADD COLUMN cleanup_workspace_id UUID;

ALTER TABLE storages_pending_file_deletions
    ADD CONSTRAINT ck_storages_pending_file_deletions_cleanup_scope
    CHECK ((cleanup_database_id IS NULL) = (cleanup_workspace_id IS NULL));

CREATE INDEX idx_storages_pending_file_deletions_cleanup_scope
    ON storages_pending_file_deletions (cleanup_workspace_id, cleanup_database_id)
    WHERE cleanup_database_id IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'restore a pre-upgrade database backup with its matching binary to roll back';
END;
$$;
-- +goose StatementEnd
