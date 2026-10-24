package verification_runs

import (
	"context"
	"net/http"
	"sync"
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
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
	test_utils "databasus-backend/internal/util/testing"
)

type recordedNotification struct {
	notifier     *notifiers.Notifier
	notification notifier_models.Notification
}

type recordingNotificationSender struct {
	mu       sync.Mutex
	recorded []recordedNotification
	notifyCh chan struct{}
}

func newRecordingNotificationSender() *recordingNotificationSender {
	return &recordingNotificationSender{notifyCh: make(chan struct{}, 16)}
}

func (s *recordingNotificationSender) SendNotification(
	_ context.Context,
	notifier *notifiers.Notifier,
	notification notifier_models.Notification,
) {
	s.mu.Lock()
	s.recorded = append(s.recorded, recordedNotification{notifier, notification})
	s.mu.Unlock()

	s.notifyCh <- struct{}{}
}

func (s *recordingNotificationSender) waitForNotification(
	t *testing.T,
	timeout time.Duration,
) recordedNotification {
	t.Helper()

	select {
	case <-s.notifyCh:
	case <-time.After(timeout):
		t.Fatalf("timed out after %s waiting for notification", timeout)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.recorded[len(s.recorded)-1]
}

func (s *recordingNotificationSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.recorded)
}

func installRecordingNotificationSender(t *testing.T) *recordingNotificationSender {
	t.Helper()

	recorder := newRecordingNotificationSender()
	original := getVerificationService().notifierService
	getVerificationService().notifierService = recorder

	t.Cleanup(func() { getVerificationService().notifierService = original })

	return recorder
}

func enableFailureNotificationsViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	databaseID uuid.UUID,
) {
	t.Helper()

	verification_config.SaveVerificationConfigViaAPI(t, router, userToken, databaseID,
		verification_config.SaveBackupVerificationConfigDTO{
			IsScheduledVerificationEnabled: false,
			ScheduleType:                   verification_config.VerificationScheduleInterval,
			VerificationInterval:           intervals.Interval{Type: intervals.IntervalHourly},
			SendNotificationsOn: []verification_config.VerificationNotificationType{
				verification_config.NotificationVerificationFailed,
			},
		})
}

func Test_SubmitReport_WhenBackupRejected_SendsFailureNotification(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	enableFailureNotificationsViaAPI(t, router, owner.Token, database.ID)

	recorder := installRecordingNotificationSender(t)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	agent := verification_agents.CreateTestVerificationAgent(
		t,
		router,
		owner.Token,
		"notif-rejected-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	pgRestoreExit := 1
	failMessage := "pg_restore: schema mismatch"
	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		ReportRequest{
			Status:            VerificationStatusFailed,
			PgRestoreExitCode: &pgRestoreExit,
			FailMessage:       &failMessage,
		},
		http.StatusNoContent,
	)

	sent := recorder.waitForNotification(t, 2*time.Second)
	require.NotNil(t, sent.notifier)
	assert.Equal(t, notifier.ID, sent.notifier.ID, "must dispatch to the database's notifier")
	assert.Equal(t, notifier_models.NotificationTypeVerificationFailed, sent.notification.Type)
	assert.Contains(t, sent.notification.Heading, database.Name, "title must name the database")
	assert.Contains(t, sent.notification.Heading, "failed", "title must convey failure")
	assert.Contains(t, sent.notification.Message, failMessage,
		"notification body must include the agent's diagnostic")
}

func Test_SubmitReport_WhenRestoredTooSmall_SendsFailureNotification(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	enableFailureNotificationsViaAPI(t, router, owner.Token, database.ID)

	recorder := installRecordingNotificationSender(t)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "notif-tiny-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	tinyRestoredSize := int64(1_000_000) // 1 MB ≪ 20% of 100 MB
	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		ReportRequest{
			Status:                  VerificationStatusCompleted,
			PgRestoreExitCode:       new(int),
			DBSizeBytesAfterRestore: &tinyRestoredSize,
		},
		http.StatusNoContent,
	)

	sent := recorder.waitForNotification(t, 2*time.Second)
	require.NotNil(t, sent.notifier)
	assert.Equal(t, notifier.ID, sent.notifier.ID)
	assert.Equal(t, notifier_models.NotificationTypeVerificationFailed, sent.notification.Type)
	assert.Contains(t, sent.notification.Heading, "failed", "size-guard terminal must be reported as a failure")
	assert.Contains(t, sent.notification.Message, "less than 20%",
		"notification body must explain why the restore was rejected")
}

func Test_SubmitReport_WhenAgentSetupFailedRepeatedly_SendsFailureNotificationOnlyAtTerminal(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	enableFailureNotificationsViaAPI(t, router, owner.Token, database.ID)

	recorder := installRecordingNotificationSender(t)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "notif-setup-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	enqueued := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	for attempt := range MaxAgentSideAttempts {
		claim := ClaimVerificationViaAPI(
			t, router, agent.Agent.ID, agent.Token,
			AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
		)
		require.Equal(
			t,
			enqueued.ID,
			claim.VerificationID,
			"requeued row must be the same verification (attempt %d)",
			attempt+1,
		)

		failMessage := "agent setup failed"
		test_utils.MakePostRequest(
			t, router,
			reportPathFor(agent.Agent.ID, claim.VerificationID),
			"Bearer "+agent.Token,
			ReportRequest{Status: VerificationStatusFailed, FailMessage: &failMessage},
			http.StatusNoContent,
		)

		if attempt < MaxAgentSideAttempts-1 {
			// Mid-retry: confirm no notification was sent. Give goroutines a beat
			// to misfire before asserting the recorder is still empty.
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, 0, recorder.count(),
				"must not notify between retries (attempt %d)", attempt+1)
		}
	}

	sent := recorder.waitForNotification(t, 2*time.Second)
	assert.Equal(t, notifier.ID, sent.notifier.ID)
	assert.Equal(t, notifier_models.NotificationTypeVerificationFailed, sent.notification.Type)
	assert.Contains(t, sent.notification.Heading, "failed")
	assert.Equal(t, 1, recorder.count(), "exactly one notification on terminal — not one per attempt")
}

func Test_ReapStaleRuns_WhenAgentLostContact_SendsFailureNotificationOnlyAtTerminal(t *testing.T) {
	shrinkThresholds(t, 1*time.Hour, 1*time.Millisecond)

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	enableFailureNotificationsViaAPI(t, router, owner.Token, database.ID)

	recorder := installRecordingNotificationSender(t)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	agent := verification_agents.CreateTestVerificationAgent(
		t,
		router,
		owner.Token,
		"notif-silent-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	enqueued := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	for attempt := range MaxAgentSideAttempts {
		ClaimVerificationViaAPI(
			t, router, agent.Agent.ID, agent.Token,
			AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
		)

		require.NoError(t, storage.GetDb().
			Model(&verification_agents.Agent{}).
			Where("id = ?", agent.Agent.ID).
			Update("last_seen_at", time.Now().UTC().Add(-1*time.Hour)).Error)

		require.NoError(t, GetVerificationScheduler().reapStaleRuns(t.Context(), logger.GetLogger()))

		if attempt < MaxAgentSideAttempts-1 {
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, 0, recorder.count(),
				"must not notify between reaper-driven retries (attempt %d)", attempt+1)
		}
	}

	sent := recorder.waitForNotification(t, 2*time.Second)
	assert.Equal(t, notifier.ID, sent.notifier.ID)
	assert.Equal(t, notifier_models.NotificationTypeVerificationFailed, sent.notification.Type)
	assert.Contains(t, sent.notification.Heading, "failed")
	assert.Contains(t, sent.notification.Message, "agent went silent")

	final := GetVerificationByIDViaAPI(t, router, owner.Token, enqueued.ID)
	assert.Equal(t, VerificationStatusFailed, final.Status)
}

func Test_SubmitReport_WhenFailureNotificationsDisabled_DoesNotSend(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	// Default config has IsScheduledVerificationEnabled=false and no notification types.
	// Do not call enableFailureNotificationsViaAPI — failure notification must NOT fire.

	recorder := installRecordingNotificationSender(t)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "notif-off-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	pgRestoreExit := 1
	failMessage := "pg_restore: schema mismatch"
	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		ReportRequest{
			Status:            VerificationStatusFailed,
			PgRestoreExitCode: &pgRestoreExit,
			FailMessage:       &failMessage,
		},
		http.StatusNoContent,
	)

	time.Sleep(100 * time.Millisecond) // let any goroutine misfire surface
	assert.Equal(t, 0, recorder.count(), "no failure notification type configured → no send")
}
