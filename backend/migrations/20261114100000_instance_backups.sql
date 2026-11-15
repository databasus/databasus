-- +goose Up
CREATE TABLE instancebackups_configs (
    id                             BOOLEAN     NOT NULL DEFAULT TRUE,
    schedule_type                  TEXT        NOT NULL DEFAULT 'HOURLY',
    schedule_time_of_day           TEXT,
    schedule_weekday               INTEGER,
    schedule_day_of_month          INTEGER,
    schedule_cron_expression       TEXT,
    retention_days                 INTEGER     NOT NULL DEFAULT 30,
    protection_state               TEXT        NOT NULL DEFAULT 'NOT_CONFIGURED',
    next_selection_generation      BIGINT      NOT NULL DEFAULT 1,
    is_run_requested               BOOLEAN     NOT NULL DEFAULT FALSE,
    setup_deferred_until           TIMESTAMPTZ,
    acknowledged_master_key_sha256 TEXT,
    master_key_acknowledged_at     TIMESTAMPTZ,
    recovery_paused_operation_id   UUID,
    recovery_paused_at             TIMESTAMPTZ,
    archive_namespace              UUID        NOT NULL DEFAULT gen_random_uuid()
);

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT pk_instancebackups_configs
    PRIMARY KEY (id);

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT chk_instancebackups_configs_singleton
    CHECK (id);

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT chk_instancebackups_configs_retention_days
    CHECK (retention_days BETWEEN 1 AND 3650);

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT chk_instancebackups_configs_schedule_weekday
    CHECK (schedule_weekday BETWEEN 0 AND 7);

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT chk_instancebackups_configs_schedule_day_of_month
    CHECK (schedule_day_of_month BETWEEN 1 AND 31);

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT chk_instancebackups_configs_recovery_pause
    CHECK ((recovery_paused_operation_id IS NULL) = (recovery_paused_at IS NULL));

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT chk_instancebackups_configs_next_selection_generation
    CHECK (next_selection_generation > 0);

ALTER TABLE instancebackups_configs
    ADD CONSTRAINT chk_instancebackups_configs_master_key_acknowledgment
    CHECK (num_nonnulls(acknowledged_master_key_sha256, master_key_acknowledged_at) IN (0, 2));

INSERT INTO instancebackups_configs DEFAULT VALUES;

CREATE TABLE instancebackups_destinations (
    storage_id                  UUID        NOT NULL,
    selection_generation        BIGINT      NOT NULL,
    selected_at                 TIMESTAMPTZ NOT NULL,
    outstanding_due_at          TIMESTAMPTZ,
    last_successful_snapshot_at TIMESTAMPTZ,
    last_error                  TEXT
);

ALTER TABLE instancebackups_destinations
    ADD CONSTRAINT pk_instancebackups_destinations
    PRIMARY KEY (storage_id);

ALTER TABLE instancebackups_destinations
    ADD CONSTRAINT fk_instancebackups_destinations_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;

ALTER TABLE instancebackups_destinations
    ADD CONSTRAINT chk_instancebackups_destinations_selection_generation
    CHECK (selection_generation > 0);

CREATE TABLE instancebackups_backups (
    id               UUID        NOT NULL DEFAULT gen_random_uuid(),
    status           TEXT        NOT NULL,
    dump_status      TEXT        NOT NULL,
    trigger          TEXT        NOT NULL,
    source_release   TEXT        NOT NULL,
    started_at       TIMESTAMPTZ NOT NULL,
    snapshot_at      TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    size_bytes       BIGINT,
    failure_reason   TEXT,
    spool_path       TEXT,
    spool_expires_at TIMESTAMPTZ
);

ALTER TABLE instancebackups_backups
    ADD CONSTRAINT pk_instancebackups_backups
    PRIMARY KEY (id);

ALTER TABLE instancebackups_backups
    ADD CONSTRAINT chk_instancebackups_backups_size_bytes
    CHECK (size_bytes >= 0);

ALTER TABLE instancebackups_backups
    ADD CONSTRAINT chk_instancebackups_backups_spool
    CHECK (num_nonnulls(spool_path, spool_expires_at) IN (0, 2));

CREATE UNIQUE INDEX uq_instancebackups_backups_one_running_dump
    ON instancebackups_backups ((TRUE))
    WHERE dump_status = 'RUNNING';

CREATE INDEX idx_instancebackups_backups_history
    ON instancebackups_backups (COALESCE(snapshot_at, started_at) DESC, id DESC);

CREATE TABLE instancebackups_copies (
    id                   UUID        NOT NULL DEFAULT gen_random_uuid(),
    backup_id            UUID        NOT NULL,
    storage_id           UUID,
    selection_generation BIGINT      NOT NULL,
    storage_name         TEXT        NOT NULL,
    workspace_name       TEXT        NOT NULL,
    status               TEXT        NOT NULL,
    attempt_count        INTEGER     NOT NULL DEFAULT 0,
    next_attempt_at      TIMESTAMPTZ,
    file_name            TEXT,
    stored_at            TIMESTAMPTZ,
    last_error           TEXT
);

ALTER TABLE instancebackups_copies
    ADD CONSTRAINT pk_instancebackups_copies
    PRIMARY KEY (id);

ALTER TABLE instancebackups_copies
    ADD CONSTRAINT fk_instancebackups_copies_backup_id
    FOREIGN KEY (backup_id)
    REFERENCES instancebackups_backups (id)
    ON DELETE CASCADE;

ALTER TABLE instancebackups_copies
    ADD CONSTRAINT fk_instancebackups_copies_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;

ALTER TABLE instancebackups_copies
    ADD CONSTRAINT uq_instancebackups_copies_backup_storage
    UNIQUE (backup_id, storage_id);

ALTER TABLE instancebackups_copies
    ADD CONSTRAINT chk_instancebackups_copies_attempt_count
    CHECK (attempt_count >= 0);

ALTER TABLE instancebackups_copies
    ADD CONSTRAINT chk_instancebackups_copies_selection_generation
    CHECK (selection_generation > 0);

CREATE INDEX idx_instancebackups_copies_storage_id
    ON instancebackups_copies (storage_id);

CREATE TABLE instancebackups_incidents (
    id                   UUID        NOT NULL DEFAULT gen_random_uuid(),
    kind                 TEXT        NOT NULL,
    started_at           TIMESTAMPTZ NOT NULL,
    resolved_at          TIMESTAMPTZ,
    last_mailed_at       TIMESTAMPTZ,
    mail_attempt_count   INTEGER     NOT NULL DEFAULT 0,
    next_mail_attempt_at TIMESTAMPTZ,
    last_mail_error      TEXT
);

ALTER TABLE instancebackups_incidents
    ADD CONSTRAINT pk_instancebackups_incidents
    PRIMARY KEY (id);

ALTER TABLE instancebackups_incidents
    ADD CONSTRAINT chk_instancebackups_incidents_mail_attempt_count
    CHECK (mail_attempt_count >= 0);

CREATE UNIQUE INDEX uq_instancebackups_incidents_one_open
    ON instancebackups_incidents ((TRUE))
    WHERE resolved_at IS NULL;

CREATE TABLE instancebackups_tokens (
    token_sha256 TEXT        NOT NULL,
    purpose      TEXT        NOT NULL,
    user_id      UUID        NOT NULL,
    copy_id      UUID,
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ
);

ALTER TABLE instancebackups_tokens
    ADD CONSTRAINT pk_instancebackups_tokens
    PRIMARY KEY (token_sha256);

ALTER TABLE instancebackups_tokens
    ADD CONSTRAINT fk_instancebackups_tokens_copy_id
    FOREIGN KEY (copy_id)
    REFERENCES instancebackups_copies (id)
    ON DELETE CASCADE;

CREATE INDEX idx_instancebackups_tokens_copy_id
    ON instancebackups_tokens (copy_id);

-- +goose Down
DROP TABLE instancebackups_tokens;
DROP TABLE instancebackups_incidents;
DROP TABLE instancebackups_copies;
DROP TABLE instancebackups_backups;
DROP TABLE instancebackups_destinations;
DROP TABLE instancebackups_configs;
