package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/migrations"
)

const verificationMigrationName = "20261029122104_move_verification_into_module.sql"

func TestVerificationMigration(t *testing.T) {
	t.Parallel()

	t.Run("repairs unsupported values while preserving valid configs and runs", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, verificationMigrationName)
		fixture := insertVerificationFixture(t, db)
		configIDs := make([]uuid.UUID, 5)
		for index := range configIDs {
			configIDs[index] = insertMigratedDatabase(t, db, &fixture.workspaceID)
			execute(t, db, "INSERT INTO backup_verification_configs (database_id) VALUES (?)", configIDs[index])
		}
		execute(
			t,
			db,
			"UPDATE backup_verification_configs SET schedule_type = 'UNKNOWN' WHERE database_id = ?",
			configIDs[0],
		)
		execute(t, db, `UPDATE backup_verification_configs SET interval_type = 'UNKNOWN', schedule_type = 'INTERVAL',
   is_scheduled_verification_enabled = true WHERE database_id = ?`, configIDs[1])
		execute(t, db, `UPDATE backup_verification_configs SET send_notifications_on =
   'VERIFICATION_SUCCESS,VERIFICATION_FAILED,VERIFICATION_SUCCESS,UNKNOWN' WHERE database_id = ?`, configIDs[2])
		execute(t, db, `UPDATE backup_verification_configs SET interval_type = 'UNKNOWN', time_of_day = '03:30',
   weekday = 3 WHERE database_id = ?`, configIDs[4])
		unknownStatusID := insertOldVerification(
			t,
			db,
			oldVerification{fixture: fixture, status: "PAUSED", trigger: "MANUAL"},
		)
		execute(
			t,
			db,
			"UPDATE restore_verifications SET fail_message = 'earlier failure' WHERE id = ?",
			unknownStatusID,
		)
		unknownTriggerID := insertOldVerification(
			t,
			db,
			oldVerification{fixture: fixture, status: "PENDING", trigger: "UNKNOWN"},
		)
		completedID := insertOldVerification(
			t,
			db,
			oldVerification{fixture: fixture, status: "COMPLETED", trigger: "SCHEDULED"},
		)
		agentID := uuid.New()
		execute(
			t,
			db,
			"INSERT INTO verification_agents (id, name, token_hash) VALUES (?, 'agent', ?)",
			agentID,
			agentID.String(),
		)
		execute(t, db, "UPDATE restore_verifications SET agent_id = ? WHERE id = ?", agentID, completedID)
		execute(
			t,
			db,
			`INSERT INTO restore_verification_table_stats (restore_verification_id, schema_name, name, row_count)
   VALUES (?, 'public', 'customers', 12), (?, 'public', 'orders', 34)`,
			completedID,
			completedID,
		)

		applyAllMigrations(t, db, verificationMigrationName)

		assert.Equal(t, statusRow{Status: "FAILED", FailMessage: new("The upgrade replaced the unknown status PAUSED")},
			readStatusRow(t, db, "verification_runs", unknownStatusID))
		assert.Equal(
			t,
			statusRow{Status: "PENDING", FailMessage: nil},
			readStatusRow(t, db, "verification_runs", unknownTriggerID),
		)
		assert.Equal(
			t,
			[]string{"MANUAL"},
			listStrings(t, db, "SELECT trigger FROM verification_runs WHERE id = ?", unknownTriggerID),
		)
		assert.Equal(
			t,
			statusRow{Status: "COMPLETED", FailMessage: nil},
			readStatusRow(t, db, "verification_runs", completedID),
		)
		for index, configID := range configIDs {
			var config struct {
				ScheduleType        string
				IntervalType        string
				TimeOfDay           *string
				Weekday             *int
				SendNotificationsOn pq.StringArray `gorm:"type:text[]"`
			}
			require.NoError(
				t,
				db.WithContext(t.Context()).
					Raw("SELECT * FROM verification_configs WHERE database_id = ?", configID).
					Scan(&config).
					Error,
			)
			assert.Equal(t, "WEEKLY", config.IntervalType)
			if index == 1 {
				assert.Equal(t, "INTERVAL", config.ScheduleType)
				assert.Equal(t, new("04:00"), config.TimeOfDay)
				assert.Equal(t, new(0), config.Weekday)
			} else {
				assert.Equal(t, "AFTER_BACKUP", config.ScheduleType)
			}
			if index == 4 {
				assert.Equal(t, new("03:30"), config.TimeOfDay)
				assert.Equal(t, new(3), config.Weekday)
			}
			if index == 2 {
				assert.ElementsMatch(
					t,
					[]string{"VERIFICATION_SUCCESS", "VERIFICATION_FAILED"},
					config.SendNotificationsOn,
				)
			} else {
				assert.NotNil(t, config.SendNotificationsOn)
				assert.Empty(t, config.SendNotificationsOn)
			}
			if index == 3 {
				assert.Nil(t, config.TimeOfDay)
				assert.Nil(t, config.Weekday)
			}
		}
		var statistics []struct {
			Name     string
			RowCount int64
		}
		require.NoError(
			t,
			db.WithContext(t.Context()).
				Raw("SELECT name, row_count FROM verification_run_table_stats WHERE run_id = ? ORDER BY name", completedID).
				Scan(&statistics).
				Error,
		)
		require.Len(t, statistics, 2)
		assert.Equal(t, "customers", statistics[0].Name)
		assert.Equal(t, int64(12), statistics[0].RowCount)
		assert.Equal(t, "orders", statistics[1].Name)
		assert.Equal(t, int64(34), statistics[1].RowCount)
		assert.Equal(
			t,
			[]string{agentID.String()},
			listStrings(t, db, "SELECT agent_id::text FROM verification_runs WHERE id = ?", completedID),
		)
		execute(t, db, "DELETE FROM verification_agents WHERE id = ?", agentID)
		assert.Equal(
			t,
			[]bool{true},
			listBools(t, db, "SELECT agent_id IS NULL FROM verification_runs WHERE id = ?", completedID),
		)
		execute(t, db, "DELETE FROM verification_runs WHERE id = ?", completedID)
		assert.Empty(t, listIDs(t, db, "SELECT id FROM verification_run_table_stats"))
	})

	t.Run("rejects deletion of referenced rows after migration", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, verificationMigrationName)
		fixture := insertVerificationFixture(t, db)
		runID := insertOldVerification(t, db, oldVerification{fixture: fixture, status: "PENDING", trigger: "MANUAL"})
		configDatabaseID := insertMigratedDatabase(t, db, &fixture.workspaceID)
		execute(t, db, "INSERT INTO backup_verification_configs (database_id) VALUES (?)", configDatabaseID)

		applyAllMigrations(t, db, verificationMigrationName)

		refusedStatements := []struct {
			statement  string
			values     []any
			constraint string
		}{
			{"DELETE FROM databases WHERE id = ?", []any{configDatabaseID}, "fk_verification_configs_database"},
			{"DELETE FROM backups_logical WHERE id = ?", []any{fixture.backupID}, "fk_verification_runs_backup"},
		}
		for _, refusedStatement := range refusedStatements {
			err := db.WithContext(t.Context()).Exec(refusedStatement.statement, refusedStatement.values...).Error
			require.Error(t, err)
			assert.Contains(t, err.Error(), refusedStatement.constraint)
		}
		// A backup from another database isolates the run's own reference from the backup's RESTRICT.
		runDatabaseID := insertMigratedDatabase(t, db, &fixture.workspaceID)
		execute(t, db, "UPDATE verification_runs SET database_id = ? WHERE id = ?", runDatabaseID, runID)
		err := db.WithContext(t.Context()).Exec("DELETE FROM databases WHERE id = ?", runDatabaseID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fk_verification_runs_database")
		for _, indexName := range []string{"uq_verification_agents_token_hash", "idx_verification_runs_database_created_at", "idx_verification_runs_status_created_at", "idx_verification_runs_agent_status", "idx_verification_runs_backup", "idx_verification_run_table_stats_run"} {
			assert.Equal(t, []bool{true}, listBools(t, db, "SELECT to_regclass(?) IS NOT NULL", indexName))
		}
		for _, oldName := range []string{"backup_verification_configs", "restore_verifications", "restore_verification_table_stats"} {
			assert.Equal(t, []bool{true}, listBools(t, db, "SELECT to_regclass(?) IS NULL", oldName))
		}
	})
	t.Run("restores the old schema and cascades when rolled back", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, verificationMigrationName)
		fixture := insertVerificationFixture(t, db)
		runID := insertOldVerification(t, db, oldVerification{fixture: fixture, status: "COMPLETED", trigger: "MANUAL"})
		execute(t, db, `INSERT INTO backup_verification_configs (database_id, send_notifications_on)
 VALUES (?, 'VERIFICATION_SUCCESS')`, fixture.databaseID)
		execute(
			t,
			db,
			`INSERT INTO restore_verification_table_stats (restore_verification_id, schema_name, name, row_count)
 VALUES (?, 'public', 'items', 3)`,
			runID,
		)
		applyAllMigrations(t, db, verificationMigrationName)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			sqlDB,
			migrations.Files,
			goose.WithDisableGlobalRegistry(true),
		)
		require.NoError(t, err)
		_, err = provider.ApplyVersion(t.Context(), 20261029175054, false)
		require.NoError(t, err)
		_, err = provider.ApplyVersion(t.Context(), 20261029122104, false)
		require.NoError(t, err)
		assert.Equal(t, []string{"VERIFICATION_SUCCESS"}, listStrings(t, db,
			"SELECT send_notifications_on FROM backup_verification_configs WHERE database_id = ?", fixture.databaseID))
		execute(t, db, "UPDATE restore_verifications SET status = 'UNKNOWN', trigger = 'UNKNOWN' WHERE id = ?", runID)
		execute(t, db, `UPDATE backup_verification_configs SET schedule_type = 'UNKNOWN', interval_type = 'UNKNOWN',
 send_notifications_on = 'UNKNOWN' WHERE database_id = ?`, fixture.databaseID)
		assert.Equal(
			t,
			[]string{runID.String()},
			listStrings(t, db, "SELECT restore_verification_id::text FROM restore_verification_table_stats"),
		)
		assert.Equal(t, []bool{true}, listBools(t, db, "SELECT to_regclass('idx_verification_runs_backup') IS NULL"))
		for _, constraint := range []string{"fk_backup_verification_configs_database", "fk_restore_verifications_database", "fk_restore_verifications_backup"} {
			assert.Equal(
				t,
				[]string{"c"},
				listStrings(t, db, "SELECT confdeltype::text FROM pg_constraint WHERE conname = ?", constraint),
			)
		}
		execute(t, db, "DELETE FROM backups_logical WHERE id = ?", fixture.backupID)
		assert.Empty(t, listIDs(t, db, "SELECT id FROM restore_verifications"))
		assert.Empty(t, listIDs(t, db, "SELECT id FROM restore_verification_table_stats"))
		execute(t, db, "DELETE FROM databases WHERE id = ?", fixture.databaseID)
		assert.Empty(t, listIDs(t, db, "SELECT database_id FROM backup_verification_configs"))
	})
}

type verificationFixture struct {
	workspaceID uuid.UUID
	databaseID  uuid.UUID
	backupID    uuid.UUID
}

func insertVerificationFixture(t *testing.T, db *gorm.DB) verificationFixture {
	t.Helper()
	workspaceID := insertWorkspace(t, db)
	databaseID := insertMigratedDatabase(t, db, &workspaceID)
	storageID := insertStorage(t, db, workspaceID)
	backupID := uuid.New()
	execute(t, db, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
  VALUES (?, ?, ?, 'COMPLETED', 'backup')`, backupID, databaseID, storageID)
	return verificationFixture{workspaceID: workspaceID, databaseID: databaseID, backupID: backupID}
}

type oldVerification struct {
	fixture verificationFixture
	status  string
	trigger string
}

func insertOldVerification(t *testing.T, db *gorm.DB, verification oldVerification) uuid.UUID {
	t.Helper()
	runID := uuid.New()
	execute(t, db, `INSERT INTO restore_verifications (id, database_id, backup_id, status, trigger)
  VALUES (?, ?, ?, ?, ?)`, runID, verification.fixture.databaseID, verification.fixture.backupID, verification.status, verification.trigger)
	return runID
}
