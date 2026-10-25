package verification_runs

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_agents "databasus-backend/internal/features/verification/agents"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
	test_utils "databasus-backend/internal/util/testing"
)

func claimPathFor(agentID uuid.UUID) string {
	return fmt.Sprintf("/api/v1/agent/verifications/%s/claim", agentID)
}

func reportPathFor(agentID, verificationID uuid.UUID) string {
	return fmt.Sprintf("/api/v1/agent/verifications/%s/%s/report", agentID, verificationID)
}

func Test_ClaimVerification_WhenPendingFitsBudget_ReturnsJobAssignment(t *testing.T) {
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

	enqueued := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "claim-test-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	assert.Equal(t, enqueued.ID, assignment.VerificationID)
	assert.Equal(t, backup.ID, assignment.BackupID)
	assert.InDelta(t, 100, assignment.BackupSizeMb, 0.001)
	assert.InDelta(t, float64(EstimateRequiredForRestoreDiskMb(backup)), assignment.MaxContainerDiskMb, 0.001)
	require.NotNil(t, assignment.Database)
	assert.Equal(t, databases.DatabaseTypePostgresLogical, assignment.Database.Type)
	require.NotNil(t, assignment.Database.PostgresqlLogical)
	assert.Equal(t, "16", string(assignment.Database.PostgresqlLogical.Version))

	updated := GetVerificationByIDViaAPI(t, router, owner.Token, enqueued.ID)
	assert.Equal(t, VerificationStatusRunning, updated.Status)
	require.NotNil(t, updated.AgentID)
	assert.Equal(t, agent.Agent.ID, *updated.AgentID)
}

func Test_ClaimVerification_WhenAgentHasFiveGbAndBackupIsTiny_ReturnsJobAssignment(t *testing.T) {
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

	tinyBackup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 0.01)

	enqueuedVerification := EnqueueManualVerificationViaAPI(t, router, owner.Token, tinyBackup.ID)

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "small-disk-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 2, MaxRAMMb: 1024, MaxDiskGb: 5, MaxConcurrentJobs: 1},
	)

	assert.Equal(t, enqueuedVerification.ID, assignment.VerificationID)
	assert.Equal(t, tinyBackup.ID, assignment.BackupID)
}

func Test_ClaimVerification_WhenBackupUsesTimescaleDB_ForwardsTimescaledbVersion(t *testing.T) {
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
	backup.TimescaledbVersion = "2.17.0"
	require.NoError(t, backups_core_logical.GetBackupRepository().Save(backup))

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "ts-claim-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	assert.Equal(t, "2.17.0", assignment.TimescaledbVersion)
}

func Test_ClaimVerification_WhenBackupTooBigForBudget_DoNotAssignJobWithoutError(t *testing.T) {
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

	largeBackup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100*1024)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, largeBackup.ID)

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "no-room-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	test_utils.MakePostRequest(
		t, router,
		claimPathFor(agent.Agent.ID),
		"Bearer "+agent.Token,
		ClaimRequest{Capacity: AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 1, MaxConcurrentJobs: 2}},
		http.StatusNoContent,
	)
}

func Test_ClaimVerification_WhenMultiplePendingFit_ReturnsOldestThatFitsBudget(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	databaseSmall := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), databaseSmall)
	databaseBig := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), databaseBig)

	oldBig := backuping_logical.SeedTestBackup(t, databaseBig.ID, testStorage.ID, 4*1024)
	youngSmall := backuping_logical.SeedTestBackup(t, databaseSmall.ID, testStorage.ID, 100)

	oldVerification := EnqueueManualVerificationViaAPI(t, router, owner.Token, oldBig.ID)
	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", oldVerification.ID).
		Update("created_at", time.Now().UTC().Add(-1*time.Hour)).Error)

	youngVerification := EnqueueManualVerificationViaAPI(t, router, owner.Token, youngSmall.ID)

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "size-pick-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 6, MaxConcurrentJobs: 2},
	)

	assert.Equal(
		t, youngVerification.ID, assignment.VerificationID,
		"the older oversized verification must be skipped by the fit check, not picked over the younger one that fits",
	)
}

func Test_ClaimVerification_WhenAgentHasRunningRows_DeductsTheirSizeFromBudget(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	databaseRunning := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), databaseRunning)
	databasePending := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), databasePending)

	bigBackup := backuping_logical.SeedTestBackup(t, databaseRunning.ID, testStorage.ID, 1000)
	pendingBackup := backuping_logical.SeedTestBackup(t, databasePending.ID, testStorage.ID, 1000)

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "budget-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, bigBackup.ID)

	roomyCapacity := ClaimRequest{
		Capacity: AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	}

	var firstAssignment JobAssignment
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		claimPathFor(agent.Agent.ID),
		"Bearer "+agent.Token,
		roomyCapacity,
		http.StatusOK, &firstAssignment,
	)
	assert.Equal(t, bigBackup.ID, firstAssignment.BackupID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, pendingBackup.ID)

	test_utils.MakePostRequest(
		t, router,
		claimPathFor(agent.Agent.ID),
		"Bearer "+agent.Token,
		ClaimRequest{Capacity: AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 4, MaxConcurrentJobs: 2}},
		http.StatusNoContent,
	)
}

func Test_ClaimVerification_WhenAlreadyClaimedByAnotherAgent_OnlyReassignsAfterCancelAndReschedule(t *testing.T) {
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

	originalRow := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	agentA := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "a-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agentA.Agent.ID)
	agentB := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "b-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agentB.Agent.ID)

	capacity := AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2}

	firstAssignment := ClaimVerificationViaAPI(t, router, agentA.Agent.ID, agentA.Token, capacity)
	assert.Equal(t, originalRow.ID, firstAssignment.VerificationID)

	test_utils.MakePostRequest(
		t, router,
		claimPathFor(agentB.Agent.ID),
		"Bearer "+agentB.Token,
		ClaimRequest{Capacity: capacity},
		http.StatusNoContent,
	)

	afterFirstClaim := GetVerificationByIDViaAPI(t, router, owner.Token, originalRow.ID)
	require.NotNil(t, afterFirstClaim.AgentID)
	assert.Equal(t, agentA.Agent.ID, *afterFirstClaim.AgentID)

	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", originalRow.ID).
		Update("status", VerificationStatusCanceled).Error)

	test_utils.MakePostRequest(
		t, router,
		claimPathFor(agentB.Agent.ID),
		"Bearer "+agentB.Token,
		ClaimRequest{Capacity: capacity},
		http.StatusNoContent,
	)

	rescheduledRow := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	secondAssignment := ClaimVerificationViaAPI(t, router, agentB.Agent.ID, agentB.Token, capacity)
	assert.Equal(t, rescheduledRow.ID, secondAssignment.VerificationID)
	assert.NotEqual(t, originalRow.ID, secondAssignment.VerificationID)

	finalOriginal := GetVerificationByIDViaAPI(t, router, owner.Token, originalRow.ID)
	require.NotNil(t, finalOriginal.AgentID)
	assert.Equal(t, agentA.Agent.ID, *finalOriginal.AgentID, "original row's owner must never have flipped to agent B")
}

func Test_SubmitReport_WhenStatusCompleted_PersistsResultsAndTableStats(t *testing.T) {
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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "report-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	rowCount := int64(1234)
	tableCount := 5
	schemaCount := 1
	// Healthy restore: well above minRestoredSizeRatio (20%) of the seeded
	// 100 MB raw size, so the size guard does not flag it.
	dbSize := int64(80_000_000)
	restoreMs := int64(8000)
	verifyMs := int64(2000)

	report := ReportRequest{
		Status:                  VerificationStatusCompleted,
		PgRestoreExitCode:       new(int),
		RestoreDurationMs:       &restoreMs,
		VerifyDurationMs:        &verifyMs,
		DBSizeBytesAfterRestore: &dbSize,
		TableCount:              &tableCount,
		SchemaCount:             &schemaCount,
		TableStats: []ReportTableStat{
			{SchemaName: "public", Name: "users", RowCount: rowCount},
			{SchemaName: "public", Name: "orders", RowCount: 7},
		},
	}

	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		report,
		http.StatusNoContent,
	)

	final := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusCompleted, final.Status)
	require.NotNil(t, final.FinishedAt)
	require.Len(t, final.TableStats, 2)

	verifiedBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusVerifiedSuccessful,
		verifiedBackup.RestoreVerificationStatus,
		"a terminal COMPLETED verification must stamp the backup VERIFIED_SUCCESSFUL",
	)
}

func Test_SubmitReport_WhenCompletedWithNonZeroExitCode_MarksCompleted(t *testing.T) {
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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "report-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	// When the agent tolerates a restore whose only failures were missing
	// extensions, it reports COMPLETED but carries the real non-zero pg_restore
	// exit code. The backend must accept that as success (the size guard, not the
	// exit code, is the arbiter on the COMPLETED path).
	pgRestoreExit := 1
	dbSize := int64(80_000_000)

	report := ReportRequest{
		Status:                  VerificationStatusCompleted,
		PgRestoreExitCode:       &pgRestoreExit,
		DBSizeBytesAfterRestore: &dbSize,
	}

	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		report,
		http.StatusNoContent,
	)

	final := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusCompleted, final.Status,
		"a COMPLETED report with a non-zero pg_restore exit (tolerated missing extensions) must not be rejected")
	require.NotNil(t, final.PgRestoreExitCode)
	assert.Equal(t, 1, *final.PgRestoreExitCode)

	verifiedBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusVerifiedSuccessful,
		verifiedBackup.RestoreVerificationStatus,
		"a tolerated missing-extension restore is a successful verification",
	)
}

func Test_SubmitReport_WhenRowWasReapedWhileAgentWasSilent_Returns410(t *testing.T) {
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

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "silent-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	require.NoError(t, GetVerificationScheduler().reapStaleRuns(t.Context(), logger.GetLogger()))

	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		ReportRequest{Status: VerificationStatusCompleted},
		http.StatusGone,
	)
}

func Test_SubmitReport_WhenAgentReportsBackupRejected_StoresMessageVerbatimAndTerminal(t *testing.T) {
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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "fail-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	pgRestoreExit := 1
	verbatimMessage := "pg_restore: server closed connection unexpectedly"
	report := ReportRequest{
		Status:            VerificationStatusFailed,
		PgRestoreExitCode: &pgRestoreExit,
		FailMessage:       &verbatimMessage,
	}

	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		report,
		http.StatusNoContent,
	)

	final := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(
		t,
		VerificationStatusFailed,
		final.Status,
		"pg_restore non-zero exit is backup-side → terminal immediately",
	)
	assert.Equal(t, 1, final.AttemptCount, "backup-side failure must NOT advance the attempt counter")
	require.NotNil(t, final.FailMessage)
	assert.Equal(t, verbatimMessage, *final.FailMessage)

	rejectedBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusVerificationFailed,
		rejectedBackup.RestoreVerificationStatus,
		"a PG-error terminal failure must stamp the backup VERIFICATION_FAILED",
	)
}

func Test_SubmitReport_WhenAgentReportsDiskLimitExceeded_MarksTerminalWithoutRetry(t *testing.T) {
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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "disk-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	// FailureKind drives the verdict even though PgRestoreExitCode is nil (which
	// would otherwise be the retryable agent-side path) — the per-job disk
	// budget is server-computed, so exceeding it is terminal, not retried.
	diskKind := string(FailureReasonDiskLimitExceeded)
	verbatimMessage := "restore exceeded job disk budget of 200 MB"
	report := ReportRequest{
		Status:      VerificationStatusFailed,
		FailureKind: &diskKind,
		FailMessage: &verbatimMessage,
	}

	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		report,
		http.StatusNoContent,
	)

	final := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(
		t,
		VerificationStatusFailed,
		final.Status,
		"exceeding the server-computed disk budget is terminal",
	)
	assert.Equal(t, 1, final.AttemptCount, "disk-limit failure must NOT advance the attempt counter")
	require.NotNil(t, final.FailMessage)
	assert.Equal(t, verbatimMessage, *final.FailMessage)

	rejectedBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusVerificationFailed,
		rejectedBackup.RestoreVerificationStatus,
		"a terminal disk-limit failure must stamp the backup VERIFICATION_FAILED",
	)
}

func Test_SubmitReport_WhenAgentSetupFailedWithRetriesRemaining_RequeuesVerification(t *testing.T) {
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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "retry-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	// PgRestoreExitCode==nil → agent-side (setup failure), eligible for retry.
	failMessage := "download torn before pg_restore could run"
	test_utils.MakePostRequest(
		t, router,
		reportPathFor(agent.Agent.ID, assignment.VerificationID),
		"Bearer "+agent.Token,
		ReportRequest{Status: VerificationStatusFailed, FailMessage: &failMessage},
		http.StatusNoContent,
	)

	requeued := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusPending, requeued.Status, "must requeue, not go terminal")
	assert.Equal(t, 2, requeued.AttemptCount, "attempt counter must advance")
	assert.Nil(t, requeued.AgentID, "agent ownership must be released on requeue")
	assert.Nil(t, requeued.FailMessage, "fail message must be cleared so next attempt starts clean")
	assert.Nil(t, requeued.StartedAt)
	assert.Nil(t, requeued.FinishedAt)

	requeuedBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusNotVerified,
		requeuedBackup.RestoreVerificationStatus,
		"a requeued (non-terminal) agent-side failure must leave the backup NOT_VERIFIED",
	)
}

func Test_SubmitReport_WhenAgentSetupFailedRepeatedly_MarksTerminalAfterMaxAttempts(t *testing.T) {
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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "exhaust-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	enqueued := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	failMessages := []string{
		"setup failure: attempt 1",
		"setup failure: attempt 2",
		"setup failure: attempt 3 (terminal)",
	}

	for attempt, failMessage := range failMessages {
		claim := ClaimVerificationViaAPI(
			t, router, agent.Agent.ID, agent.Token,
			AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
		)
		require.Equal(t, enqueued.ID, claim.VerificationID,
			"requeued row must be the same verification, not a fresh one (attempt %d)", attempt+1)

		test_utils.MakePostRequest(
			t, router,
			reportPathFor(agent.Agent.ID, claim.VerificationID),
			"Bearer "+agent.Token,
			ReportRequest{Status: VerificationStatusFailed, FailMessage: &failMessage},
			http.StatusNoContent,
		)
	}

	final := GetVerificationByIDViaAPI(t, router, owner.Token, enqueued.ID)
	assert.Equal(t, VerificationStatusFailed, final.Status, "third failure exhausts MaxAgentSideAttempts → terminal")
	assert.Equal(t, MaxAgentSideAttempts, final.AttemptCount, "attempt counter stays at the exhausted value")
	require.NotNil(t, final.FailMessage)
	assert.Equal(t, failMessages[len(failMessages)-1], *final.FailMessage,
		"terminal message is the last agent-reported one verbatim")

	failedAfterAllRetriesBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusVerificationFailed,
		failedAfterAllRetriesBackup.RestoreVerificationStatus,
		"agent-side failures that exhaust all retries are terminal → backup VERIFICATION_FAILED",
	)
}

func Test_SubmitReport_WhenCompletedButRestoredSizeBelow20Percent_MarksTerminalWithoutRetry(t *testing.T) {
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

	// Backup with a recorded raw DB size of 100 MB ≈ 104_857_600 bytes — the 20%
	// floor is therefore ~20 MB. Reporting a 1 MB restored DB must trip the guard.
	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	agent := verification_agents.CreateTestVerificationAgent(
		t,
		router,
		owner.Token,
		"tiny-restore-"+uuid.New().String(),
	)
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

	final := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusFailed, final.Status,
		"size guard converts a too-small COMPLETED into a terminal FAILED")
	assert.Equal(t, 1, final.AttemptCount, "backup-side failure must NOT advance the attempt counter")
	require.NotNil(t, final.FailMessage)
	assert.Contains(t, *final.FailMessage, "less than 20%")

	tooSmallBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusVerificationFailed,
		tooSmallBackup.RestoreVerificationStatus,
		"a too-small restore routes through the failure path → backup VERIFICATION_FAILED, never successful",
	)
}

func Test_Heartbeat_WhenReportedJobIsCanceledOrDeleted_ReturnsItAsAbortID(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	databaseRunning := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), databaseRunning)
	databaseCanceled := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), databaseCanceled)

	runningBackup := backuping_logical.SeedTestBackup(t, databaseRunning.ID, testStorage.ID, 100)
	canceledBackup := backuping_logical.SeedTestBackup(t, databaseCanceled.ID, testStorage.ID, 100)

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "hb-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	roomyCapacity := AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2}

	EnqueueManualVerificationViaAPI(t, router, owner.Token, runningBackup.ID)
	stillRunning := ClaimVerificationViaAPI(t, router, agent.Agent.ID, agent.Token, roomyCapacity)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, canceledBackup.ID)
	canceled := ClaimVerificationViaAPI(t, router, agent.Agent.ID, agent.Token, roomyCapacity)
	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", canceled.VerificationID).
		Update("status", VerificationStatusCanceled).Error)

	deletedRowID := uuid.New()

	currentIDs := []uuid.UUID{stillRunning.VerificationID, canceled.VerificationID, deletedRowID}

	var heartbeatResponse verification_agents.HeartbeatResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		fmt.Sprintf("/api/v1/agent/verification/%s/heartbeat", agent.Agent.ID),
		"Bearer "+agent.Token,
		verification_agents.HeartbeatRequest{
			MaxCPU: 1, MaxRAMGb: 1, MaxDiskGb: 1, MaxConcurrentJobs: 1,
			CurrentVerificationIDs: currentIDs,
		},
		http.StatusOK, &heartbeatResponse,
	)

	require.Len(t, heartbeatResponse.AbortVerificationIDs, 2)

	abortSet := map[uuid.UUID]struct{}{}
	for _, id := range heartbeatResponse.AbortVerificationIDs {
		abortSet[id] = struct{}{}
	}
	_, canceledAborted := abortSet[canceled.VerificationID]
	_, deletedAborted := abortSet[deletedRowID]
	_, stillOursAborted := abortSet[stillRunning.VerificationID]

	assert.True(t, canceledAborted, "canceled row must be in abort list")
	assert.True(t, deletedAborted, "row that no longer exists must be in abort list")
	assert.False(t, stillOursAborted, "still-running row owned by this agent must not be aborted")
}

func Test_CancelVerification_WhenRunning_AgentHeartbeatReturnsIDAsAbort(t *testing.T) {
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

	agent := verification_agents.CreateTestVerificationAgent(
		t,
		router,
		owner.Token,
		"cancel-running-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	CancelVerificationViaAPI(t, router, owner.Token, assignment.VerificationID)

	var heartbeatResponse verification_agents.HeartbeatResponse
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		fmt.Sprintf("/api/v1/agent/verification/%s/heartbeat", agent.Agent.ID),
		"Bearer "+agent.Token,
		verification_agents.HeartbeatRequest{
			MaxCPU: 1, MaxRAMGb: 1, MaxDiskGb: 1, MaxConcurrentJobs: 1,
			CurrentVerificationIDs: []uuid.UUID{assignment.VerificationID},
		},
		http.StatusOK, &heartbeatResponse,
	)

	require.Len(t, heartbeatResponse.AbortVerificationIDs, 1,
		"agent must learn about the user-cancel on its next heartbeat")
	assert.Equal(t, assignment.VerificationID, heartbeatResponse.AbortVerificationIDs[0])

	canceled := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusCanceled, canceled.Status)

	canceledBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusNotVerified,
		canceledBackup.RestoreVerificationStatus,
		"cancellation bypasses the terminal hook → backup stays NOT_VERIFIED",
	)
}

func Test_GetBackups_WhenBackupNeverVerified_StatusIsNotVerified(t *testing.T) {
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

	freshBackup := GetBackupViaAPI(t, router, owner.Token, database.ID, backup.ID)
	assert.Equal(
		t,
		backups_core_logical.RestoreVerificationStatusNotVerified,
		freshBackup.RestoreVerificationStatus,
		"a backup that was never verified defaults to NOT_VERIFIED via the column default",
	)
}
