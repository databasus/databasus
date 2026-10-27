package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/util/testing/containers"
)

func TestSavePhysicalBackupConfig(t *testing.T) {
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
		tm.app.ExecuteSQL(t, "INSERT INTO physical_wal_streamers (database_id, status) VALUES (?, 'RUNNING')",
			physicalDatabase.Id)

		switchPhysicalBackupType(t, tm, physicalDatabase, apiclient.PhysicalBackupTypeFULLINCREMENTAL)

		assert.Zero(t, countRows(t, tm,
			"SELECT COUNT(*) FROM physical_wal_streamers WHERE database_id = ?", physicalDatabase.Id))
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
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}
