package migrations_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/migrations"
)

const physicalBackupsMigrationName = "20261027214332_move_physical_backups_into_module.sql"

func TestPhysicalBackupsMigration(t *testing.T) {
	t.Parallel()

	t.Run("repairs invalid values and preserves the chain with renamed and constrained tables", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, physicalBackupsMigrationName)
		workspaceID := insertWorkspace(t, db)
		databaseID := insertMigratedDatabase(t, db, &workspaceID)
		storageID := insertStorage(t, db, workspaceID)
		fullID := uuid.New()
		firstIncrementalID := uuid.New()
		secondIncrementalID := uuid.New()
		execute(t, db, `INSERT INTO physical_full_backups
   (id, database_id, storage_id, timeline_id, status, file_name, manifest_file_name)
   VALUES (?, ?, ?, 1, 'COMPLETED', 'full', 'full-manifest')`, fullID, databaseID, storageID)
		for _, incremental := range []struct {
			id     uuid.UUID
			parent *uuid.UUID
		}{
			{firstIncrementalID, nil}, {secondIncrementalID, &firstIncrementalID},
		} {
			execute(t, db, `INSERT INTO physical_incremental_backups
    (id, database_id, storage_id, timeline_id, status, root_full_backup_id, parent_incremental_backup_id)
    VALUES (?, ?, ?, 1, 'COMPLETED', ?, ?)`, incremental.id, databaseID, storageID, fullID, incremental.parent)
		}

		type repairCase struct {
			column   string
			value    any
			salt     *string
			message  *string
			repaired string
			status   string
			failure  *string
		}
		repairs := []repairCase{
			{
				column:   "status",
				value:    "PAUSED",
				repaired: "ERROR",
				status:   "ERROR",
				failure:  new("The upgrade replaced the unknown status PAUSED"),
			},
			{
				column:   "compression",
				value:    "BZIP",
				repaired: "ZSTD",
				status:   "ERROR",
				failure:  new("The upgrade replaced the unknown compression BZIP"),
			},
			{column: "encryption", value: "AES", repaired: "NONE", status: "COMPLETED"},
			{column: "encryption", value: "AES", salt: new("salt"), repaired: "ENCRYPTED", status: "COMPLETED"},
			{
				column:   "error_reason",
				value:    "UNKNOWN",
				repaired: "",
				status:   "COMPLETED",
				failure:  new("The upgrade cleared the unknown error reason UNKNOWN"),
			},
			{
				column:   "error_reason",
				value:    "UNKNOWN",
				message:  new("original failure"),
				repaired: "",
				status:   "COMPLETED",
				failure:  new("original failure; The upgrade cleared the unknown error reason UNKNOWN"),
			},
			{
				column:   "error_reason",
				value:    "UNKNOWN",
				message:  new(""),
				repaired: "",
				status:   "COMPLETED",
				failure:  new("The upgrade cleared the unknown error reason UNKNOWN"),
			},
		}
		repairedIDs := map[string][]uuid.UUID{}
		for _, table := range []string{"physical_full_backups", "physical_incremental_backups"} {
			for _, repair := range repairs {
				backupID := uuid.New()
				columns, values := "", ""
				arguments := []any{backupID, databaseID, storageID}
				if table == "physical_incremental_backups" {
					columns = ", root_full_backup_id"
					values = ", ?"
					arguments = append(arguments, fullID)
				}
				execute(
					t,
					db,
					"INSERT INTO "+table+" (id, database_id, storage_id, timeline_id, status"+columns+") VALUES (?, ?, ?, 1, 'COMPLETED'"+values+")",
					arguments...)
				execute(
					t,
					db,
					"UPDATE "+table+" SET "+repair.column+" = ?, encryption_salt = ?, fail_message = ? WHERE id = ?",
					repair.value,
					repair.salt,
					repair.message,
					backupID,
				)
				repairedIDs[table] = append(repairedIDs[table], backupID)
			}
		}
		for _, salt := range []*string{nil, new("salt")} {
			execute(
				t,
				db,
				`INSERT INTO physical_wal_segments (database_id, storage_id, timeline_id, wal_filename, start_lsn, end_lsn, encryption, encryption_salt)
    VALUES (?, ?, 1, '000000010000000000000001', ?::pg_lsn, '0/3000000', 'AES', ?)`,
				databaseID,
				storageID,
				fmt.Sprintf("0/%X", len(listIDs(t, db, "SELECT id FROM physical_wal_segments"))+1),
				salt,
			)
		}
		execute(
			t,
			db,
			`INSERT INTO physical_wal_history_files (database_id, storage_id, timeline_id, file_name, history_filename)
   VALUES (?, ?, 2, 'history', '00000002.history')`,
			databaseID,
			storageID,
		)
		execute(
			t,
			db,
			`INSERT INTO physical_in_flight_backups (database_id, backup_type, backup_id) VALUES (?, 'FULL', ?)`,
			databaseID,
			fullID,
		)
		execute(t, db, `INSERT INTO physical_wal_streamers (database_id, status) VALUES (?, 'UNKNOWN')`, databaseID)
		insertOldPhysicalSample(t, db, databaseID, "UNKNOWN")
		insertOldPhysicalSample(t, db, databaseID, "REACHABLE")

		applyMigrationsBefore(t, db, mergePhysicalBackupsMigrationName)

		for oldTable, table := range map[string]string{"physical_full_backups": "backups_physical_full", "physical_incremental_backups": "backups_physical_incremental"} {
			for index, repair := range repairs {
				backupID := repairedIDs[oldTable][index]
				assert.Equal(
					t,
					statusRow{Status: repair.status, FailMessage: repair.failure},
					readStatusRow(t, db, table, backupID),
				)
				assert.Equal(
					t,
					[]string{repair.repaired},
					listStrings(t, db, "SELECT COALESCE("+repair.column+", '') FROM "+table+" WHERE id = ?", backupID),
				)
			}
		}
		assert.Equal(
			t,
			[]uuid.UUID{fullID},
			listIDs(t, db, "SELECT id FROM backups_physical_full WHERE file_name = 'full'"),
		)
		assert.Equal(
			t,
			[]uuid.UUID{firstIncrementalID},
			listIDs(
				t,
				db,
				"SELECT parent_incremental_backup_id FROM backups_physical_incremental WHERE id = '"+secondIncrementalID.String()+"'",
			),
		)
		assert.Equal(
			t,
			[]string{"NONE", "ENCRYPTED"},
			listStrings(t, db, "SELECT encryption FROM backups_physical_wal_segments ORDER BY start_lsn"),
		)
		assert.Equal(t, []string{"FAILED"}, listStrings(t, db, "SELECT status FROM backups_physical_wal_streamers"))
		assert.Equal(
			t,
			[]string{"REACHABLE"},
			listStrings(t, db, "SELECT source_state FROM backups_physical_wal_stream_samples"),
		)

		err := db.WithContext(t.Context()).Exec("UPDATE backups_physical_in_flight SET backup_type = 'UNKNOWN'").Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ck_backups_physical_in_flight_backup_type")

		execute(t, db, "UPDATE backups_physical_wal_stream_samples SET slot_wal_status = 'FUTURE_POSTGRES_VALUE'")
		for _, suffix := range []string{"full", "incremental", "in_flight", "wal_segments", "wal_history_files", "wal_streamers", "wal_stream_samples"} {
			table := "backups_physical_" + suffix
			assert.NotEmpty(t, listStrings(t, db, "SELECT database_id::text FROM "+table))
			assert.Equal(
				t,
				[]string{"r"},
				listStrings(
					t,
					db,
					`SELECT confdeltype::text FROM pg_constraint WHERE conname = ?`,
					"fk_"+table+"_database",
				),
			)
		}
		assert.Empty(
			t,
			listStrings(
				t,
				db,
				`SELECT conname FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND (conname LIKE 'fk_physical_%' OR conname LIKE 'physical_%' OR conname LIKE 'uk_physical_%' OR conname LIKE 'chk_physical_%')`,
			),
		)
		assert.Empty(
			t,
			listStrings(
				t,
				db,
				`SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND (indexname LIKE 'idx_physical_%' OR indexname LIKE 'physical_%' OR indexname LIKE 'uk_physical_%')`,
			),
		)
		require.Error(t, db.WithContext(t.Context()).Exec("DELETE FROM databases WHERE id = ?", databaseID).Error)
		execute(t, db, "DELETE FROM backups_physical_incremental WHERE database_id = ?", databaseID)
		assert.Empty(t, listIDs(t, db, "SELECT id FROM backups_physical_incremental"))

		sqlDB, err := db.DB()
		require.NoError(t, err)
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			sqlDB,
			migrations.Files,
			goose.WithDisableGlobalRegistry(true),
		)
		require.NoError(t, err)
		_, err = provider.Down(t.Context())
		require.NoError(t, err)
		for _, table := range []string{"physical_full_backups", "physical_incremental_backups", "physical_in_flight_backups", "physical_wal_segments", "physical_wal_history_files", "physical_wal_streamers", "physical_wal_stream_samples"} {
			assert.Equal(
				t,
				[]string{"c"},
				listStrings(
					t,
					db,
					"SELECT confdeltype::text FROM pg_constraint WHERE conname = ?",
					"fk_"+table+"_database_id",
				),
			)
			assert.NotEmpty(
				t,
				listStrings(
					t,
					db,
					"SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename = ?",
					table,
				),
			)
		}
		execute(
			t,
			db,
			"UPDATE physical_full_backups SET status = 'UNKNOWN', compression = 'UNKNOWN', encryption = 'UNKNOWN', error_reason = 'UNKNOWN'",
		)
		execute(t, db, "UPDATE physical_wal_streamers SET status = 'UNKNOWN'")
		execute(t, db, "UPDATE physical_wal_stream_samples SET source_state = 'UNKNOWN'")
		execute(t, db, "UPDATE physical_wal_segments SET encryption = 'UNKNOWN'")
		execute(t, db, "DELETE FROM databases WHERE id = ?", databaseID)
		assert.Empty(t, listIDs(t, db, "SELECT id FROM physical_full_backups"))
	})
}

func insertOldPhysicalSample(t *testing.T, db *gorm.DB, databaseID uuid.UUID, sourceState string) {
	t.Helper()
	execute(t, db, `INSERT INTO physical_wal_stream_samples
 (database_id, sampled_at, session_started_at, source_state, archived_segments, archived_wal_bytes,
 archived_stored_bytes, wal_segment_size_bytes, highest_concurrent_uploads, local_queue_bytes,
 local_queue_pause_bytes, receiver_restarts, receiver_pauses, is_receiver_running, is_forced_rotation_denied)
 VALUES (?, NOW(), NOW(), ?, 0, 0, 0, 16777216, 0, 0, 0, 0, 0, TRUE, FALSE)`, databaseID, sourceState)
}
