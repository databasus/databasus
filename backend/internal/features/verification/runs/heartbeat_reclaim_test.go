package verification_runs

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_agents "databasus-backend/internal/features/verification/agents"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
)

func shrinkReclaimGrace(t *testing.T, grace time.Duration) {
	t.Helper()

	original := agentJobReclaimGrace
	agentJobReclaimGrace = grace

	t.Cleanup(func() { agentJobReclaimGrace = original })
}

func newReclaimFixture(
	t *testing.T,
) (router *gin.Engine, ownerToken string, database databases.Database, storageID uuid.UUID) {
	t.Helper()

	router = createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace) })

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID) })

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	t.Cleanup(func() { notifiers_bridge.RemoveTestNotifier(notifier) })

	database = databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	t.Cleanup(func() { databases_bridge.RemoveTestDatabase(t.Context(), database) })

	return router, owner.Token, database, testStorage.ID
}

func newReclaimAgent(
	t *testing.T,
	router *gin.Engine,
	ownerToken string,
) *verification_agents.CreatedAgentResponse {
	t.Helper()

	agent := verification_agents.CreateTestVerificationAgent(
		t, router, ownerToken, "reclaim-"+uuid.New().String(),
	)
	t.Cleanup(func() {
		verification_agents.RemoveTestVerificationAgent(t, router, ownerToken, agent.Agent.ID)
	})

	return agent
}

func claimRunningJob(
	t *testing.T,
	router *gin.Engine,
	ownerToken string,
	agent *verification_agents.CreatedAgentResponse,
	backupID uuid.UUID,
) *JobAssignment {
	t.Helper()

	EnqueueManualVerificationViaAPI(t, router, ownerToken, backupID)

	return ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)
}

func backdateStartedAt(t *testing.T, verificationID uuid.UUID) {
	t.Helper()

	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", verificationID).
		Update("started_at", time.Now().UTC().Add(-1*time.Hour)).Error)
}

func Test_OnAgentHeartbeated_WhenLiveAgentDropsRunningJobPastGrace_Requeues(t *testing.T) {
	shrinkReclaimGrace(t, 1*time.Millisecond)

	router, ownerToken, database, storageID := newReclaimFixture(t)
	agent := newReclaimAgent(t, router, ownerToken)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, storageID, 100)
	assignment := claimRunningJob(t, router, ownerToken, agent, backup.ID)
	backdateStartedAt(t, assignment.VerificationID)

	_, err := GetVerificationService().OnAgentHeartbeated(t.Context(), agent.Agent, nil)
	require.NoError(t, err)

	requeued := GetVerificationByIDViaAPI(t, router, ownerToken, assignment.VerificationID)
	assert.Equal(t, VerificationStatusPending, requeued.Status)
	assert.Equal(t, 2, requeued.AttemptCount)
	assert.Nil(t, requeued.AgentID)
	assert.Nil(t, requeued.FailMessage)
}

func Test_OnAgentHeartbeated_WhenJobStillReportedByAgent_DoesNotRequeue(t *testing.T) {
	shrinkReclaimGrace(t, 1*time.Millisecond)

	router, ownerToken, database, storageID := newReclaimFixture(t)
	agent := newReclaimAgent(t, router, ownerToken)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, storageID, 100)
	assignment := claimRunningJob(t, router, ownerToken, agent, backup.ID)
	backdateStartedAt(t, assignment.VerificationID)

	_, err := GetVerificationService().OnAgentHeartbeated(t.Context(),
		agent.Agent, []uuid.UUID{assignment.VerificationID},
	)
	require.NoError(t, err)

	stillRunning := GetVerificationByIDViaAPI(t, router, ownerToken, assignment.VerificationID)
	assert.Equal(t, VerificationStatusRunning, stillRunning.Status,
		"a job the agent still reports must not be reclaimed")
	assert.Equal(t, 1, stillRunning.AttemptCount)
	require.NotNil(t, stillRunning.AgentID)
	assert.Equal(t, agent.Agent.ID, *stillRunning.AgentID)
}

func Test_OnAgentHeartbeated_WhenRunningJobWithinGrace_DoesNotRequeue(t *testing.T) {
	shrinkReclaimGrace(t, 1*time.Hour)

	router, ownerToken, database, storageID := newReclaimFixture(t)
	agent := newReclaimAgent(t, router, ownerToken)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, storageID, 100)
	assignment := claimRunningJob(t, router, ownerToken, agent, backup.ID)

	_, err := GetVerificationService().OnAgentHeartbeated(t.Context(), agent.Agent, nil)
	require.NoError(t, err)

	stillRunning := GetVerificationByIDViaAPI(t, router, ownerToken, assignment.VerificationID)
	assert.Equal(t, VerificationStatusRunning, stillRunning.Status,
		"a freshly-claimed job within the grace window must not be reclaimed")
	assert.Equal(t, 1, stillRunning.AttemptCount)
}

func Test_OnAgentHeartbeated_WhenDroppedJobAtMaxAttempts_MarksTerminal(t *testing.T) {
	shrinkReclaimGrace(t, 1*time.Millisecond)

	router, ownerToken, database, storageID := newReclaimFixture(t)
	agent := newReclaimAgent(t, router, ownerToken)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, storageID, 100)
	assignment := claimRunningJob(t, router, ownerToken, agent, backup.ID)

	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", assignment.VerificationID).
		Updates(map[string]any{
			"attempt_count": MaxAgentSideAttempts,
			"started_at":    time.Now().UTC().Add(-1 * time.Hour),
		}).Error)

	_, err := GetVerificationService().OnAgentHeartbeated(t.Context(), agent.Agent, nil)
	require.NoError(t, err)

	final := GetVerificationByIDViaAPI(t, router, ownerToken, assignment.VerificationID)
	assert.Equal(t, VerificationStatusFailed, final.Status)
	assert.Equal(t, MaxAgentSideAttempts, final.AttemptCount, "terminal must not advance attempt_count")
	require.NotNil(t, final.FinishedAt)
	require.NotNil(t, final.FailMessage)
	assert.Contains(t, *final.FailMessage, "dropped by its still-online agent")
}

func Test_OnAgentHeartbeated_WhenJobAlreadyCompleted_DoesNotResurrect(t *testing.T) {
	shrinkReclaimGrace(t, 1*time.Millisecond)

	router, ownerToken, database, storageID := newReclaimFixture(t)
	agent := newReclaimAgent(t, router, ownerToken)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, storageID, 100)
	assignment := claimRunningJob(t, router, ownerToken, agent, backup.ID)

	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", assignment.VerificationID).
		Updates(map[string]any{
			"status":      VerificationStatusCompleted,
			"finished_at": time.Now().UTC(),
			"started_at":  time.Now().UTC().Add(-1 * time.Hour),
		}).Error)

	_, err := GetVerificationService().OnAgentHeartbeated(t.Context(), agent.Agent, nil)
	require.NoError(t, err)

	final := GetVerificationByIDViaAPI(t, router, ownerToken, assignment.VerificationID)
	assert.Equal(t, VerificationStatusCompleted, final.Status,
		"a concurrently-completed row must not be resurrected by the reclaim")
}

func Test_OnAgentHeartbeated_WhenAgentDropsOneStillRunsAnother_ReclaimsOnlyDropped(t *testing.T) {
	shrinkReclaimGrace(t, 1*time.Millisecond)

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

	testStorage := storages_bridge.CreateTestStorage(workspace.ID)
	defer storages_bridge.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	defer notifiers_bridge.RemoveTestNotifier(notifier)

	databaseKept := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), databaseKept)

	databaseDropped := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	defer databases_bridge.RemoveTestDatabase(t.Context(), databaseDropped)

	agent := newReclaimAgent(t, router, owner.Token)

	backupKept := backups_bridge.SeedTestLogicalBackup(t, databaseKept.ID, testStorage.ID, 100)
	backupDropped := backups_bridge.SeedTestLogicalBackup(t, databaseDropped.ID, testStorage.ID, 100)

	kept := claimRunningJob(t, router, owner.Token, agent, backupKept.ID)
	dropped := claimRunningJob(t, router, owner.Token, agent, backupDropped.ID)
	backdateStartedAt(t, kept.VerificationID)
	backdateStartedAt(t, dropped.VerificationID)

	aborts, err := GetVerificationService().OnAgentHeartbeated(t.Context(),
		agent.Agent, []uuid.UUID{kept.VerificationID},
	)
	require.NoError(t, err)
	assert.Empty(t, aborts, "a job still RUNNING and owned must not be in the abort list")

	keptRow := GetVerificationByIDViaAPI(t, router, owner.Token, kept.VerificationID)
	assert.Equal(t, VerificationStatusRunning, keptRow.Status)
	assert.Equal(t, 1, keptRow.AttemptCount)

	droppedRow := GetVerificationByIDViaAPI(t, router, owner.Token, dropped.VerificationID)
	assert.Equal(t, VerificationStatusPending, droppedRow.Status)
	assert.Equal(t, 2, droppedRow.AttemptCount)
	assert.Nil(t, droppedRow.AgentID)
}

func Test_OnAgentHeartbeated_WhenAgentReportsNoLongerOwnedJob_ReturnsItInAbortList(t *testing.T) {
	shrinkReclaimGrace(t, 1*time.Millisecond)

	router, ownerToken, database, storageID := newReclaimFixture(t)
	agent := newReclaimAgent(t, router, ownerToken)

	backup := backups_bridge.SeedTestLogicalBackup(t, database.ID, storageID, 100)
	assignment := claimRunningJob(t, router, ownerToken, agent, backup.ID)

	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", assignment.VerificationID).
		Updates(map[string]any{
			"status":      VerificationStatusCompleted,
			"finished_at": time.Now().UTC(),
		}).Error)

	aborts, err := GetVerificationService().OnAgentHeartbeated(t.Context(),
		agent.Agent, []uuid.UUID{assignment.VerificationID},
	)
	require.NoError(t, err)

	assert.Contains(t, aborts, assignment.VerificationID,
		"a job the agent still reports but the backend no longer owns as RUNNING must be returned for abort")

	final := GetVerificationByIDViaAPI(t, router, ownerToken, assignment.VerificationID)
	assert.Equal(t, VerificationStatusCompleted, final.Status,
		"computing the abort list must not mutate the row")
}
