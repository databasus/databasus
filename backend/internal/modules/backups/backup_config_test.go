package backups_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestReadBackupConfig(t *testing.T) {
	t.Parallel()

	t.Run(
		"reads the logical default of a database whose config was never saved and stores nothing",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)

			response := getBackupConfig(t, tm.owner, databaseID)

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			defaultConfig := *response.JSON200
			assert.Equal(t, databaseID, defaultConfig.DatabaseId)
			assert.False(t, defaultConfig.IsBackupsEnabled)
			assert.Nil(t, defaultConfig.StorageId)
			assert.Equal(t, apiclient.BackupEncryptionNONE, defaultConfig.Encryption)

			defaultTimeOfDay := "04:00"
			assert.Equal(t, apiclient.LogicalBackupSettings{
				Type: apiclient.LogicalBackupSettingsTypeLOGICAL,
				Schedule: apiclient.Schedule{
					Type:           apiclient.ScheduleTypeDAILY,
					TimeOfDay:      &defaultTimeOfDay,
					Weekday:        nil,
					DayOfMonth:     nil,
					CronExpression: nil,
				},
				Retention: apiclient.LogicalRetention{
					Policy:    apiclient.LogicalRetentionPolicyTIMEPERIOD,
					Period:    apiclient.RetentionPeriodTHREEMONTHS,
					Count:     0,
					GfsHours:  0,
					GfsDays:   0,
					GfsWeeks:  0,
					GfsMonths: 0,
					GfsYears:  0,
				},
				NotificationTriggers: []apiclient.LogicalNotificationTrigger{
					apiclient.LogicalNotificationTriggerBACKUPFAILED,
					apiclient.LogicalNotificationTriggerBACKUPSUCCESS,
				},
				IsRetryIfFailed:     true,
				MaxFailedTriesCount: 3,
			}, getLogicalSettings(t, defaultConfig))
			assert.Empty(t, listWorkspaceConfigDatabaseIDs(t, tm.owner, tm.workspace.Id))
			assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", databaseID))
		},
	)

	t.Run("reads the saved values after the first save and lists the config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)

		savedConfig := saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

		response := getBackupConfig(t, tm.owner, databaseID)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.True(t, response.JSON200.IsBackupsEnabled)
		assert.Equal(t, &storageID, response.JSON200.StorageId)
		assert.Equal(t, newLogicalBackupSettings(), getLogicalSettings(t, *response.JSON200))
		assert.Equal(t, getLogicalSettings(t, savedConfig), getLogicalSettings(t, *response.JSON200))
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceConfigDatabaseIDs(t, tm.owner, tm.workspace.Id))
	})
}

func TestSaveBackupConfig(t *testing.T) {
	t.Parallel()

	t.Run("refuses a physical config for a logical database and keeps the config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		timeOfDay := "04:00"

		response := updateBackupConfig(t, tm.owner, databaseID, apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &storageID,
			Encryption:       apiclient.BackupEncryptionNONE,
			Settings: fromPhysicalBackupSettings(t, apiclient.PhysicalBackupSettings{
				Type: apiclient.PhysicalBackupSettingsTypePHYSICAL,
				FullBackupSchedule: apiclient.Schedule{
					Type:           apiclient.ScheduleTypeDAILY,
					TimeOfDay:      &timeOfDay,
					Weekday:        nil,
					DayOfMonth:     nil,
					CronExpression: nil,
				},
				IncrementalBackupSchedule: nil,
				Retention: apiclient.PhysicalRetention{
					Policy:      apiclient.PhysicalRetentionPolicyFULLBACKUPS,
					ChainsCount: 0,
					FullBackups: &apiclient.FullBackupsRetention{
						Policy:    apiclient.FullBackupsRetentionPolicyLASTN,
						Count:     7,
						GfsHours:  0,
						GfsDays:   0,
						GfsWeeks:  0,
						GfsMonths: 0,
						GfsYears:  0,
					},
				},
				NotificationTriggers: []apiclient.NotificationTrigger{},
				WalLagThresholdBytes: 0,
			}),
		})

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
		storedConfig := getBackupConfig(t, tm.owner, databaseID).JSON200
		assert.Equal(t, "LOGICAL", getSettingsType(t, *storedConfig))
		assert.Equal(t, newLogicalBackupSettings(), getLogicalSettings(t, *storedConfig))
	})

	t.Run("refuses a retention count of zero, names the field and keeps the config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		invalidSettings := newLogicalBackupSettings()
		invalidSettings.Retention.Count = 0

		response := updateBackupConfig(t, tm.owner, databaseID,
			newLogicalBackupConfigInput(t, &storageID, invalidSettings))

		requireInvalidField(t, response.StatusCode(), response.Body, "settings.retention.count")
		assert.Equal(
			t,
			newLogicalBackupSettings(),
			getLogicalSettings(t, *getBackupConfig(t, tm.owner, databaseID).JSON200),
		)
	})

	for _, undatedCron := range []string{"0 0 31 2 *", "0 0 31 2,4 *"} {
		t.Run("refuses a cron schedule with no date to run on and keeps the config with "+undatedCron,
			func(t *testing.T) {
				t.Parallel()

				tm := newTeam(t)
				databaseID := tm.insertLogicalDatabase(t)
				storageID := tm.createLocalStorage(t)
				saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
				undatedSettings := newLogicalBackupSettings()
				undatedSettings.Schedule = newCronSchedule(undatedCron)

				response := updateBackupConfig(t, tm.owner, databaseID,
					newLogicalBackupConfigInput(t, &storageID, undatedSettings))

				requireInvalidField(t, response.StatusCode(), response.Body, "settings.schedule.cronExpression")
				assert.Equal(
					t,
					newLogicalBackupSettings(),
					getLogicalSettings(t, *getBackupConfig(t, tm.owner, databaseID).JSON200),
				)
			})
	}

	t.Run("saves a cron schedule with a date only in leap years", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		leapDaySettings := newLogicalBackupSettings()
		leapDaySettings.Schedule = newCronSchedule("0 0 29 2 *")

		response := updateBackupConfig(t, tm.owner, databaseID,
			newLogicalBackupConfigInput(t, &storageID, leapDaySettings))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(
			t,
			leapDaySettings,
			getLogicalSettings(t, *getBackupConfig(t, tm.owner, databaseID).JSON200),
		)
	})

	t.Run("names a storage of the database's workspace and leaves the storage as it was", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		storageBeforeSave := getStorage(t, tm.owner, storageID)

		savedConfig := saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

		assert.Equal(t, &storageID, savedConfig.StorageId)
		assert.Equal(t, storageBeforeSave, getStorage(t, tm.owner, storageID))
	})

	t.Run("ignores storage fields beside the storage ID and leaves the storage as it was", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		storageBeforeSave := getStorage(t, tm.owner, storageID)
		requestBody := newLogicalBackupConfigBody(t, &storageID)
		requestBody["storage"] = map[string]any{
			"id":        storageID,
			"name":      "Renamed",
			"type":      "S3",
			"accessKey": "changed",
		}

		response := updateBackupConfigWithBody(t, tm.owner, databaseID, requestBody)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, &storageID, response.JSON200.StorageId)
		assert.Equal(t, storageBeforeSave, getStorage(t, tm.owner, storageID))
	})

	t.Run("names no storage and creates none when the body carries a storage object without an ID", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		requestBody := newLogicalBackupConfigBody(t, nil)
		requestBody["storage"] = map[string]any{"name": "Smuggled", "type": "LOCAL"}

		response := updateBackupConfigWithBody(t, tm.owner, databaseID, requestBody)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Nil(t, response.JSON200.StorageId)
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM storages WHERE workspace_id = ?", tm.workspace.Id))
	})

	t.Run("refuses a storage ID that no storage has and creates no storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		unknownStorageID := uuid.New()

		response := updateBackupConfig(t, tm.owner, databaseID,
			newLogicalBackupConfigInput(t, &unknownStorageID, newLogicalBackupSettings()))

		requireInvalidField(t, response.StatusCode(), response.Body, "storageId")
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM storages WHERE id = ?", unknownStorageID))
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", databaseID))
	})

	t.Run("refuses a storage of another workspace and keeps the config's storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		otherWorkspace := tm.createWorkspace(t)
		otherWorkspaceStorageID := tm.app.CreateLocalStorage(t, tm.owner, otherWorkspace.Id)

		response := updateBackupConfig(t, tm.owner, databaseID,
			newLogicalBackupConfigInput(t, &otherWorkspaceStorageID, newLogicalBackupSettings()))

		requireInvalidField(t, response.StatusCode(), response.Body, "storageId")
		assert.Equal(t, &storageID, getBackupConfig(t, tm.owner, databaseID).JSON200.StorageId)
	})
}

func TestSaveBackupConfigRacingStorageTransfer(t *testing.T) {
	t.Parallel()

	t.Run("lets exactly one of a save and a transfer of its storage succeed", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		targetWorkspace := tm.createWorkspace(t)
		databaseID := tm.insertLogicalDatabase(t)
		previousStorageID := tm.createLocalStorage(t)

		const roundCount = 8

		for range roundCount {
			saveLogicalBackupConfig(t, tm.owner, databaseID, previousStorageID)
			racedStorageID := tm.createLocalStorage(t)

			saveStatus := make(chan int, 1)
			transferStatus := make(chan int, 1)
			startSignal := make(chan struct{})

			go func() {
				<-startSignal
				saveStatus <- saveConfigWithStorageFromGoroutine(t, tm.owner, databaseID, racedStorageID)
			}()

			go func() {
				<-startSignal
				transferStatus <- transferStorageFromGoroutine(t, tm.owner, racedStorageID, targetWorkspace.Id)
			}()

			close(startSignal)

			roundSaveStatus, roundTransferStatus := <-saveStatus, <-transferStatus
			storedStorageID := getBackupConfig(t, tm.owner, databaseID).JSON200.StorageId
			storageWorkspaceID := getStorage(t, tm.owner, racedStorageID).WorkspaceId

			if roundSaveStatus == http.StatusOK {
				assert.Equal(t, http.StatusConflict, roundTransferStatus)
				assert.Equal(t, &racedStorageID, storedStorageID)
				assert.Equal(t, tm.workspace.Id, storageWorkspaceID)
			} else {
				assert.Equal(t, http.StatusBadRequest, roundSaveStatus)
				assert.Equal(t, http.StatusNoContent, roundTransferStatus)
				assert.Equal(t, &previousStorageID, storedStorageID)
				assert.Equal(t, targetWorkspace.Id, storageWorkspaceID)
			}
		}
	})
}

func TestListBackupConfigs(t *testing.T) {
	t.Parallel()

	t.Run("lists the saved configs of a workspace in one response with their storages", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		storageID := tm.createLocalStorage(t)
		savedDatabaseIDs := []uuid.UUID{
			tm.insertLogicalDatabase(t), tm.insertLogicalDatabase(t), tm.insertLogicalDatabase(t),
		}

		for _, databaseID := range savedDatabaseIDs {
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		}

		response := listBackupConfigs(t, tm.owner, apiclient.ListBackupConfigsParams{
			WorkspaceId: &tm.workspace.Id,
			StorageId:   nil,
			Limit:       nil,
			After:       nil,
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		listedDatabaseIDs := make([]uuid.UUID, 0, len(response.JSON200.Items))
		for _, listedConfig := range response.JSON200.Items {
			listedDatabaseIDs = append(listedDatabaseIDs, listedConfig.DatabaseId)
			assert.Equal(t, &storageID, listedConfig.StorageId)
		}

		assert.ElementsMatch(t, savedDatabaseIDs, listedDatabaseIDs)
		assert.Nil(t, response.JSON200.NextCursor)
	})

	t.Run("pages the configs of a workspace in database ID order", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		storageID := tm.createLocalStorage(t)
		savedDatabaseIDs := []uuid.UUID{
			tm.insertLogicalDatabase(t), tm.insertLogicalDatabase(t), tm.insertLogicalDatabase(t),
		}

		for _, databaseID := range savedDatabaseIDs {
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		}

		pageLimit := 2
		listedDatabaseIDs := make([]uuid.UUID, 0, len(savedDatabaseIDs))

		var nextCursor *string

		for {
			response := listBackupConfigs(t, tm.owner, apiclient.ListBackupConfigsParams{
				WorkspaceId: &tm.workspace.Id,
				StorageId:   nil,
				Limit:       &pageLimit,
				After:       nextCursor,
			})
			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

			for _, listedConfig := range response.JSON200.Items {
				listedDatabaseIDs = append(listedDatabaseIDs, listedConfig.DatabaseId)
			}

			nextCursor = response.JSON200.NextCursor
			if nextCursor == nil {
				break
			}
		}

		assert.True(t, isSortedByID(listedDatabaseIDs), listedDatabaseIDs)
		assert.ElementsMatch(t, savedDatabaseIDs, listedDatabaseIDs)
	})

	t.Run("lists the config that backs up into a storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		usedStorageID := tm.createLocalStorage(t)
		otherStorageID := tm.createLocalStorage(t)
		usingDatabaseID := tm.insertLogicalDatabase(t)
		saveLogicalBackupConfig(t, tm.owner, usingDatabaseID, usedStorageID)
		saveLogicalBackupConfig(t, tm.owner, tm.insertLogicalDatabase(t), otherStorageID)
		pageLimit := 1

		response := listBackupConfigs(t, tm.owner, apiclient.ListBackupConfigsParams{
			WorkspaceId: nil,
			StorageId:   &usedStorageID,
			Limit:       &pageLimit,
			After:       nil,
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		require.Len(t, response.JSON200.Items, 1)
		assert.Equal(t, usingDatabaseID, response.JSON200.Items[0].DatabaseId)
	})

	t.Run("refuses a list with both filters or with neither", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		storageID := tm.createLocalStorage(t)

		bothFiltersResponse := listBackupConfigs(t, tm.owner, apiclient.ListBackupConfigsParams{
			WorkspaceId: &tm.workspace.Id,
			StorageId:   &storageID,
			Limit:       nil,
			After:       nil,
		})
		noFilterResponse := listBackupConfigs(t, tm.owner, apiclient.ListBackupConfigsParams{
			WorkspaceId: nil,
			StorageId:   nil,
			Limit:       nil,
			After:       nil,
		})

		requireProblem(t, bothFiltersResponse.StatusCode(), bothFiltersResponse.Body,
			http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
		requireProblem(t, noFilterResponse.StatusCode(), noFilterResponse.Body,
			http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
	})
}

func TestCopyDatabaseBackupConfig(t *testing.T) {
	t.Parallel()

	t.Run("gives the copy of a database the copy of its saved logical config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		hourlySettings := newLogicalBackupSettings()
		hourlySettings.Schedule = apiclient.Schedule{
			Type:           apiclient.ScheduleTypeHOURLY,
			TimeOfDay:      nil,
			Weekday:        nil,
			DayOfMonth:     nil,
			CronExpression: nil,
		}
		saveResponse := updateBackupConfig(t, tm.owner, databaseID,
			newLogicalBackupConfigInput(t, &storageID, hourlySettings))
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))

		copyResponse, err := tm.owner.Client.CopyDatabaseWithResponse(t.Context(), databaseID)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, copyResponse.StatusCode(), string(copyResponse.Body))

		copiedConfig := getBackupConfig(t, tm.owner, copyResponse.JSON201.Id).JSON200
		assert.True(t, copiedConfig.IsBackupsEnabled)
		assert.Equal(t, &storageID, copiedConfig.StorageId)
		assert.Equal(t, hourlySettings, getLogicalSettings(t, *copiedConfig))
	})
}

func TestDeleteDatabaseBackupConfig(t *testing.T) {
	t.Parallel()

	t.Run("deletes the saved config with the database", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, tm.createLocalStorage(t))

		response, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)
		require.NoError(t, err)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listWorkspaceConfigDatabaseIDs(t, tm.owner, tm.workspace.Id))
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", databaseID))
	})

	t.Run("keeps the config when the deletion is refused while a logical backup runs", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
			VALUES (?, ?, ?, 'IN_PROGRESS', ?)`, uuid.New(), databaseID, storageID, "running-"+uuid.NewString())

		response, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)
		require.NoError(t, err)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceConfigDatabaseIDs(t, tm.owner, tm.workspace.Id))
	})
}

func TestBackupConfigAccess(t *testing.T) {
	t.Parallel()

	t.Run("lets a viewer list the workspace's configs and read a database's config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		databaseID := tm.insertLogicalDatabase(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, tm.createLocalStorage(t))

		readResponse := getBackupConfig(t, viewer, databaseID)

		require.Equal(t, http.StatusOK, readResponse.StatusCode(), string(readResponse.Body))
		assert.Equal(t, []uuid.UUID{databaseID}, listWorkspaceConfigDatabaseIDs(t, viewer, tm.workspace.Id))
	})

	t.Run("lets a member save a config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		databaseID := tm.insertLogicalDatabase(t)

		savedConfig := saveLogicalBackupConfig(t, member, databaseID, tm.createLocalStorage(t))

		assert.True(t, savedConfig.IsBackupsEnabled)
	})

	t.Run("refuses a viewer's save with 403 and keeps the config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		viewerSettings := newLogicalBackupSettings()
		viewerSettings.Retention.Count = 9

		response := updateBackupConfig(
			t,
			viewer,
			databaseID,
			newLogicalBackupConfigInput(t, &storageID, viewerSettings),
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", databaseID))
	})

	t.Run("answers a user outside the workspace with 404 on every operation and keeps the config", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.SignUp(t, apptest.NewTestEmail("stranger"))
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
		strangerSettings := newLogicalBackupSettings()
		strangerSettings.Retention.Count = 9

		readResponse := getBackupConfig(t, stranger, databaseID)
		saveResponse := updateBackupConfig(t, stranger, databaseID,
			newLogicalBackupConfigInput(t, &storageID, strangerSettings))
		workspaceListResponse := listBackupConfigs(t, stranger, apiclient.ListBackupConfigsParams{
			WorkspaceId: &tm.workspace.Id,
			StorageId:   nil,
			Limit:       nil,
			After:       nil,
		})
		storageListResponse := listBackupConfigs(t, stranger, apiclient.ListBackupConfigsParams{
			WorkspaceId: nil,
			StorageId:   &storageID,
			Limit:       nil,
			After:       nil,
		})

		requireProblem(t, readResponse.StatusCode(), readResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, saveResponse.StatusCode(), saveResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, workspaceListResponse.StatusCode(), workspaceListResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, storageListResponse.StatusCode(), storageListResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		assert.Equal(
			t,
			newLogicalBackupSettings(),
			getLogicalSettings(t, *getBackupConfig(t, tm.owner, databaseID).JSON200),
		)
	})

	t.Run("answers a missing database like a database of another workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.SignUp(t, apptest.NewTestEmail("stranger"))
		databaseID := tm.insertLogicalDatabase(t)

		strangerResponse := getBackupConfig(t, stranger, databaseID)
		missingResponse := getBackupConfig(t, tm.owner, uuid.New())

		requireProblem(t, strangerResponse.StatusCode(), strangerResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		assert.JSONEq(t, string(strangerResponse.Body), string(missingResponse.Body))
	})
}

func getStorage(t *testing.T, session apptest.Session, storageID uuid.UUID) apiclient.Storage {
	t.Helper()

	response, err := session.Client.GetStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func newLogicalBackupConfigBody(t *testing.T, storageID *uuid.UUID) map[string]any {
	t.Helper()

	encodedInput, err := json.Marshal(newLogicalBackupConfigInput(t, storageID, newLogicalBackupSettings()))
	require.NoError(t, err)

	var requestBody map[string]any
	require.NoError(t, json.Unmarshal(encodedInput, &requestBody))

	return requestBody
}

func updateBackupConfigWithBody(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
	requestBody map[string]any,
) *apiclient.UpdateBackupConfigResponse {
	t.Helper()

	encodedBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	response, err := session.Client.UpdateBackupConfigWithBodyWithResponse(t.Context(), databaseID,
		"application/json", bytes.NewReader(encodedBody))
	require.NoError(t, err)

	return response
}

// saveConfigWithStorageFromGoroutine reports a broken request as status 0 instead of failing the test,
// because it runs outside the test's goroutine.
func saveConfigWithStorageFromGoroutine(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) int {
	t.Helper()

	var settings apiclient.BackupSettings
	if err := settings.FromLogicalBackupSettings(newLogicalBackupSettings()); err != nil {
		return 0
	}

	response, err := session.Client.UpdateBackupConfigWithResponse(t.Context(), databaseID, apiclient.BackupConfigInput{
		IsBackupsEnabled: true,
		StorageId:        &storageID,
		Encryption:       apiclient.BackupEncryptionNONE,
		Settings:         settings,
	})
	if err != nil {
		return 0
	}

	return response.StatusCode()
}

func transferStorageFromGoroutine(
	t *testing.T,
	session apptest.Session,
	storageID uuid.UUID,
	targetWorkspaceID uuid.UUID,
) int {
	t.Helper()

	response, err := session.Client.TransferStorageWithResponse(t.Context(), storageID, apiclient.StorageTransfer{
		TargetWorkspaceId: targetWorkspaceID,
	})
	if err != nil {
		return 0
	}

	return response.StatusCode()
}

func isSortedByID(databaseIDs []uuid.UUID) bool {
	for index := 1; index < len(databaseIDs); index++ {
		if bytes.Compare(databaseIDs[index-1][:], databaseIDs[index][:]) > 0 {
			return false
		}
	}

	return true
}
