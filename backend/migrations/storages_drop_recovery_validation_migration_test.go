package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoragesDropRecoveryValidationMigration(t *testing.T) {
	t.Parallel()

	t.Run("keeps a local storage that waited for validation and drops its status", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261119120000_storages_drop_recovery_validation.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := uuid.New()
		execute(t, db, `INSERT INTO storages (id, type, name, workspace_id, recovery_validation_status)
			VALUES (?, 'LOCAL', 'local', ?, 'REQUIRES_VALIDATION')`, storageID, workspaceID)

		applyAllMigrations(t, db, migrationName)

		storageIDs := listIDs(t, db, "SELECT id FROM storages")
		require.Len(t, storageIDs, 1)
		assert.Equal(t, storageID, storageIDs[0])

		var statusColumns []string
		require.NoError(t, db.WithContext(t.Context()).Raw(
			`SELECT column_name FROM information_schema.columns
			WHERE table_name = 'storages' AND column_name = 'recovery_validation_status'`,
		).Scan(&statusColumns).Error)
		assert.Empty(t, statusColumns)
	})

	t.Run("restores the status column as available for every storage when rolled back", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261119120000_storages_drop_recovery_validation.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		applyAllMigrations(t, db, migrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := uuid.New()
		execute(t, db, "INSERT INTO storages (id, type, name, workspace_id) VALUES (?, 'LOCAL', 'local', ?)",
			storageID, workspaceID)

		rollBackMigration(t, db, 20261119120000)

		var statuses []string
		require.NoError(t, db.WithContext(t.Context()).
			Raw("SELECT recovery_validation_status FROM storages WHERE id = ?", storageID).Scan(&statuses).Error)
		assert.Equal(t, []string{"AVAILABLE"}, statuses)
		var columnShapes []string
		require.NoError(t, db.WithContext(t.Context()).Raw(
			`SELECT data_type || ' ' || is_nullable || ' ' || column_default FROM information_schema.columns
			WHERE table_name = 'storages' AND column_name = 'recovery_validation_status'`,
		).Scan(&columnShapes).Error)
		assert.Equal(t, []string{"text NO 'AVAILABLE'::text"}, columnShapes)
	})
}
