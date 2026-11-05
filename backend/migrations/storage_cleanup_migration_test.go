package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageCleanupMigration(t *testing.T) {
	t.Parallel()

	t.Run("creates a single queue with paired attribution after discarding unreleased cleanup", func(t *testing.T) {
		t.Parallel()

		const migrationName = "20261104120000_simplify_storage_cleanup.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := insertStorage(t, db, workspaceID)
		execute(
			t,
			db,
			`INSERT INTO storages_pending_deletions (storage_id, file_name) VALUES (?, 'unreleased.dump')`,
			storageID,
		)

		applyAllMigrations(t, db, migrationName)

		assert.Empty(t, listIDs(t, db, "SELECT id FROM storages_pending_file_deletions"))

		var obsoleteTables []string
		require.NoError(t, db.WithContext(t.Context()).Raw(`SELECT table_name FROM information_schema.tables
            WHERE table_schema='public' AND table_name IN ('storages_pending_deletions', 'storages_deletion_batches', 'storages_deletion_batch_items', 'backups_storage_switches')`).Scan(&obsoleteTables).Error)
		assert.Empty(t, obsoleteTables)

		databaseID := uuid.New()
		execute(
			t,
			db,
			`INSERT INTO storages_pending_file_deletions (storage_id, file_name, cleanup_database_id, cleanup_workspace_id)
            VALUES (?, 'scoped.dump', ?, ?)`,
			storageID,
			databaseID,
			workspaceID,
		)

		err := db.WithContext(t.Context()).
			Exec(`INSERT INTO storages_pending_file_deletions (storage_id, file_name, cleanup_database_id)
            VALUES (?, 'invalid.dump', ?)`, storageID, databaseID).
			Error
		require.Error(t, err)

		err = db.WithContext(t.Context()).Exec(`INSERT INTO storages_pending_file_deletions (storage_id, file_name)
            VALUES (?, 'scoped.dump')`, storageID).Error
		require.Error(t, err)

		execute(t, db, "DELETE FROM storages WHERE id=?", storageID)

		assert.Empty(t, listIDs(t, db, "SELECT id FROM storages_pending_file_deletions"))
	})
}
