package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageSwitchCleanupMigration(t *testing.T) {
	t.Parallel()

	t.Run("preserves ordinary obligations and keeps historical identities after deletion", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261030120000_storage_switch_cleanup.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		workspaceID := insertWorkspace(t, db)
		databaseID := insertMigratedDatabase(t, db, &workspaceID)
		storageID := insertStorage(t, db, workspaceID)
		pendingID := uuid.New()
		execute(
			t,
			db,
			`INSERT INTO storages_pending_deletions (id,storage_id,file_name,not_before) VALUES (?,?,'old.dump','2026-10-30')`,
			pendingID,
			storageID,
		)
		execute(
			t,
			db,
			`INSERT INTO backups_configs (database_id,method,storage_id) VALUES (?,'LOGICAL',?)`,
			databaseID,
			storageID,
		)

		applyAllMigrations(t, db, migrationName)

		assert.Equal(
			t,
			[]bool{true},
			listBools(t, db, "SELECT is_ordinary FROM storages_pending_deletions WHERE id=?", pendingID),
		)
		assert.Equal(
			t,
			[]string{"0"},
			listStrings(t, db, "SELECT storage_generation::text FROM backups_configs WHERE database_id=?", databaseID),
		)
		assert.Equal(t, []uuid.UUID{storageID}, listIDs(t, db,
			"SELECT last_selected_storage_id FROM backups_configs"))
		batchID := uuid.New()
		execute(
			t,
			db,
			`INSERT INTO storages_deletion_batches (id,status,created_at,cleanup_deadline,completed_at) VALUES (?,'COMPLETED','2026-10-30','2026-10-31','2026-10-30')`,
			batchID,
		)
		execute(
			t,
			db,
			`INSERT INTO storages_deletion_batch_items (batch_id,storage_id,file_name,status) VALUES (?,?,'old.dump','REMAINING')`,
			batchID,
			storageID,
		)
		switchID := uuid.New()
		execute(
			t,
			db,
			`INSERT INTO backups_storage_switches (id,database_id,workspace_id,previous_storage_id,target_storage_id,deletion_batch_id,created_at) VALUES (?,?,?,?,?,?, '2026-10-30')`,
			switchID,
			databaseID,
			workspaceID,
			storageID,
			uuid.New(),
			batchID,
		)
		err := db.WithContext(t.Context()).
			Exec(`INSERT INTO storages_deletion_batch_items (batch_id,storage_id,file_name) VALUES (?,?,'old.dump')`, batchID, storageID).
			Error
		require.Error(t, err)
		err = db.WithContext(t.Context()).Exec(`INSERT INTO backups_storage_switches
			(id,database_id,workspace_id,target_storage_id,deletion_batch_id,created_at)
			VALUES (?,?,?,?,?,'2026-10-30')`, uuid.New(), databaseID, workspaceID, storageID, uuid.New()).Error
		require.Error(t, err, "a storage switch must reference an existing deletion batch")
		err = db.WithContext(t.Context()).Exec("DELETE FROM storages_deletion_batches WHERE id=?", batchID).Error
		require.Error(t, err, "an operation's deletion batch must remain available")
		err = db.WithContext(t.Context()).Exec(`UPDATE storages_deletion_batches SET total=1 WHERE id=?`, batchID).Error
		require.Error(t, err)
		err = db.WithContext(t.Context()).
			Exec(`UPDATE backups_configs SET storage_generation=-1 WHERE database_id=?`, databaseID).
			Error
		require.Error(t, err)
		execute(t, db, "UPDATE backups_configs SET storage_id=NULL WHERE database_id=?", databaseID)
		execute(t, db, "DELETE FROM storages WHERE id=?", storageID)
		assert.Equal(t, []uuid.UUID{storageID}, listIDs(t, db,
			"SELECT last_selected_storage_id FROM backups_configs"))
		execute(t, db, "DELETE FROM backups_configs WHERE database_id=?", databaseID)
		execute(t, db, "DELETE FROM databases WHERE id=?", databaseID)
		assert.Equal(t, []uuid.UUID{switchID}, listIDs(t, db, "SELECT id FROM backups_storage_switches"))
		assert.Equal(t, []uuid.UUID{storageID}, listIDs(t, db, "SELECT storage_id FROM storages_deletion_batch_items"))
		assert.Empty(t, listIDs(t, db, "SELECT id FROM storages_pending_deletions"))
	})
}
