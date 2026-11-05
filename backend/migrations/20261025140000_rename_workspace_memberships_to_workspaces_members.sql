-- +goose Up
-- +goose StatementBegin
ALTER TABLE workspace_memberships
    RENAME TO workspaces_members;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    RENAME CONSTRAINT workspace_memberships_pkey TO workspaces_members_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    RENAME CONSTRAINT uk_workspace_memberships_user_workspace TO uk_workspaces_members_user_workspace;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    RENAME CONSTRAINT fk_workspace_memberships_workspace_id TO fk_workspaces_members_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_workspace_memberships_workspace_user
    RENAME TO idx_workspaces_members_workspace_user;
-- +goose StatementEnd

-- The unique constraint on (user_id, workspace_id) and the index on (workspace_id, user_id) serve
-- every query these three served.
-- +goose StatementBegin
DROP INDEX idx_workspace_memberships_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_workspace_memberships_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_workspace_memberships_user_workspace;
-- +goose StatementEnd

-- A damaged row loses its rights instead of stopping the upgrade.
-- +goose StatementBegin
UPDATE workspaces_members
SET role = 'WORKSPACE_VIEWER'
WHERE role NOT IN ('WORKSPACE_OWNER', 'WORKSPACE_ADMIN', 'WORKSPACE_MEMBER', 'WORKSPACE_VIEWER');
-- +goose StatementEnd

-- The earliest owner keeps the workspace; every other owner keeps the most of their rights as an
-- admin.
-- +goose StatementBegin
UPDATE workspaces_members
SET role = 'WORKSPACE_ADMIN'
WHERE role = 'WORKSPACE_OWNER'
    AND id NOT IN (
        SELECT DISTINCT ON (workspace_id) id
        FROM workspaces_members
        WHERE role = 'WORKSPACE_OWNER'
        ORDER BY workspace_id, created_at, id
    );
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX uq_workspaces_members_owner
    ON workspaces_members (workspace_id)
    WHERE role = 'WORKSPACE_OWNER';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    DROP CONSTRAINT fk_workspace_memberships_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    ADD CONSTRAINT fk_workspaces_members_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    DROP CONSTRAINT fk_databases_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    ADD CONSTRAINT fk_databases_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    DROP CONSTRAINT fk_storages_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    ADD CONSTRAINT fk_storages_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    DROP CONSTRAINT fk_notifiers_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    ADD CONSTRAINT fk_notifiers_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifiers
    DROP CONSTRAINT fk_notifiers_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    ADD CONSTRAINT fk_notifiers_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    DROP CONSTRAINT fk_storages_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE storages
    ADD CONSTRAINT fk_storages_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    DROP CONSTRAINT fk_databases_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    ADD CONSTRAINT fk_databases_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    DROP CONSTRAINT fk_workspaces_members_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    ADD CONSTRAINT fk_workspace_memberships_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS uq_workspaces_members_owner;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_workspace_memberships_user_workspace
    ON workspaces_members (user_id, workspace_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_workspace_memberships_workspace_id
    ON workspaces_members (workspace_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_workspace_memberships_user_id
    ON workspaces_members (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_workspaces_members_workspace_user
    RENAME TO idx_workspace_memberships_workspace_user;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    RENAME CONSTRAINT fk_workspaces_members_workspace_id TO fk_workspace_memberships_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    RENAME CONSTRAINT uk_workspaces_members_user_workspace TO uk_workspace_memberships_user_workspace;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    RENAME CONSTRAINT workspaces_members_pkey TO workspace_memberships_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE workspaces_members
    RENAME TO workspace_memberships;
-- +goose StatementEnd
