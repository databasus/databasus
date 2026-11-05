package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestStorageReferencesMigration(t *testing.T) {
	t.Parallel()

	t.Run("restricts every backup reference and keeps cascading the pending deletions", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)

		var deleteActions []struct {
			Conname     string
			Confdeltype string
		}
		require.NoError(t, db.WithContext(t.Context()).Raw(
			`SELECT conname, confdeltype::text AS confdeltype FROM pg_constraint
			WHERE contype = 'f' AND confrelid = 'storages'::regclass AND conname IN ?`,
			[]string{
				"fk_backups_configs_storage",
				"fk_backups_logical_storage",
				"fk_backups_physical_storage",
				"fk_backups_physical_wal_segments_storage",
				"fk_backups_physical_wal_history_files_storage",
				"fk_storages_pending_file_deletions_storage_id",
			},
		).Scan(&deleteActions).Error)

		deleteActionByConstraint := map[string]string{}
		for _, deleteAction := range deleteActions {
			deleteActionByConstraint[deleteAction.Conname] = deleteAction.Confdeltype
		}

		assert.Equal(t, map[string]string{
			"fk_backups_configs_storage":                    "r",
			"fk_backups_logical_storage":                    "r",
			"fk_backups_physical_storage":                   "r",
			"fk_backups_physical_wal_segments_storage":      "r",
			"fk_backups_physical_wal_history_files_storage": "r",
			"fk_storages_pending_file_deletions_storage_id": "c",
		}, deleteActionByConstraint)
	})

	t.Run("refuses a direct delete of a storage that backups reference", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		workspaceID := insertWorkspace(t, db)
		databaseID := insertMigratedDatabase(t, db, &workspaceID)

		configStorageID := insertStorage(t, db, workspaceID)
		execute(t, db, `INSERT INTO backups_configs (database_id, method, storage_id) VALUES (?, 'LOGICAL', ?)`,
			databaseID, configStorageID)

		logicalBackupStorageID := insertStorage(t, db, workspaceID)
		logicalBackupID := uuid.New()
		execute(t, db, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
			VALUES (?, ?, ?, 'COMPLETED', 'backup.dump')`, logicalBackupID, databaseID, logicalBackupStorageID)

		physicalBackupStorageID := insertStorage(t, db, workspaceID)
		physicalBackupID := uuid.New()
		execute(t, db, `INSERT INTO backups_physical (id, database_id, storage_id, timeline_id, status, kind)
			VALUES (?, ?, ?, 1, 'ERROR', 'FULL')`, physicalBackupID, databaseID, physicalBackupStorageID)

		for _, storageID := range []uuid.UUID{configStorageID, logicalBackupStorageID, physicalBackupStorageID} {
			deletion := db.WithContext(t.Context()).Exec("DELETE FROM storages WHERE id = ?", storageID)
			require.Error(t, deletion.Error)
			assert.True(t, database.IsForeignKeyViolation(deletion.Error), deletion.Error)
		}

		assert.ElementsMatch(t, []uuid.UUID{configStorageID},
			listIDs(t, db, "SELECT storage_id FROM backups_configs"))
		assert.ElementsMatch(t, []uuid.UUID{logicalBackupID}, listIDs(t, db, "SELECT id FROM backups_logical"))
		assert.ElementsMatch(t, []uuid.UUID{physicalBackupID}, listIDs(t, db, "SELECT id FROM backups_physical"))
		assert.ElementsMatch(t,
			[]uuid.UUID{configStorageID, logicalBackupStorageID, physicalBackupStorageID},
			listIDs(t, db, "SELECT id FROM storages"))
	})
}

func openMigratedDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), testdb.Open(t))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func insertStorage(t *testing.T, db *gorm.DB, workspaceID uuid.UUID) uuid.UUID {
	t.Helper()

	storageID := uuid.New()
	execute(t, db, "INSERT INTO storages (id, type, name, workspace_id) VALUES (?, 'LOCAL', ?, ?)",
		storageID, "storage "+storageID.String(), workspaceID)

	return storageID
}
