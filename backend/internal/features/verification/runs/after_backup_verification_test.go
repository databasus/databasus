package verification_runs

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/logger"
)

func enableAfterBackupVerificationViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	databaseID uuid.UUID,
) {
	t.Helper()

	verification_config.SaveVerificationConfigViaAPI(t, router, userToken, databaseID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: true,
			ScheduleType:                   verification_config.VerificationScheduleAfterBackup,
		})
}

func publishLogicalBackupCompleted(t *testing.T, completedBackup backups.LogicalBackup) {
	t.Helper()

	require.NoError(t, GetVerificationService().OnLogicalBackupCompleted(
		t.Context(),
		backups.LogicalBackupCompleted{BackupID: completedBackup.ID, DatabaseID: completedBackup.DatabaseID},
	))
}

func Test_CreateScheduledRuns_WhenScheduleTypeAfterBackup_DoesNotCreateTimeBasedRun(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	defer notifiers_bridge.RemoveTestNotifier(notifier)

	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), database)

	backups_bridge.SeedTestLogicalBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Empty(t, rows, "after-backup config must never get a time-based scheduled run")
}

func Test_OnLogicalBackupCompleted_WhenScheduleTypeAfterBackup_EnqueuesPendingRun(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	defer notifiers_bridge.RemoveTestNotifier(notifier)

	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), database)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	publishLogicalBackupCompleted(t, backup)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, VerificationStatusPending, rows[0].Status)
	assert.Equal(t, VerificationTriggerAfterBackup, rows[0].Trigger)
	assert.Equal(t, backup.ID, rows[0].BackupID)
}

func Test_OnLogicalBackupCompleted_WhenPendingAfterBackupExists_CancelsOldAndEnqueuesNew(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	defer notifiers_bridge.RemoveTestNotifier(notifier)

	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), database)

	firstBackup := backups_bridge.SeedTestLogicalBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	publishLogicalBackupCompleted(t, firstBackup)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	firstRunID := rows[0].ID

	newerBackup := backups_bridge.SeedTestLogicalBackup(t, database.ID, testStorage.ID, 150)
	publishLogicalBackupCompleted(t, newerBackup)

	rowsAfter := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rowsAfter, 2)

	var pendingCount int
	for _, row := range rowsAfter {
		if row.ID == firstRunID {
			assert.Equal(t, VerificationStatusCanceled, row.Status, "old PENDING must be canceled")
			require.NotNil(t, row.FailMessage)
			assert.Contains(t, *row.FailMessage, "superseded")
			continue
		}

		pendingCount++
		assert.Equal(t, VerificationStatusPending, row.Status)
		assert.Equal(t, VerificationTriggerAfterBackup, row.Trigger)
		assert.Equal(t, newerBackup.ID, row.BackupID, "fresh run must point at the newest backup")
	}
	assert.Equal(t, 1, pendingCount, "queue must stay bounded at one PENDING after-backup run")
}

func Test_OnLogicalBackupCompleted_WhenManualVerificationPending_LeavesItAndStillEnqueuesAfterBackup(
	t *testing.T,
) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	defer notifiers_bridge.RemoveTestNotifier(notifier)

	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), database)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	manualRun := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	publishLogicalBackupCompleted(t, backup)

	manualAfter := GetVerificationByIDViaAPI(t, router, owner.Token, manualRun.ID)
	assert.Equal(
		t,
		VerificationStatusPending,
		manualAfter.Status,
		"a user-triggered MANUAL run must survive a backup completion",
	)
	assert.Equal(t, VerificationTriggerManual, manualAfter.Trigger)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 2, "manual run plus the new after-backup run")

	var afterBackupPending int
	for _, row := range rows {
		if row.Trigger == VerificationTriggerAfterBackup && row.Status == VerificationStatusPending {
			afterBackupPending++
			assert.Equal(t, backup.ID, row.BackupID)
		}
	}
	assert.Equal(t, 1, afterBackupPending, "after-backup run still enqueued alongside the manual one")
}

func Test_OnLogicalBackupCompleted_WhenScheduleTypeInterval_DoesNothing(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	defer notifiers_bridge.RemoveTestNotifier(notifier)

	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), database)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, testStorage.ID, 100)

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	publishLogicalBackupCompleted(t, backup)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Empty(t, rows, "interval-mode config must ignore backup-completion events")
}

func Test_OnLogicalBackupCompleted_WhenDisabled_DoesNothing(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	defer notifiers_bridge.RemoveTestNotifier(notifier)

	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), database)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, testStorage.ID, 100)

	verification_config.SaveVerificationConfigViaAPI(t, router, owner.Token, database.ID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: false,
			ScheduleType:                   verification_config.VerificationScheduleAfterBackup,
		})

	publishLogicalBackupCompleted(t, backup)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Empty(t, rows, "disabled config must not enqueue after-backup verifications")
}
