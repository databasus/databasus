-- +goose Up
ALTER TABLE backups_physical_wal_streamers
    ADD COLUMN claim_id UUID NOT NULL DEFAULT gen_random_uuid();

ALTER TABLE backups_physical
    ADD COLUMN storage_generation BIGINT NOT NULL DEFAULT 0;

ALTER TABLE backups_configs
    ADD COLUMN storage_generation BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN last_selected_storage_id UUID;

UPDATE backups_configs SET last_selected_storage_id = storage_id;

ALTER TABLE backups_configs
    ADD CONSTRAINT ck_backups_configs_storage_generation
    CHECK (storage_generation >= 0);

ALTER TABLE storages_pending_deletions
    ADD COLUMN is_ordinary BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE storages_deletion_batches (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status           TEXT NOT NULL DEFAULT 'CLEANUP_PENDING',
    created_at       TIMESTAMPTZ NOT NULL,
    cleanup_deadline TIMESTAMPTZ NOT NULL,
    completed_at     TIMESTAMPTZ,
    total            BIGINT NOT NULL DEFAULT 0,
    pending          BIGINT NOT NULL DEFAULT 0,
    deleted          BIGINT NOT NULL DEFAULT 0,
    remaining        BIGINT NOT NULL DEFAULT 0
);

ALTER TABLE storages_deletion_batches
    ADD CONSTRAINT ck_storages_deletion_batches_counts
    CHECK (total = pending + deleted + remaining AND pending >= 0 AND deleted >= 0 AND remaining >= 0);

ALTER TABLE storages_deletion_batches
    ADD CONSTRAINT ck_storages_deletion_batches_completion
    CHECK ((completed_at IS NULL) = (pending > 0));

ALTER TABLE storages_deletion_batches
    ADD CONSTRAINT ck_storages_deletion_batches_deadline
    CHECK (cleanup_deadline > created_at);

CREATE INDEX idx_storages_deletion_batches_pending_deadline
    ON storages_deletion_batches (cleanup_deadline)
    WHERE completed_at IS NULL;

CREATE TABLE storages_deletion_batch_items (
    batch_id           UUID NOT NULL,
    storage_id         UUID NOT NULL,
    file_name          TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'PENDING',
    failed_attempts    INTEGER NOT NULL DEFAULT 0,
    attempt_generation INTEGER,
    attempt_expires_at TIMESTAMPTZ,
    PRIMARY KEY (batch_id, storage_id, file_name)
);

ALTER TABLE storages_deletion_batch_items
    ADD CONSTRAINT fk_storages_deletion_batch_items_batch
    FOREIGN KEY (batch_id)
    REFERENCES storages_deletion_batches (id)
    ON DELETE CASCADE;

ALTER TABLE storages_deletion_batch_items
    ADD CONSTRAINT ck_storages_deletion_batch_items_file_name
    CHECK (btrim(file_name) <> '');

ALTER TABLE storages_deletion_batch_items
    ADD CONSTRAINT ck_storages_deletion_batch_items_attempts
    CHECK (failed_attempts >= 0);

ALTER TABLE storages_deletion_batch_items
    ADD CONSTRAINT ck_storages_deletion_batch_items_lease
    CHECK ((attempt_generation IS NULL) = (attempt_expires_at IS NULL));

CREATE INDEX idx_storages_deletion_batch_items_pending_file
    ON storages_deletion_batch_items (storage_id, file_name)
    WHERE status = 'PENDING';

CREATE TABLE backups_storage_switches (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id         UUID NOT NULL,
    workspace_id        UUID NOT NULL,
    previous_storage_id UUID,
    target_storage_id   UUID NOT NULL,
    deletion_batch_id   UUID NOT NULL UNIQUE,
    created_at          TIMESTAMPTZ NOT NULL
);

ALTER TABLE backups_storage_switches
    ADD CONSTRAINT fk_backups_storage_switches_deletion_batch
    FOREIGN KEY (deletion_batch_id)
    REFERENCES storages_deletion_batches (id)
    ON DELETE RESTRICT;

CREATE INDEX idx_backups_storage_switches_workspace_history
    ON backups_storage_switches (workspace_id, created_at DESC, id DESC);

CREATE INDEX idx_backups_storage_switches_database_history
    ON backups_storage_switches (workspace_id, database_id, created_at DESC, id DESC);

-- +goose Down
ALTER TABLE backups_physical_wal_streamers DROP COLUMN claim_id;
ALTER TABLE backups_physical DROP COLUMN storage_generation;
DROP TABLE backups_storage_switches;
DROP TABLE storages_deletion_batch_items;
DROP TABLE storages_deletion_batches;
ALTER TABLE storages_pending_deletions DROP COLUMN is_ordinary;
ALTER TABLE backups_configs DROP COLUMN storage_generation;
ALTER TABLE backups_configs DROP COLUMN last_selected_storage_id;
