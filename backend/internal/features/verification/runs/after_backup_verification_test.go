package verification_runs

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
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

func Test_CreateScheduledRuns_WhenScheduleTypeAfterBackup_DoesNotCreateTimeBasedRun(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Empty(t, rows, "after-backup config must never get a time-based scheduled run")
}

func Test_OnBackupCompleted_WhenScheduleTypeAfterBackup_EnqueuesPendingRun(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	GetVerificationService().OnBackupCompleted(backup.ID)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, VerificationStatusPending, rows[0].Status)
	assert.Equal(t, VerificationTriggerAfterBackup, rows[0].Trigger)
	assert.Equal(t, backup.ID, rows[0].BackupID)
}

func Test_OnBackupCompleted_WhenPendingAfterBackupExists_CancelsOldAndEnqueuesNew(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	firstBackup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	service := GetVerificationService()
	service.OnBackupCompleted(firstBackup.ID)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	firstRunID := rows[0].ID

	newerBackup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 150)
	service.OnBackupCompleted(newerBackup.ID)

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

func Test_OnBackupCompleted_WhenManualVerificationPending_LeavesItAndStillEnqueuesAfterBackup(
	t *testing.T,
) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	manualRun := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	GetVerificationService().OnBackupCompleted(backup.ID)

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

func Test_OnBackupCompleted_WhenScheduleTypeInterval_DoesNothing(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	GetVerificationService().OnBackupCompleted(backup.ID)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Empty(t, rows, "interval-mode config must ignore backup-completion events")
}

func Test_OnBackupCompleted_WhenDisabled_DoesNothing(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	verification_config.SaveVerificationConfigViaAPI(t, router, owner.Token, database.ID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: false,
			ScheduleType:                   verification_config.VerificationScheduleAfterBackup,
		})

	GetVerificationService().OnBackupCompleted(backup.ID)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Empty(t, rows, "disabled config must not enqueue after-backup verifications")
}

func Test_MakeBackup_WhenAfterBackupConfigured_SchedulesAndReplacesPendingVerification(
	t *testing.T,
) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, testStorage)
	enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)

	// Start the real backups scheduler singleton so it subscribes to the
	// backup-completion channel — the seam the verification listener is wired
	// onto by verification_runs.SetupDependencies (called in createTestRouter).
	stopScheduler := backuping_logical.StartSchedulerForTest(t, backuping_logical.GetBackupsScheduler())
	defer stopScheduler()

	backuper := backuping_logical.CreateTestBackuperWithUseCase(&backuping_logical.CreateSuccessBackupUsecase{})

	// completeBackup reproduces the scheduler's in-process dispatch exactly:
	// MakeBackup drives the backup to COMPLETED, then the completion hook fires
	// the registered listeners — the seam the verification listener is wired onto.
	completeBackup := func() uuid.UUID {
		backup := backuping_logical.SeedInProgressTestBackup(t, database.ID, testStorage.ID)
		backuper.MakeBackup(t.Context(), backup.ID, false)
		backuping_logical.TriggerBackupCompletedForTest(backup.ID)

		return backup.ID
	}

	waitForSinglePending := func(expectedBackupID uuid.UUID) *RestoreVerification {
		t.Helper()

		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			var pending []*RestoreVerification
			for _, row := range ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID) {
				if row.Status == VerificationStatusPending {
					pending = append(pending, row)
				}
			}

			if len(pending) == 1 && pending[0].BackupID == expectedBackupID {
				return pending[0]
			}

			time.Sleep(100 * time.Millisecond)
		}

		t.Fatalf("timed out waiting for a single PENDING after-backup verification on backup %s", expectedBackupID)
		return nil
	}

	// First completed backup → exactly one PENDING after-backup verification.
	firstBackupID := completeBackup()
	firstRun := waitForSinglePending(firstBackupID)
	assert.Equal(t, VerificationTriggerAfterBackup, firstRun.Trigger)

	// Second completed backup while the first run is still PENDING → the stale
	// PENDING run is canceled and a fresh one is scheduled for the newer backup.
	secondBackupID := completeBackup()
	secondRun := waitForSinglePending(secondBackupID)

	// Terminalize the remaining PENDING run even if an assertion below aborts —
	// FindOldestPendingClaimablesTop100 spans the shared test DB, so a leaked
	// PENDING run could be claimed by a later test's agent.
	defer CancelVerificationViaAPI(t, router, owner.Token, secondRun.ID)

	assert.NotEqual(t, firstRun.ID, secondRun.ID)
	assert.Equal(t, VerificationTriggerAfterBackup, secondRun.Trigger)

	firstFinal := GetVerificationByIDViaAPI(t, router, owner.Token, firstRun.ID)
	assert.Equal(
		t,
		VerificationStatusCanceled,
		firstFinal.Status,
		"a newer completed backup must cancel the still-PENDING after-backup run",
	)
}
