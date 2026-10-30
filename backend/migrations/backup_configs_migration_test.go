package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const backupConfigsMigrationName = "20261027120000_move_backup_configs_into_module.sql"

func TestBackupConfigsMigration(t *testing.T) {
	t.Parallel()

	t.Run("keeps the config of the database's method when a database has a config of each method", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupConfigsMigrationName)
		workspaceID := insertWorkspace(t, db)

		mysqlDatabaseID := insertOldDatabase(t, db, "MYSQL", &workspaceID)
		insertOldLogicalBackupConfig(t, db, mysqlDatabaseID, "BACKUP_FAILED")
		insertOldPhysicalBackupConfig(t, db, mysqlDatabaseID, "WAL_GAP")
		physicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_PHYSICAL", &workspaceID)
		insertOldLogicalBackupConfig(t, db, physicalDatabaseID, "BACKUP_FAILED")
		insertOldPhysicalBackupConfig(t, db, physicalDatabaseID, "WAL_GAP")

		applyAllMigrations(t, db, backupConfigsMigrationName)

		assert.Equal(t, []string{"LOGICAL"},
			listStrings(t, db, "SELECT method FROM backups_configs WHERE database_id = ?", mysqlDatabaseID))
		assert.Equal(t, []string{"PHYSICAL"},
			listStrings(t, db, "SELECT method FROM backups_configs WHERE database_id = ?", physicalDatabaseID))
		assert.ElementsMatch(t, []uuid.UUID{mysqlDatabaseID},
			listIDs(t, db, "SELECT database_id FROM backups_configs_logical"))
		assert.ElementsMatch(t, []uuid.UUID{physicalDatabaseID},
			listIDs(t, db, "SELECT database_id FROM backups_configs_physical"))
	})

	t.Run("replaces with the method's default every value the new constraints reject", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupConfigsMigrationName)
		workspaceID := insertWorkspace(t, db)

		logicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_LOGICAL", &workspaceID)
		execute(t, db, `INSERT INTO logical_backup_configs (database_id, send_notifications_on, interval_type,
			retention_policy_type, retention_time_period, retention_count, encryption, max_failed_tries_count)
			VALUES (?, '', 'YEARLY', 'FOREVER_AND_EVER', 'DECADE', 12, 'AES', 4)`, logicalDatabaseID)
		physicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_PHYSICAL", &workspaceID)
		execute(t, db, `INSERT INTO physical_backup_configs (database_id, send_notifications_on, full_interval_type,
			incremental_interval_type, retention, chains_retention_count, full_backups_retention_policy,
			full_backups_retention_count, encryption, wal_lag_threshold_bytes)
			VALUES (?, '', 'YEARLY', 'SECONDLY', 'EVERYTHING', 5, 'SOME', 9, 'AES', 1024)`, physicalDatabaseID)

		applyAllMigrations(t, db, backupConfigsMigrationName)

		var logicalRow struct {
			IntervalType        string
			TimeOfDay           *string
			RetentionPolicyType string
			RetentionTimePeriod string
			RetentionCount      int
			MaxFailedTriesCount int
			Encryption          string
		}
		require.NoError(t, db.WithContext(t.Context()).Raw(`SELECT interval_type, time_of_day, retention_policy_type,
			retention_time_period, retention_count, max_failed_tries_count, backups_configs.encryption
			FROM backups_configs_logical JOIN backups_configs USING (database_id)
			WHERE database_id = ?`, logicalDatabaseID).Scan(&logicalRow).Error)

		assert.Equal(t, "DAILY", logicalRow.IntervalType)
		assert.Equal(t, new("04:00"), logicalRow.TimeOfDay)
		assert.Equal(t, "TIME_PERIOD", logicalRow.RetentionPolicyType)
		assert.Equal(t, "3_MONTH", logicalRow.RetentionTimePeriod)
		assert.Equal(t, 12, logicalRow.RetentionCount)
		assert.Equal(t, 4, logicalRow.MaxFailedTriesCount)
		assert.Equal(t, "NONE", logicalRow.Encryption)

		var physicalRow struct {
			FullIntervalType           string
			FullTimeOfDay              *string
			IncrementalIntervalType    string
			Retention                  string
			ChainsRetentionCount       int
			FullBackupsRetentionPolicy string
			FullBackupsRetentionCount  int
			WalLagThresholdBytes       int64
			Encryption                 string
		}
		require.NoError(t, db.WithContext(t.Context()).Raw(`SELECT full_interval_type, full_time_of_day,
			incremental_interval_type, retention, chains_retention_count, full_backups_retention_policy,
			full_backups_retention_count, wal_lag_threshold_bytes, backups_configs.encryption
			FROM backups_configs_physical JOIN backups_configs USING (database_id)
			WHERE database_id = ?`, physicalDatabaseID).Scan(&physicalRow).Error)

		assert.Equal(t, "DAILY", physicalRow.FullIntervalType)
		assert.Equal(t, new("04:00"), physicalRow.FullTimeOfDay)
		assert.Empty(t, physicalRow.IncrementalIntervalType)
		assert.Equal(t, "FULL_BACKUPS", physicalRow.Retention)
		assert.Equal(t, 5, physicalRow.ChainsRetentionCount)
		assert.Equal(t, "LAST_N", physicalRow.FullBackupsRetentionPolicy)
		assert.Equal(t, 9, physicalRow.FullBackupsRetentionCount)
		assert.Equal(t, int64(1024), physicalRow.WalLagThresholdBytes)
		assert.Equal(t, "NONE", physicalRow.Encryption)
	})

	t.Run("keeps the triggers of the method in order without empty, duplicate and unknown entries", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupConfigsMigrationName)
		workspaceID := insertWorkspace(t, db)

		logicalDatabaseID := insertOldDatabase(t, db, "MARIADB", &workspaceID)
		insertOldLogicalBackupConfig(t, db, logicalDatabaseID,
			",BACKUP_SUCCESS,,BACKUP_FAILED,BACKUP_SUCCESS,WAL_GAP,UNKNOWN")
		physicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_PHYSICAL", &workspaceID)
		insertOldPhysicalBackupConfig(t, db, physicalDatabaseID, "WAL_GAP,,CHAIN_BROKEN,WAL_GAP,NOPE,BACKUP_FAILED")
		silentDatabaseID := insertOldDatabase(t, db, "MONGODB", &workspaceID)
		insertOldLogicalBackupConfig(t, db, silentDatabaseID, "")

		applyAllMigrations(t, db, backupConfigsMigrationName)

		assert.Equal(t, pq.StringArray{"BACKUP_SUCCESS", "BACKUP_FAILED"}, getTriggers(t, db, logicalDatabaseID))
		assert.Equal(t, pq.StringArray{"WAL_GAP", "CHAIN_BROKEN", "BACKUP_FAILED"},
			getTriggers(t, db, physicalDatabaseID))
		assert.Equal(t, pq.StringArray{}, getTriggers(t, db, silentDatabaseID))
	})

	t.Run("keeps every value of a valid config of each method", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupConfigsMigrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := insertStorage(t, db, workspaceID)

		logicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_LOGICAL", &workspaceID)
		execute(t, db, `INSERT INTO logical_backup_configs (database_id, is_backups_enabled, storage_id,
			send_notifications_on, is_retry_if_failed, max_failed_tries_count, encryption, retention_policy_type,
			retention_time_period, retention_count, retention_gfs_hours, retention_gfs_days, retention_gfs_weeks,
			retention_gfs_months, retention_gfs_years, interval_type, time_of_day, weekday, day_of_month,
			cron_expression)
			VALUES (?, TRUE, ?, 'BACKUP_FAILED', TRUE, 5, 'ENCRYPTED', 'GFS', 'YEAR', 4, 1, 2, 3, 4, 5, 'WEEKLY',
			'02:30', 3, 17, '0 3 * * 1')`, logicalDatabaseID, storageID)
		physicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_PHYSICAL", &workspaceID)
		execute(t, db, `INSERT INTO physical_backup_configs (database_id, is_backups_enabled, storage_id,
			send_notifications_on, encryption, full_interval_type, full_time_of_day, full_weekday,
			incremental_interval_type, incremental_time_of_day, retention, chains_retention_count,
			full_backups_retention_policy, full_backups_retention_count, full_backups_retention_gfs_hours,
			full_backups_retention_gfs_days, full_backups_retention_gfs_weeks, full_backups_retention_gfs_months,
			full_backups_retention_gfs_years, wal_lag_threshold_bytes, force_full_requested_at,
			force_incremental_requested_at)
			VALUES (?, TRUE, ?, 'CHAIN_BROKEN,BACKUP_SUCCESS', 'ENCRYPTED', 'WEEKLY', '01:15', 2, 'DAILY', '05:45',
			'CHAINS_AND_FULL_BACKUPS', 6, 'GFS', 0, 1, 2, 3, 4, 5, 2048, '2026-10-01 10:00:00+00',
			'2026-10-02 11:00:00+00')`, physicalDatabaseID, storageID)

		logicalChildBefore := readRowAsText(t, db, `SELECT retention_policy_type, retention_time_period,
			retention_count, retention_gfs_hours, retention_gfs_days, retention_gfs_weeks, retention_gfs_months,
			retention_gfs_years, interval_type, time_of_day, weekday, day_of_month, cron_expression,
			is_retry_if_failed, max_failed_tries_count FROM logical_backup_configs WHERE database_id = ?`,
			logicalDatabaseID)
		physicalChildBefore := readRowAsText(t, db, `SELECT full_interval_type, full_time_of_day, full_weekday,
			full_day_of_month, full_cron_expression, incremental_interval_type, incremental_time_of_day,
			incremental_weekday, incremental_day_of_month, incremental_cron_expression, retention,
			chains_retention_count, full_backups_retention_policy, full_backups_retention_count,
			full_backups_retention_gfs_hours, full_backups_retention_gfs_days, full_backups_retention_gfs_weeks,
			full_backups_retention_gfs_months, full_backups_retention_gfs_years, wal_lag_threshold_bytes,
			force_full_requested_at, force_incremental_requested_at
			FROM physical_backup_configs WHERE database_id = ?`, physicalDatabaseID)

		applyAllMigrations(t, db, backupConfigsMigrationName)

		assert.Equal(t, logicalChildBefore, readRowAsText(t, db, `SELECT retention_policy_type, retention_time_period,
			retention_count, retention_gfs_hours, retention_gfs_days, retention_gfs_weeks, retention_gfs_months,
			retention_gfs_years, interval_type, time_of_day, weekday, day_of_month, cron_expression,
			is_retry_if_failed, max_failed_tries_count FROM backups_configs_logical WHERE database_id = ?`,
			logicalDatabaseID))
		assert.Equal(t, physicalChildBefore, readRowAsText(t, db, `SELECT full_interval_type, full_time_of_day,
			full_weekday, full_day_of_month, full_cron_expression, incremental_interval_type, incremental_time_of_day,
			incremental_weekday, incremental_day_of_month, incremental_cron_expression, retention,
			chains_retention_count, full_backups_retention_policy, full_backups_retention_count,
			full_backups_retention_gfs_hours, full_backups_retention_gfs_days, full_backups_retention_gfs_weeks,
			full_backups_retention_gfs_months, full_backups_retention_gfs_years, wal_lag_threshold_bytes,
			force_full_requested_at, force_incremental_requested_at
			FROM backups_configs_physical WHERE database_id = ?`, physicalDatabaseID))
		assert.Equal(t,
			[]string{"LOGICAL|t|" + storageID.String() + "|ENCRYPTED|{BACKUP_FAILED}"},
			listStrings(t, db, `SELECT concat_ws('|', method, is_backups_enabled, storage_id, encryption,
				notification_triggers) FROM backups_configs WHERE database_id = ?`, logicalDatabaseID))
		assert.Equal(t,
			[]string{"PHYSICAL|t|" + storageID.String() + "|ENCRYPTED|{CHAIN_BROKEN,BACKUP_SUCCESS}"},
			listStrings(t, db, `SELECT concat_ws('|', method, is_backups_enabled, storage_id, encryption,
				notification_triggers) FROM backups_configs WHERE database_id = ?`, physicalDatabaseID))
	})

	t.Run("refuses every row the new constraints forbid", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupConfigsMigrationName)
		applyAllMigrations(t, db, backupConfigsMigrationName)
		workspaceID := insertWorkspace(t, db)
		logicalDatabaseID := insertOldDatabase(t, db, "MYSQL", &workspaceID)
		execute(t, db, "INSERT INTO backups_configs (database_id, method) VALUES (?, 'LOGICAL')", logicalDatabaseID)
		execute(t, db, "INSERT INTO backups_configs_logical (database_id, interval_type) VALUES (?, 'DAILY')",
			logicalDatabaseID)
		unsavedDatabaseID := insertOldDatabase(t, db, "MYSQL", &workspaceID)

		for _, refusedWrite := range []struct {
			name      string
			statement string
			values    []any
		}{
			{
				"a second config of a database", "INSERT INTO backups_configs (database_id, method) VALUES (?, 'PHYSICAL')",
				[]any{logicalDatabaseID},
			},
			{
				"a config of a database that does not exist",
				"INSERT INTO backups_configs (database_id, method) VALUES (?, 'LOGICAL')",
				[]any{uuid.New()},
			},
			{
				"a config of a storage that does not exist",
				"INSERT INTO backups_configs (database_id, method, storage_id) VALUES (?, 'LOGICAL', ?)",
				[]any{unsavedDatabaseID, uuid.New()},
			},
			{"a method row without its config", `INSERT INTO backups_configs_logical (database_id, interval_type)
				VALUES (?, 'DAILY')`, []any{unsavedDatabaseID}},
			{"a physical row on a logical config", `INSERT INTO backups_configs_physical
				(database_id, full_interval_type) VALUES (?, 'DAILY')`, []any{logicalDatabaseID}},
			{
				"the deletion of a database with a config", "DELETE FROM databases WHERE id = ?",
				[]any{logicalDatabaseID},
			},
		} {
			refusal := db.WithContext(t.Context()).Exec(refusedWrite.statement, refusedWrite.values...)
			require.Error(t, refusal.Error, refusedWrite.name)
		}

		physicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_PHYSICAL", &workspaceID)
		execute(t, db, "INSERT INTO backups_configs (database_id, method) VALUES (?, 'PHYSICAL')", physicalDatabaseID)
		execute(t, db, "INSERT INTO backups_configs_physical (database_id, full_interval_type) VALUES (?, 'DAILY')",
			physicalDatabaseID)

		execute(t, db, "DELETE FROM backups_configs WHERE database_id = ?", physicalDatabaseID)
		assert.Empty(t, listIDs(t, db, "SELECT database_id FROM backups_configs_physical"))
	})
}

func insertOldLogicalBackupConfig(t *testing.T, db *gorm.DB, databaseID uuid.UUID, notifications string) {
	t.Helper()

	execute(t, db, `INSERT INTO logical_backup_configs (database_id, send_notifications_on, interval_type,
		retention_time_period) VALUES (?, ?, 'DAILY', 'MONTH')`, databaseID, notifications)
}

func insertOldPhysicalBackupConfig(t *testing.T, db *gorm.DB, databaseID uuid.UUID, notifications string) {
	t.Helper()

	execute(t, db, `INSERT INTO physical_backup_configs (database_id, send_notifications_on, full_interval_type,
		full_time_of_day, full_backups_retention_policy, full_backups_retention_count)
		VALUES (?, ?, 'DAILY', '04:00', 'LAST_N', 7)`, databaseID, notifications)
}

func getTriggers(t *testing.T, db *gorm.DB, databaseID uuid.UUID) pq.StringArray {
	t.Helper()

	var triggers pq.StringArray
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT notification_triggers FROM backups_configs WHERE database_id = ?", databaseID,
	).Row().Scan(&triggers))

	return triggers
}

// readRowAsText reads one row as the text of its record, so two reads compare every column at once.
func readRowAsText(t *testing.T, db *gorm.DB, query string, values ...any) string {
	t.Helper()

	var rowText string
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT selected::text FROM ("+query+") AS selected", values...,
	).Row().Scan(&rowText))

	return rowText
}
