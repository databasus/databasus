package verification_runs

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
	test_utils "databasus-backend/internal/util/testing"
)

func shrinkThresholds(t *testing.T, pendingMax, staleAgent time.Duration) {
	t.Helper()

	originalPending := maxPendingDuration
	originalStale := StaleAgentThreshold

	maxPendingDuration = pendingMax
	StaleAgentThreshold = staleAgent

	t.Cleanup(func() {
		maxPendingDuration = originalPending
		StaleAgentThreshold = originalStale
	})
}

func enableHourlyVerificationViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	databaseID uuid.UUID,
) {
	t.Helper()

	verification_config.SaveVerificationConfigViaAPI(t, router, userToken, databaseID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: true,
			ScheduleType:                   verification_config.VerificationScheduleInterval,
			VerificationInterval:           intervals.Interval{Type: intervals.IntervalHourly},
		})
}

func Test_CreateScheduledRuns_WhenMultipleBackupsExist_PicksLatestForVerification(t *testing.T) {
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

	older := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 50)
	older.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	require.NoError(t, storage.GetDb().Save(older).Error)

	latestBackup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, VerificationStatusPending, rows[0].Status)
	assert.Equal(t, VerificationTriggerScheduled, rows[0].Trigger)
	assert.Equal(t, latestBackup.ID, rows[0].BackupID)
}

func Test_CreateScheduledRuns_WhenScheduledPendingAlreadyExists_DoesNotInsertNewRun(t *testing.T) {
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

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Len(t, rows, 1, "second tick must not insert a second row while a SCHEDULED PENDING exists")
}

func Test_CreateScheduledRuns_WhenManualPendingExists_StillCreatesScheduledRun(t *testing.T) {
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

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 2, "manual non-terminal row must not block scheduled creation")

	var manualCount, scheduledCount int
	for _, row := range rows {
		switch row.Trigger {
		case VerificationTriggerManual:
			manualCount++
			assert.Equal(t, VerificationStatusPending, row.Status, "manual row must remain untouched")
		case VerificationTriggerScheduled:
			scheduledCount++
			assert.Equal(t, VerificationStatusPending, row.Status)
		}
	}
	assert.Equal(t, 1, manualCount)
	assert.Equal(t, 1, scheduledCount)
}

func Test_ReapStaleRuns_WhenPendingExceedsMaxDuration_MarksFailedWithoutRetry(t *testing.T) {
	shrinkThresholds(t, 5*time.Millisecond, 5*time.Second)

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

	stalePending := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", stalePending.ID).
		Update("created_at", time.Now().UTC().Add(-1*time.Hour)).Error)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.reapStaleRuns(t.Context(), logger.GetLogger()))

	final := GetVerificationByIDViaAPI(t, router, owner.Token, stalePending.ID)
	assert.Equal(t, VerificationStatusFailed, final.Status)
	require.NotNil(t, final.FailMessage)
	assert.Contains(t, *final.FailMessage, "not picked up")
}

func Test_ReapStaleRuns_WhenOwningAgentRemoved_RequeuesThenRetiresIfUnclaimed(t *testing.T) {
	shrinkThresholds(t, 5*time.Millisecond, 1*time.Millisecond)

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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "soft-del-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	test_utils.MakeDeleteRequest(
		t, router,
		"/api/v1/verification/agents/"+agent.Agent.ID.String(),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.reapStaleRuns(t.Context(), logger.GetLogger()))

	requeued := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(
		t,
		VerificationStatusPending,
		requeued.Status,
		"owning agent gone → requeue so another agent can claim it, not terminal",
	)
	assert.Equal(t, 2, requeued.AttemptCount)
	assert.Nil(t, requeued.AgentID)
	assert.Nil(t, requeued.FailMessage)

	// No agent ever claims it: backdating created_at past the (shrunk)
	// maxPendingDuration drives the stale-PENDING reaper, which retires it.
	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", assignment.VerificationID).
		Update("created_at", time.Now().UTC().Add(-1*time.Hour)).Error)
	require.NoError(t, scheduler.reapStaleRuns(t.Context(), logger.GetLogger()))

	final := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusFailed, final.Status)
	require.NotNil(t, final.FailMessage)
	assert.Contains(t, *final.FailMessage, "not picked up")
}

func Test_ReapStaleRuns_WhenAgentLostContactWithRetriesRemaining_RequeuesVerification(t *testing.T) {
	shrinkThresholds(t, 1*time.Hour, 1*time.Millisecond)

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

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	agent := verification_agents.CreateTestVerificationAgent(
		t,
		router,
		owner.Token,
		"silent-retry-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	require.NoError(t, storage.GetDb().
		Model(&verification_agents.Agent{}).
		Where("id = ?", agent.Agent.ID).
		Update("last_seen_at", time.Now().UTC().Add(-1*time.Hour)).Error)

	require.NoError(t, GetVerificationScheduler().reapStaleRuns(t.Context(), logger.GetLogger()))

	requeued := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusPending, requeued.Status, "must requeue, not go terminal — retries left")
	assert.Equal(t, 2, requeued.AttemptCount)
	assert.Nil(t, requeued.AgentID)
	assert.Nil(t, requeued.FailMessage)
}

func Test_ReapStaleRuns_WhenPendingExceedsMaxDuration_MarksTerminalRegardlessOfPolicy(t *testing.T) {
	shrinkThresholds(t, 5*time.Millisecond, 5*time.Second)

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

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	stalePending := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", stalePending.ID).
		Update("created_at", time.Now().UTC().Add(-1*time.Hour)).Error)

	require.NoError(t, GetVerificationScheduler().reapStaleRuns(t.Context(), logger.GetLogger()))

	final := GetVerificationByIDViaAPI(t, router, owner.Token, stalePending.ID)
	assert.Equal(t, VerificationStatusFailed, final.Status,
		"unclaimed-too-long always goes terminal — agent-side retry policy does not apply to fleet-side failures")
	assert.Equal(t, 1, final.AttemptCount, "no retry → attempt counter must NOT advance")
	require.NotNil(t, final.FailMessage)
	assert.Contains(t, *final.FailMessage, "not picked up")
}

func Test_CancelAutomaticVerificationsForDisabledSchedules_WhenScheduledVerificationDisabled_CancelsNonTerminalRows(
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

	backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	scheduledRowID := rows[0].ID

	verification_config.SaveVerificationConfigViaAPI(t, router, owner.Token, database.ID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: false,
			ScheduleType:                   verification_config.VerificationScheduleInterval,
			VerificationInterval:           intervals.Interval{Type: intervals.IntervalHourly},
		})

	require.NoError(t, scheduler.cancelAutomaticVerificationsForDisabledSchedules(t.Context(), logger.GetLogger()))

	final := GetVerificationByIDViaAPI(t, router, owner.Token, scheduledRowID)
	assert.Equal(t, VerificationStatusCanceled, final.Status)
	require.NotNil(t, final.FailMessage)
	assert.Contains(t, *final.FailMessage, "schedule was disabled")
}

func Test_CancelAutomaticVerificationsForDisabledSchedules_WhenManualVerificationExists_LeavesItPending(
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

	verification_config.SaveVerificationConfigViaAPI(t, router, owner.Token, database.ID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: false,
			ScheduleType:                   verification_config.VerificationScheduleInterval,
			VerificationInterval:           intervals.Interval{Type: intervals.IntervalHourly},
		})

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	manualVerification := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	require.NoError(
		t,
		GetVerificationScheduler().cancelAutomaticVerificationsForDisabledSchedules(t.Context(), logger.GetLogger()),
	)

	verificationAfterSweep := GetVerificationByIDViaAPI(t, router, owner.Token, manualVerification.ID)
	assert.Equal(t, VerificationStatusPending, verificationAfterSweep.Status)
	assert.Nil(t, verificationAfterSweep.FailMessage)
}

func Test_CancelAutomaticVerificationsForDisabledSchedules_WhenManualVerificationIsRunning_LeavesItRunning(
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

	verification_config.SaveVerificationConfigViaAPI(t, router, owner.Token, database.ID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: false,
			ScheduleType:                   verification_config.VerificationScheduleInterval,
			VerificationInterval:           intervals.Interval{Type: intervals.IntervalHourly},
		})

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	manualVerification := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	agent := verification_agents.CreateTestVerificationAgent(
		t, router, owner.Token, "manual-running-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 1},
	)
	require.Equal(t, manualVerification.ID, assignment.VerificationID)

	require.NoError(
		t,
		GetVerificationScheduler().cancelAutomaticVerificationsForDisabledSchedules(t.Context(), logger.GetLogger()),
	)

	verificationAfterSweep := GetVerificationByIDViaAPI(t, router, owner.Token, manualVerification.ID)
	assert.Equal(t, VerificationStatusRunning, verificationAfterSweep.Status)
	assert.Nil(t, verificationAfterSweep.FailMessage)
}

func Test_CancelAutomaticVerificationsForDisabledSchedules_WhenAutomaticVerificationExists_CancelsIt(
	t *testing.T,
) {
	automaticVerificationCases := []struct {
		scenario  string
		trigger   VerificationTrigger
		isRunning bool
	}{
		{"SCHEDULED PENDING", VerificationTriggerScheduled, false},
		{"SCHEDULED RUNNING", VerificationTriggerScheduled, true},
		{"AFTER_BACKUP PENDING", VerificationTriggerAfterBackup, false},
		{"AFTER_BACKUP RUNNING", VerificationTriggerAfterBackup, true},
	}

	for _, verificationCase := range automaticVerificationCases {
		t.Run(verificationCase.scenario, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
			workspace := workspaces_testing.CreateTestWorkspace(
				t.Context(), "ws "+uuid.New().String(), owner, router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			testStorage := storages.CreateTestStorage(workspace.ID)
			defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

			notifier := notifiers.CreateTestNotifier(workspace.ID)
			defer notifiers.RemoveTestNotifier(notifier)

			database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
			defer databases.RemoveTestDatabase(t.Context(), database)

			backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

			switch verificationCase.trigger {
			case VerificationTriggerScheduled:
				enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)
				require.NoError(t, GetVerificationScheduler().createScheduledRuns(t.Context(), logger.GetLogger()))
			case VerificationTriggerAfterBackup:
				enableAfterBackupVerificationViaAPI(t, router, owner.Token, database.ID)
				GetVerificationService().OnBackupCompleted(backup.ID)
			default:
				t.Fatalf("unsupported automatic verification trigger %q", verificationCase.trigger)
			}

			verifications := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
			require.Len(t, verifications, 1)
			automaticVerification := verifications[0]
			require.Equal(t, verificationCase.trigger, automaticVerification.Trigger)

			if verificationCase.isRunning {
				agent := verification_agents.CreateTestVerificationAgent(
					t, router, owner.Token, "automatic-running-"+uuid.New().String(),
				)
				defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

				assignment := ClaimVerificationViaAPI(
					t, router, agent.Agent.ID, agent.Token,
					AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 1},
				)
				require.Equal(t, automaticVerification.ID, assignment.VerificationID)
			}

			disabledConfig := verification_config.SaveBackupVerificationConfigDTO{
				IsScheduledVerificationEnabled: false,
				ScheduleType:                   verification_config.VerificationScheduleAfterBackup,
			}
			if verificationCase.trigger == VerificationTriggerScheduled {
				disabledConfig.ScheduleType = verification_config.VerificationScheduleInterval
				disabledConfig.VerificationInterval = intervals.Interval{Type: intervals.IntervalHourly}
			}

			verification_config.SaveVerificationConfigViaAPI(
				t, router, owner.Token, database.ID, disabledConfig,
			)

			require.NoError(
				t,
				GetVerificationScheduler().cancelAutomaticVerificationsForDisabledSchedules(
					t.Context(), logger.GetLogger(),
				),
			)

			verificationAfterSweep := GetVerificationByIDViaAPI(
				t, router, owner.Token, automaticVerification.ID,
			)
			assert.Equal(t, VerificationStatusCanceled, verificationAfterSweep.Status)
			require.NotNil(t, verificationAfterSweep.FailMessage)
			assert.Contains(t, *verificationAfterSweep.FailMessage, "schedule was disabled")
		})
	}
}

func Test_CancelAutomaticVerificationsForDisabledSchedules_WhenManualAndAutomaticVerificationsCoexist_CancelsOnlyAutomatic(
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
	manualVerification := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)
	require.NoError(t, GetVerificationScheduler().createScheduledRuns(t.Context(), logger.GetLogger()))

	verifications := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, verifications, 2)

	var scheduledVerification *RestoreVerification
	for _, verification := range verifications {
		if verification.Trigger == VerificationTriggerScheduled {
			scheduledVerification = verification
			break
		}
	}
	require.NotNil(t, scheduledVerification)

	verification_config.SaveVerificationConfigViaAPI(t, router, owner.Token, database.ID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: false,
			ScheduleType:                   verification_config.VerificationScheduleInterval,
			VerificationInterval:           intervals.Interval{Type: intervals.IntervalHourly},
		})

	require.NoError(
		t,
		GetVerificationScheduler().cancelAutomaticVerificationsForDisabledSchedules(t.Context(), logger.GetLogger()),
	)

	manualVerificationAfterSweep := GetVerificationByIDViaAPI(
		t, router, owner.Token, manualVerification.ID,
	)
	assert.Equal(t, VerificationStatusPending, manualVerificationAfterSweep.Status)
	assert.Nil(t, manualVerificationAfterSweep.FailMessage)

	scheduledVerificationAfterSweep := GetVerificationByIDViaAPI(
		t, router, owner.Token, scheduledVerification.ID,
	)
	assert.Equal(t, VerificationStatusCanceled, scheduledVerificationAfterSweep.Status)
	require.NotNil(t, scheduledVerificationAfterSweep.FailMessage)
	assert.Contains(t, *scheduledVerificationAfterSweep.FailMessage, "schedule was disabled")
}

func Test_CreateScheduledRuns_AfterUserCancel_DoesNotImmediatelyRecreate(t *testing.T) {
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

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	scheduledRowID := rows[0].ID

	CancelVerificationViaAPI(t, router, owner.Token, scheduledRowID)

	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rowsAfter := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Len(
		t,
		rowsAfter,
		1,
		"scheduler must not recreate a SCHEDULED row right after a user cancel within the interval window",
	)
	assert.Equal(t, VerificationStatusCanceled, rowsAfter[0].Status)
}

func Test_CreateScheduledRuns_AfterUserCancel_DoesNotRequeueOrRestart(t *testing.T) {
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

	enableHourlyVerificationViaAPI(t, router, owner.Token, database.ID)

	scheduler := GetVerificationScheduler()
	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 1)
	scheduledRowID := rows[0].ID

	CancelVerificationViaAPI(t, router, owner.Token, scheduledRowID)

	cancelled := GetVerificationByIDViaAPI(t, router, owner.Token, scheduledRowID)
	require.Equal(t, VerificationStatusCanceled, cancelled.Status)
	require.NotNil(t, cancelled.FinishedAt)
	cancelledFinishedAt := *cancelled.FinishedAt
	cancelledAttemptCount := cancelled.AttemptCount

	require.NoError(t, scheduler.createScheduledRuns(t.Context(), logger.GetLogger()))
	require.NoError(t, scheduler.reapStaleRuns(t.Context(), logger.GetLogger()))
	require.NoError(t, scheduler.cancelAutomaticVerificationsForDisabledSchedules(t.Context(), logger.GetLogger()))

	final := GetVerificationByIDViaAPI(t, router, owner.Token, scheduledRowID)
	assert.Equal(t, VerificationStatusCanceled, final.Status)
	require.NotNil(t, final.FinishedAt)
	assert.True(
		t,
		final.FinishedAt.Equal(cancelledFinishedAt),
		"finished_at must not be rewritten after a row is CANCELED",
	)
	assert.Equal(t, cancelledAttemptCount, final.AttemptCount, "attempt_count must not advance after CANCELED")
}
