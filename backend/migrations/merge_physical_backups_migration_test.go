package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/migrations"
)

const mergePhysicalBackupsMigrationName = "20261028225903_merge_physical_full_and_incremental_backups.sql"

func TestMergePhysicalBackupsMigration(t *testing.T) {
	t.Parallel()

	t.Run("preserves chains and repairs rows before enforcing the merged schema", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, mergePhysicalBackupsMigrationName)
		workspaceID := insertWorkspace(t, db)
		databaseID := insertMigratedDatabase(t, db, &workspaceID)
		otherDatabaseID := insertMigratedDatabase(t, db, &workspaceID)
		storageID := insertStorage(t, db, workspaceID)
		createdAt := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
		fullID, secondFullID := uuid.New(), uuid.New()
		firstIncrementalID, secondIncrementalID, otherIncrementalID := uuid.New(), uuid.New(), uuid.New()
		for _, backupID := range []uuid.UUID{fullID, secondFullID} {
			execute(t, db, `INSERT INTO backups_physical_full
    (id, database_id, storage_id, timeline_id, status, file_name, start_lsn, stop_lsn, created_at, completed_at,
     encryption, encryption_salt, encryption_iv, manifest_file_name, manifest_encryption_salt, manifest_encryption_iv,
     backup_size_mb, raw_size_mb, backup_duration_ms, compression)
    VALUES (?, ?, ?, 1, 'COMPLETED', 'full', '0/100', '0/200', ?, ?,
     'ENCRYPTED', 'salt', 'iv', 'manifest', 'manifest-salt', 'manifest-iv', 4.5, 12, 1000, 'GZIP')`,
				backupID, databaseID, storageID, createdAt, createdAt)
		}
		for _, incremental := range []struct {
			id, root uuid.UUID
			parent   *uuid.UUID
		}{
			{firstIncrementalID, fullID, nil}, {secondIncrementalID, fullID, &firstIncrementalID}, {otherIncrementalID, secondFullID, nil},
		} {
			execute(t, db, `INSERT INTO backups_physical_incremental
    (id, database_id, storage_id, timeline_id, status, file_name, start_lsn, stop_lsn, created_at, completed_at,
     root_full_backup_id, parent_incremental_backup_id)
    VALUES (?, ?, ?, 1, 'COMPLETED', 'incremental', '0/200', '0/300', ?, ?, ?, ?)`,
				incremental.id, databaseID, storageID, createdAt, createdAt, incremental.root, incremental.parent)
		}
		missingTimeID, missingStopID := uuid.New(), uuid.New()
		for _, backupID := range []uuid.UUID{missingTimeID, missingStopID} {
			execute(t, db, `INSERT INTO backups_physical_full
    (id, database_id, storage_id, timeline_id, status, file_name, start_lsn, stop_lsn, created_at)
    VALUES (?, ?, ?, 1, 'COMPLETED', 'full', '0/100', '0/200', ?)`, backupID, databaseID, storageID, createdAt)
		}
		execute(t, db, "UPDATE backups_physical_full SET stop_lsn = NULL WHERE id = ?", missingStopID)
		olderRunningID, newerRunningID, otherRunningID := uuid.New(), uuid.New(), uuid.New()
		execute(t, db, `INSERT INTO backups_physical_full (id, database_id, storage_id, timeline_id, status, created_at)
   VALUES (?, ?, ?, 1, 'IN_PROGRESS', ?), (?, ?, ?, 1, 'IN_PROGRESS', ?)`,
			olderRunningID, databaseID, storageID, createdAt, otherRunningID, otherDatabaseID, storageID, createdAt)
		execute(t, db, `INSERT INTO backups_physical_incremental
   (id, database_id, storage_id, timeline_id, status, root_full_backup_id, created_at)
   VALUES (?, ?, ?, 1, 'IN_PROGRESS', ?, ?)`, newerRunningID, databaseID, storageID, fullID, createdAt.Add(time.Minute))
		execute(t, db, `INSERT INTO backups_physical_in_flight (database_id, backup_type, backup_id)
   VALUES (?, 'INCREMENTAL', ?)`, databaseID, newerRunningID)

		applyAllMigrations(t, db, mergePhysicalBackupsMigrationName)

		assert.ElementsMatch(t, []uuid.UUID{
			fullID, secondFullID, firstIncrementalID, secondIncrementalID, otherIncrementalID,
			missingTimeID, missingStopID, olderRunningID, newerRunningID, otherRunningID,
		}, listIDs(t, db, "SELECT id FROM backups_physical"))
		assert.ElementsMatch(
			t,
			[]uuid.UUID{newerRunningID, otherRunningID},
			listIDs(t, db, "SELECT id FROM backups_physical WHERE status = 'IN_PROGRESS'"),
		)
		for _, chain := range []struct{ id, root, parent uuid.UUID }{
			{firstIncrementalID, fullID, fullID},
			{secondIncrementalID, fullID, firstIncrementalID},
			{otherIncrementalID, secondFullID, secondFullID},
			{newerRunningID, fullID, fullID},
		} {
			assert.Equal(
				t,
				[]string{"INCREMENTAL"},
				listStrings(t, db, "SELECT kind FROM backups_physical WHERE id = ?", chain.id),
			)
			assert.Equal(
				t,
				[]string{chain.root.String()},
				listStrings(t, db, "SELECT root_full_backup_id::text FROM backups_physical WHERE id = ?", chain.id),
			)
			assert.Equal(
				t,
				[]string{chain.parent.String()},
				listStrings(t, db, "SELECT parent_backup_id::text FROM backups_physical WHERE id = ?", chain.id),
			)
		}
		assert.Equal(t, []string{"FULL"}, listStrings(t, db, "SELECT kind FROM backups_physical WHERE id = ?", fullID))
		assert.Empty(
			t,
			listStrings(
				t,
				db,
				"SELECT parent_backup_id::text FROM backups_physical WHERE id = ? AND parent_backup_id IS NOT NULL",
				fullID,
			),
		)
		assert.Empty(
			t,
			listStrings(
				t,
				db,
				"SELECT root_full_backup_id::text FROM backups_physical WHERE id = ? AND root_full_backup_id IS NOT NULL",
				fullID,
			),
		)
		assert.Equal(t, []string{"salt|iv|manifest|manifest-salt|manifest-iv|GZIP"}, listStrings(
			t,
			db,
			"SELECT encryption_salt || '|' || encryption_iv || '|' || manifest_file_name || '|' || manifest_encryption_salt || '|' || manifest_encryption_iv || '|' || compression FROM backups_physical WHERE id = ?",
			fullID,
		))
		var completionTime time.Time
		require.NoError(
			t,
			db.Raw("SELECT completed_at FROM backups_physical WHERE id = ?", missingTimeID).Scan(&completionTime).Error,
		)
		assert.Equal(t, createdAt, completionTime.UTC())
		assert.Equal(
			t,
			[]string{"COMPLETED"},
			listStrings(t, db, "SELECT status FROM backups_physical WHERE id = ?", missingTimeID),
		)
		assert.Equal(
			t,
			[]string{"ERROR"},
			listStrings(t, db, "SELECT status FROM backups_physical WHERE id = ?", missingStopID),
		)
		assert.Equal(
			t,
			[]string{"The upgrade marked an incomplete backup failed: missing stop_lsn"},
			listStrings(t, db, "SELECT fail_message FROM backups_physical WHERE id = ?", missingStopID),
		)
		assert.Equal(
			t,
			[]string{"APPLICATION_RESTART"},
			listStrings(t, db, "SELECT error_reason FROM backups_physical WHERE id = ?", olderRunningID),
		)
		assert.Equal(
			t,
			[]string{"Backup was interrupted by an application restart and marked failed. Trigger a new backup."},
			listStrings(t, db, "SELECT fail_message FROM backups_physical WHERE id = ?", olderRunningID),
		)

		for _, constraint := range []struct {
			assignment, name string
			backupID         uuid.UUID
		}{
			{"parent_backup_id = id", "chain", fullID},
			{"root_full_backup_id = id", "chain", fullID},
			{"parent_backup_id = NULL", "chain", firstIncrementalID},
			{"root_full_backup_id = NULL", "chain", firstIncrementalID},
			{"file_name = NULL", "completed", fullID},
			{"start_lsn = NULL", "completed", fullID},
			{"stop_lsn = NULL", "completed", fullID},
			{"completed_at = NULL", "completed", fullID},
		} {
			err := db.Exec(
				"UPDATE backups_physical SET "+constraint.assignment+" WHERE id = ?",
				constraint.backupID,
			).Error
			require.Error(t, err)
			assert.Contains(t, err.Error(), "ck_backups_physical_"+constraint.name)
		}
		err := db.Exec("UPDATE backups_physical SET status = 'IN_PROGRESS' WHERE id = ?", olderRunningID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "uq_backups_physical_database_in_progress")
		assert.ElementsMatch(t, []string{
			"backups_physical_pkey", "idx_backups_physical_database_id_status_created_at",
			"idx_backups_physical_database_id_timeline_id", "idx_backups_physical_root_full_start_lsn",
			"idx_backups_physical_parent_backup_id", "idx_backups_physical_database_id_kind_created_at",
			"uq_backups_physical_database_in_progress", "idx_backups_physical_storage_id",
		}, listStrings(t, db, "SELECT indexname FROM pg_indexes WHERE tablename = 'backups_physical'"))
		for _, reference := range []string{"database", "storage", "parent_backup", "root_full_backup"} {
			assert.Equal(
				t,
				[]string{"r"},
				listStrings(
					t,
					db,
					"SELECT confdeltype::text FROM pg_constraint WHERE conname = ?",
					"fk_backups_physical_"+reference,
				),
			)
		}
		for _, oldTable := range []string{"backups_physical_full", "backups_physical_incremental", "backups_physical_in_flight"} {
			assert.Empty(
				t,
				listStrings(
					t,
					db,
					"SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename = ?",
					oldTable,
				),
			)
		}
		sqlDB, err := db.DB()
		require.NoError(t, err)
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			sqlDB,
			migrations.Files,
			goose.WithDisableGlobalRegistry(true),
		)
		require.NoError(t, err)
		_, err = provider.ApplyVersion(t.Context(), 20261028225903, false)
		require.ErrorContains(t, err, "one-way")
		require.ErrorContains(t, err, "restore a dump taken before the upgrade")
		assert.Len(t, listIDs(t, db, "SELECT id FROM backups_physical"), 10)
	})
}
