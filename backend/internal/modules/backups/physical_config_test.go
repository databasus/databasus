package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const cronWithoutRunDate = "0 0 31 2 *"

//nolint:tparallel // The subtests share one physical PostgreSQL server.
func TestSavePhysicalBackupConfig(t *testing.T) {
	t.Parallel()

	physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")

	t.Run("refuses an incremental schedule slower than the full one", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		slowIncrementalSettings := newIncrementalBackupSettings()
		slowIncrementalSettings.FullBackupSchedule = newDailySchedule("04:00")
		weekday := 1
		slowIncrementalSettings.IncrementalBackupSchedule = &apiclient.Schedule{
			Type:           apiclient.ScheduleTypeWEEKLY,
			TimeOfDay:      new("04:00"),
			Weekday:        &weekday,
			DayOfMonth:     nil,
			CronExpression: nil,
		}

		response := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, slowIncrementalSettings))

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
		assert.Zero(t, countRows(t, tm,
			"SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", physicalDatabase.Id))
	})

	t.Run("refuses a full schedule with no date to run on and keeps the config", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		savedSettings := newIncrementalBackupSettings()
		saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, savedSettings))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
		undatedSettings := newIncrementalBackupSettings()
		undatedSettings.FullBackupSchedule = newCronSchedule(cronWithoutRunDate)

		response := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, undatedSettings))

		requireInvalidField(t, response.StatusCode(), response.Body, "settings.fullBackupSchedule.cronExpression")
		assert.Equal(
			t,
			savedSettings,
			getPhysicalSettings(t, *getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200),
		)
	})

	t.Run("refuses an incremental schedule with no date to run on and keeps the config", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		savedSettings := newIncrementalBackupSettings()
		saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, savedSettings))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
		undatedSchedule := newCronSchedule(cronWithoutRunDate)
		undatedSettings := newIncrementalBackupSettings()
		undatedSettings.IncrementalBackupSchedule = &undatedSchedule

		response := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, undatedSettings))

		requireInvalidField(
			t,
			response.StatusCode(),
			response.Body,
			"settings.incrementalBackupSchedule.cronExpression",
		)
		assert.Equal(
			t,
			savedSettings,
			getPhysicalSettings(t, *getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200),
		)
	})

	t.Run("reads a stored incremental schedule with no date to run on as the built daily schedule", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		weekday := 1
		weeklyFullSettings := newIncrementalBackupSettings()
		weeklyFullSettings.FullBackupSchedule = apiclient.Schedule{
			Type:           apiclient.ScheduleTypeWEEKLY,
			TimeOfDay:      new("03:15"),
			Weekday:        &weekday,
			DayOfMonth:     nil,
			CronExpression: nil,
		}
		saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, weeklyFullSettings))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
		tm.app.ExecuteSQL(t, `UPDATE backups_configs_physical
			SET incremental_interval_type = 'CRON', incremental_cron_expression = ? WHERE database_id = ?`,
			cronWithoutRunDate, physicalDatabase.Id)

		readSettings := getPhysicalSettings(t, *getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200)

		builtSchedule := newDailySchedule("03:15")
		assert.Equal(t, &builtSchedule, readSettings.IncrementalBackupSchedule)
	})

	t.Run(
		"reads a stored incremental schedule with no date to run on as no schedule under an hourly full schedule",
		func(t *testing.T) {
			tm := newTeam(t)
			physicalDatabase := tm.createPhysicalDatabase(t, physicalServer,
				apiclient.PhysicalBackupTypeFULLINCREMENTAL)
			storageID := tm.createLocalStorage(t)
			everyTenMinutes := newCronSchedule("*/10 * * * *")
			hourlyFullSettings := newIncrementalBackupSettings()
			hourlyFullSettings.FullBackupSchedule = apiclient.Schedule{
				Type:           apiclient.ScheduleTypeHOURLY,
				TimeOfDay:      nil,
				Weekday:        nil,
				DayOfMonth:     nil,
				CronExpression: nil,
			}
			hourlyFullSettings.IncrementalBackupSchedule = &everyTenMinutes
			saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
				newPhysicalBackupConfigInput(t, &storageID, hourlyFullSettings))
			require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
			tm.app.ExecuteSQL(t, `UPDATE backups_configs_physical
			SET incremental_cron_expression = ? WHERE database_id = ?`, cronWithoutRunDate, physicalDatabase.Id)

			readSettings := getPhysicalSettings(t, *getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200)

			assert.Nil(t, readSettings.IncrementalBackupSchedule)
		},
	)

	t.Run("refuses a backup type change while the stored full schedule has no date to run on", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		cronFullSettings := newIncrementalBackupSettings()
		cronFullSettings.FullBackupSchedule = newCronSchedule("0 3 * * 1")
		saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, cronFullSettings))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
		tm.app.ExecuteSQL(t, `UPDATE backups_configs_physical SET full_cron_expression = ? WHERE database_id = ?`,
			cronWithoutRunDate, physicalDatabase.Id)

		response := updatePhysicalBackupType(t, tm, physicalDatabase, apiclient.PhysicalBackupTypeFULL)

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)

		databaseResponse, err := tm.owner.Client.GetDatabaseWithResponse(t.Context(), physicalDatabase.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, databaseResponse.StatusCode(), string(databaseResponse.Body))

		storedSettings, err := databaseResponse.JSON200.Settings.AsPostgresqlPhysicalSettings()
		require.NoError(t, err)
		assert.Equal(t, apiclient.PhysicalBackupTypeFULLINCREMENTAL, storedSettings.BackupType)
		assert.Equal(t, 1, countRows(t, tm, `SELECT COUNT(*) FROM backups_configs_physical
			WHERE database_id = ? AND full_cron_expression = ? AND incremental_interval_type = 'HOURLY'`,
			physicalDatabase.Id, cronWithoutRunDate))
	})

	t.Run("refuses a storage ID that no storage has and creates no storage", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, newIncrementalBackupSettings()))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
		unknownStorageID := uuid.New()

		response := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &unknownStorageID, newIncrementalBackupSettings()))

		requireInvalidField(t, response.StatusCode(), response.Body, "storageId")
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM storages WHERE id = ?", unknownStorageID))
		assert.Equal(t, &storageID, getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200.StorageId)
	})

	t.Run("refuses a storage of another workspace and keeps the config's storage", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, newIncrementalBackupSettings()))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
		otherWorkspace := tm.createWorkspace(t)
		otherWorkspaceStorageID := tm.app.CreateLocalStorage(t, tm.owner, otherWorkspace.Id)

		response := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &otherWorkspaceStorageID, newIncrementalBackupSettings()))

		requireInvalidField(t, response.StatusCode(), response.Body, "storageId")
		assert.Equal(t, &storageID, getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200.StorageId)
	})

	t.Run("reads the physical default fitted to a database with incremental backups", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)

		response := getBackupConfig(t, tm.owner, physicalDatabase.Id)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		defaultSettings := getPhysicalSettings(t, *response.JSON200)
		assert.False(t, response.JSON200.IsBackupsEnabled)
		assert.Nil(t, response.JSON200.StorageId)
		assert.Equal(t, newDailySchedule("04:00"), defaultSettings.FullBackupSchedule)
		assert.NotNil(t, defaultSettings.IncrementalBackupSchedule)
		assert.Zero(t, countRows(t, tm,
			"SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", physicalDatabase.Id))
	})

	t.Run("drops the incremental schedule when the database switches to full backups only", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTAL)
		storageID := tm.createLocalStorage(t)
		savedSettings := newIncrementalBackupSettings()
		saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id,
			newPhysicalBackupConfigInput(t, &storageID, savedSettings))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))

		switchPhysicalBackupType(t, tm, physicalDatabase, apiclient.PhysicalBackupTypeFULL)

		switchedConfig := *getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200
		switchedSettings := getPhysicalSettings(t, switchedConfig)
		assert.Nil(t, switchedSettings.IncrementalBackupSchedule)
		assert.Equal(t, savedSettings.FullBackupSchedule, switchedSettings.FullBackupSchedule)
		assert.Equal(t, savedSettings.NotificationTriggers, switchedSettings.NotificationTriggers)
		assert.Equal(t, &storageID, switchedConfig.StorageId)
		assert.True(t, switchedConfig.IsBackupsEnabled)
		assert.Equal(t, 1, countRows(t, tm, `SELECT COUNT(*) FROM backups_configs_physical
			WHERE database_id = ? AND incremental_interval_type = ''`, physicalDatabase.Id))
	})

	t.Run("stops the WAL streamer when the database switches off WAL streaming", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer,
			apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM)
		tm.app.ExecuteSQL(t, "INSERT INTO backups_physical_wal_streamers (database_id, status) VALUES (?, 'RUNNING')",
			physicalDatabase.Id)

		switchPhysicalBackupType(t, tm, physicalDatabase, apiclient.PhysicalBackupTypeFULLINCREMENTAL)

		assert.Zero(t, countRows(t, tm,
			"SELECT COUNT(*) FROM backups_physical_wal_streamers WHERE database_id = ?", physicalDatabase.Id))
	})

	t.Run(
		"reads a config left unfit by an earlier release fitted and stores the fit on the next save",
		func(t *testing.T) {
			tm := newTeam(t)
			physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULL)
			storageID := tm.createLocalStorage(t)
			tm.app.ExecuteSQL(t, `INSERT INTO backups_configs (database_id, method, is_backups_enabled, storage_id)
			VALUES (?, 'PHYSICAL', TRUE, ?)`, physicalDatabase.Id, storageID)
			tm.app.ExecuteSQL(t, `INSERT INTO backups_configs_physical
			(database_id, full_interval_type, full_time_of_day, incremental_interval_type, retention,
			full_backups_retention_policy, full_backups_retention_count)
			VALUES (?, 'DAILY', '04:00', 'HOURLY', 'FULL_BACKUPS', 'LAST_N', 7)`, physicalDatabase.Id)

			readConfig := *getBackupConfig(t, tm.owner, physicalDatabase.Id).JSON200
			listResponse := listBackupConfigs(t, tm.owner, apiclient.ListBackupConfigsParams{
				WorkspaceId: &tm.workspace.Id,
				StorageId:   nil,
				Limit:       nil,
				After:       nil,
			})

			assert.Nil(t, getPhysicalSettings(t, readConfig).IncrementalBackupSchedule)
			require.Equal(t, http.StatusOK, listResponse.StatusCode(), string(listResponse.Body))
			require.Len(t, listResponse.JSON200.Items, 1)
			assert.Nil(t, getPhysicalSettings(t, listResponse.JSON200.Items[0]).IncrementalBackupSchedule)
			assert.Equal(t, 1, countRows(t, tm, `SELECT COUNT(*) FROM backups_configs_physical
			WHERE database_id = ? AND incremental_interval_type = 'HOURLY'`, physicalDatabase.Id))

			saveResponse := updateBackupConfig(t, tm.owner, physicalDatabase.Id, apiclient.BackupConfigInput{
				IsBackupsEnabled: readConfig.IsBackupsEnabled,
				StorageId:        readConfig.StorageId,
				Encryption:       readConfig.Encryption,
				Settings:         readConfig.Settings,
			})

			require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))
			assert.Equal(t, 1, countRows(t, tm, `SELECT COUNT(*) FROM backups_configs_physical
			WHERE database_id = ? AND incremental_interval_type = ''`, physicalDatabase.Id))
		},
	)
}

func newCronSchedule(expression string) apiclient.Schedule {
	return apiclient.Schedule{
		Type:           apiclient.ScheduleTypeCRON,
		TimeOfDay:      nil,
		Weekday:        nil,
		DayOfMonth:     nil,
		CronExpression: &expression,
	}
}

func newDailySchedule(timeOfDay string) apiclient.Schedule {
	return apiclient.Schedule{
		Type:           apiclient.ScheduleTypeDAILY,
		TimeOfDay:      &timeOfDay,
		Weekday:        nil,
		DayOfMonth:     nil,
		CronExpression: nil,
	}
}

// newIncrementalBackupSettings is a valid config of a database with incremental backups: a daily full
// backup with an hourly incremental one.
func newIncrementalBackupSettings() apiclient.PhysicalBackupSettings {
	return apiclient.PhysicalBackupSettings{
		Type:               apiclient.PhysicalBackupSettingsTypePHYSICAL,
		FullBackupSchedule: newDailySchedule("03:15"),
		IncrementalBackupSchedule: &apiclient.Schedule{
			Type:           apiclient.ScheduleTypeHOURLY,
			TimeOfDay:      nil,
			Weekday:        nil,
			DayOfMonth:     nil,
			CronExpression: nil,
		},
		Retention: apiclient.PhysicalRetention{
			Policy:      apiclient.PhysicalRetentionPolicyCHAINS,
			ChainsCount: 4,
			FullBackups: nil,
		},
		NotificationTriggers: []apiclient.NotificationTrigger{
			apiclient.NotificationTriggerBACKUPFAILED,
			apiclient.NotificationTriggerCHAINBROKEN,
		},
		WalLagThresholdBytes: 0,
	}
}

func newPhysicalBackupConfigInput(
	t *testing.T,
	storageID *uuid.UUID,
	physical apiclient.PhysicalBackupSettings,
) apiclient.BackupConfigInput {
	t.Helper()

	return apiclient.BackupConfigInput{
		IsBackupsEnabled: true,
		StorageId:        storageID,
		Encryption:       apiclient.BackupEncryptionNONE,
		Settings:         fromPhysicalBackupSettings(t, physical),
	}
}

func switchPhysicalBackupType(
	t *testing.T,
	tm team,
	physicalDatabase apiclient.Database,
	backupType apiclient.PhysicalBackupType,
) {
	t.Helper()

	response := updatePhysicalBackupType(t, tm, physicalDatabase, backupType)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func updatePhysicalBackupType(
	t *testing.T,
	tm team,
	physicalDatabase apiclient.Database,
	backupType apiclient.PhysicalBackupType,
) *apiclient.UpdateDatabaseResponse {
	t.Helper()

	physicalSettings, err := physicalDatabase.Settings.AsPostgresqlPhysicalSettings()
	require.NoError(t, err)

	physicalSettings.BackupType = backupType

	var settings apiclient.DatabaseSettings
	require.NoError(t, settings.FromPostgresqlPhysicalSettings(physicalSettings))

	response, err := tm.owner.Client.UpdateDatabaseWithResponse(t.Context(), physicalDatabase.Id,
		apiclient.DatabaseUpdate{
			Name:        physicalDatabase.Name,
			NotifierIds: physicalDatabase.NotifierIds,
			Settings:    settings,
		})
	require.NoError(t, err)

	return response
}
