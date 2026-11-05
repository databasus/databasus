-- +goose Up
-- +goose StatementBegin
ALTER TABLE audit_logs
    RENAME TO lib_audit_events;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    RENAME COLUMN user_id TO author_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    DROP CONSTRAINT IF EXISTS fk_audit_logs_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    ADD COLUMN author_email TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    ADD COLUMN author_name TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE lib_audit_events
SET author_email = users.email,
    author_name  = users.name
FROM users
WHERE users.id = lib_audit_events.author_user_id;
-- +goose StatementEnd

-- An internal database restored from a dump is not guaranteed to have kept the old foreign key, so
-- an event can name an account that no longer exists; such an event stays without an author.
-- +goose StatementBegin
UPDATE lib_audit_events
SET author_user_id = NULL
WHERE author_user_id IS NOT NULL
  AND author_email IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    ALTER COLUMN created_at TYPE TIMESTAMPTZ
    USING created_at AT TIME ZONE 'UTC';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    ADD CONSTRAINT ck_lib_audit_events_author
    CHECK (num_nonnulls(author_user_id, author_email, author_name) IN (0, 3));
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX IF EXISTS idx_audit_logs_user_id
    RENAME TO idx_lib_audit_events_author_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX IF EXISTS idx_audit_logs_workspace_id
    RENAME TO idx_lib_audit_events_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX IF EXISTS idx_audit_logs_created_at
    RENAME TO idx_lib_audit_events_created_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER INDEX IF EXISTS idx_lib_audit_events_created_at
    RENAME TO idx_audit_logs_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX IF EXISTS idx_lib_audit_events_workspace_id
    RENAME TO idx_audit_logs_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX IF EXISTS idx_lib_audit_events_author_user_id
    RENAME TO idx_audit_logs_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    DROP CONSTRAINT IF EXISTS ck_lib_audit_events_author;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    ALTER COLUMN created_at TYPE TIMESTAMP
    USING created_at AT TIME ZONE 'UTC';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    DROP COLUMN author_name;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    DROP COLUMN author_email;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE lib_audit_events
SET author_user_id = NULL
WHERE author_user_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM users WHERE users.id = lib_audit_events.author_user_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    ADD CONSTRAINT fk_audit_logs_user_id
    FOREIGN KEY (author_user_id)
    REFERENCES users (id)
    ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    RENAME COLUMN author_user_id TO user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_audit_events
    RENAME TO audit_logs;
-- +goose StatementEnd
