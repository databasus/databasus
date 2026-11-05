package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkspacesMigration(t *testing.T) {
	t.Parallel()

	t.Run("preserves the earliest owner and makes additional owners admins after migration", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261025140000_rename_workspace_memberships_to_workspaces_members.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		workspaceID := insertWorkspace(t, db)
		ownerID := uuid.New()
		additionalOwnerID := uuid.New()
		for _, userID := range []uuid.UUID{ownerID, additionalOwnerID} {
			execute(t, db, `INSERT INTO users (id, email, password_creation_time, role, name, status)
				VALUES (?, ?, NOW(), 'MEMBER', 'Upgrade user', 'ACTIVE')`, userID, userID.String()+"@example.com")
		}
		execute(t, db, `INSERT INTO workspace_memberships (user_id, workspace_id, role, created_at)
			VALUES (?, ?, 'WORKSPACE_OWNER', '2026-01-01'), (?, ?, 'WORKSPACE_OWNER', '2026-01-02')`,
			ownerID, workspaceID, additionalOwnerID, workspaceID)

		applyAllMigrations(t, db, migrationName)

		assert.Equal(t, []string{"WORKSPACE_OWNER"}, listStrings(t, db,
			"SELECT role FROM workspaces_members WHERE user_id = ?", ownerID))
		assert.Equal(t, []string{"WORKSPACE_ADMIN"}, listStrings(t, db,
			"SELECT role FROM workspaces_members WHERE user_id = ?", additionalOwnerID))
	})
}
