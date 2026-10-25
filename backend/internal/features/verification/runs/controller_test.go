package verification_runs

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	test_utils "databasus-backend/internal/util/testing"
)

func createTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		workspaces_controllers.GetWorkspaceController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetMembershipController().RegisterRoutes(routerGroup)
		databases.GetDatabaseController().RegisterRoutes(routerGroup)
		backups_controllers_logical.GetBackupController().RegisterRoutes(routerGroup)
		GetVerificationController().RegisterRoutes(routerGroup)
		verification_agents.GetAgentController().RegisterRoutes(routerGroup)
		verification_config.GetVerificationConfigController().RegisterRoutes(routerGroup)
	}

	verification_agents.GetAgentFacingController().RegisterRoutes(v1)
	GetVerificationAgentController().RegisterRoutes(v1)

	audit_logs.SetupDependencies()
	verification_config.SetupDependencies()
	SetupDependencies()

	return router
}

func Test_EnqueueManualVerification_AsOwner_CreatesPendingRow(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 500)

	var verification RestoreVerification
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		"/api/v1/verifications/enqueue",
		"Bearer "+owner.Token,
		EnqueueManualRequest{BackupID: backup.ID},
		http.StatusOK, &verification,
	)

	assert.Equal(t, VerificationStatusPending, verification.Status)
	assert.Equal(t, VerificationTriggerManual, verification.Trigger)
	assert.Equal(t, backup.ID, verification.BackupID)
	assert.Equal(t, database.ID, verification.DatabaseID)
	assert.Equal(t, 1, verification.AttemptCount)
	assert.Nil(t, verification.AgentID)
}

func Test_EnqueueManualVerification_WhenManualPendingExists_Returns400(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	firstBackup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	secondBackup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 200)

	var firstVerification RestoreVerification
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/verifications/enqueue", "Bearer "+owner.Token,
		EnqueueManualRequest{BackupID: firstBackup.ID},
		http.StatusOK, &firstVerification,
	)

	resp := test_utils.MakePostRequest(
		t, router, "/api/v1/verifications/enqueue", "Bearer "+owner.Token,
		EnqueueManualRequest{BackupID: secondBackup.ID},
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "please cancel it first")

	firstAfter := GetVerificationByIDViaAPI(t, router, owner.Token, firstVerification.ID)
	assert.Equal(t, VerificationStatusPending, firstAfter.Status,
		"first manual verification must remain PENDING — the second enqueue was rejected, not a displacement")

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Len(t, rows, 1, "rejected enqueue must not create a second row")
}

func Test_EnqueueManualVerification_WhenRunningExists_Returns400(t *testing.T) {
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

	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "running-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	resp := test_utils.MakePostRequest(
		t, router, "/api/v1/verifications/enqueue", "Bearer "+owner.Token,
		EnqueueManualRequest{BackupID: backup.ID},
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "please cancel it first")
}

func Test_EnqueueManualVerification_WhenScheduledNonTerminalExists_DoesNotDisplaceIt(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	scheduledRow := &RestoreVerification{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		BackupID:     backup.ID,
		Trigger:      VerificationTriggerScheduled,
		Status:       VerificationStatusPending,
		AttemptCount: 1,
		CreatedAt:    time.Now().UTC(),
	}
	require.NoError(t, storage.GetDb().Create(scheduledRow).Error)

	var manualVerification RestoreVerification
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/verifications/enqueue", "Bearer "+owner.Token,
		EnqueueManualRequest{BackupID: backup.ID},
		http.StatusOK, &manualVerification,
	)

	scheduledAfter := GetVerificationByIDViaAPI(t, router, owner.Token, scheduledRow.ID)
	assert.Equal(
		t,
		VerificationStatusPending,
		scheduledAfter.Status,
		"scheduled row must not be displaced by a manual enqueue",
	)
	assert.Equal(t, VerificationTriggerScheduled, scheduledAfter.Trigger)

	manualAfter := GetVerificationByIDViaAPI(t, router, owner.Token, manualVerification.ID)
	assert.Equal(t, VerificationStatusPending, manualAfter.Status)
	assert.Equal(t, VerificationTriggerManual, manualAfter.Trigger)

	rows := ListVerificationsByDatabaseViaAPI(t, router, owner.Token, database.ID)
	require.Len(t, rows, 2, "exactly one scheduled and one manual non-terminal row should coexist")

	var manualCount, scheduledCount int
	for _, row := range rows {
		switch row.Trigger {
		case VerificationTriggerManual:
			manualCount++
		case VerificationTriggerScheduled:
			scheduledCount++
		}
	}
	assert.Equal(t, 1, manualCount)
	assert.Equal(t, 1, scheduledCount)
}

func Test_EnqueueManualVerification_AsNonWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)
	defer databases.RemoveTestDatabase(t.Context(), database)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)

	outsider := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	resp := test_utils.MakePostRequest(
		t, router, "/api/v1/verifications/enqueue", "Bearer "+outsider.Token,
		EnqueueManualRequest{BackupID: backup.ID},
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")
}

func Test_GetVerificationByID_ReturnsTableStats_WhileListByDatabaseOmitsThem(t *testing.T) {
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
	agent := verification_agents.CreateTestVerificationAgent(t, router, owner.Token, "stats-"+uuid.New().String())
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	test_utils.MakePostRequest(
		t, router,
		"/api/v1/agent/verifications/"+agent.Agent.ID.String()+"/"+assignment.VerificationID.String()+"/report",
		"Bearer "+agent.Token,
		ReportRequest{
			Status: VerificationStatusCompleted,
			TableStats: []ReportTableStat{
				{SchemaName: "public", Name: "users", RowCount: 42},
			},
		},
		http.StatusNoContent,
	)

	detail := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	require.Len(t, detail.TableStats, 1)
	assert.Equal(t, "users", detail.TableStats[0].Name)
	assert.Equal(t, int64(42), detail.TableStats[0].RowCount)

	listResponse := test_utils.MakeGetRequest(
		t, router,
		"/api/v1/verifications/by-database/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
	)

	var listed struct {
		Verifications []map[string]any `json:"verifications"`
	}
	require.NoError(t, json.Unmarshal(listResponse.Body, &listed))
	require.Len(t, listed.Verifications, 1)

	_, hasTableStats := listed.Verifications[0]["tableStats"]
	assert.False(t, hasTableStats, "list endpoint must not include tableStats")
}

func Test_ListVerificationsByDatabase_WithLimit_CapsRowsAndReturnsTotal(t *testing.T) {
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

	for range 3 {
		verification := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
		CancelVerificationViaAPI(t, router, owner.Token, verification.ID)
	}

	var page GetVerificationsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		"/api/v1/verifications/by-database/"+database.ID.String()+"?limit=2&offset=0",
		"Bearer "+owner.Token,
		http.StatusOK,
		&page,
	)

	assert.Len(t, page.Verifications, 2)
	assert.Equal(t, int64(3), page.Total)
	assert.Equal(t, 2, page.Limit)
	assert.Equal(t, 0, page.Offset)
}

func Test_CancelVerification_AsOwner_FlipsToCanceled(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
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

	CancelVerificationViaAPI(t, router, owner.Token, enqueued.ID)

	canceled := GetVerificationByIDViaAPI(t, router, owner.Token, enqueued.ID)
	assert.Equal(t, VerificationStatusCanceled, canceled.Status)
	require.NotNil(t, canceled.FinishedAt)
	require.NotNil(t, canceled.FailMessage)
	assert.Contains(t, *canceled.FailMessage, "by user")
}

func Test_CancelVerification_WhenAlreadyCompleted_Returns400(t *testing.T) {
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
		"cancel-completed-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)
	assignment := ClaimVerificationViaAPI(
		t, router, agent.Agent.ID, agent.Token,
		AgentCapacity{MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2},
	)

	test_utils.MakePostRequest(
		t, router,
		"/api/v1/agent/verifications/"+agent.Agent.ID.String()+"/"+assignment.VerificationID.String()+"/report",
		"Bearer "+agent.Token,
		ReportRequest{Status: VerificationStatusCompleted, PgRestoreExitCode: new(int)},
		http.StatusNoContent,
	)

	resp := test_utils.MakePostRequest(
		t, router,
		"/api/v1/verifications/"+assignment.VerificationID.String()+"/cancel",
		"Bearer "+owner.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "not cancellable")

	final := GetVerificationByIDViaAPI(t, router, owner.Token, assignment.VerificationID)
	assert.Equal(t, VerificationStatusCompleted, final.Status)
}

func Test_CancelVerification_AsNonWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
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

	outsider := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	resp := test_utils.MakePostRequest(
		t, router,
		"/api/v1/verifications/"+enqueued.ID.String()+"/cancel",
		"Bearer "+outsider.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")

	stillPending := GetVerificationByIDViaAPI(t, router, owner.Token, enqueued.ID)
	assert.Equal(t, VerificationStatusPending, stillPending.Status)
}

func Test_CancelVerification_WithAgentToken_Returns401(t *testing.T) {
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

	agent := verification_agents.CreateTestVerificationAgent(
		t, router, owner.Token, "cancel-agent-rejected-"+uuid.New().String(),
	)
	defer verification_agents.RemoveTestVerificationAgent(t, router, owner.Token, agent.Agent.ID)

	test_utils.MakePostRequest(
		t, router,
		"/api/v1/verifications/"+enqueued.ID.String()+"/cancel",
		"Bearer "+agent.Token,
		nil,
		http.StatusUnauthorized,
	)

	stillPending := GetVerificationByIDViaAPI(t, router, owner.Token, enqueued.ID)
	assert.Equal(t, VerificationStatusPending, stillPending.Status)
}

func Test_DeleteDatabase_RemovesVerifications_ViaCascade(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "ws "+uuid.New().String(), owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testStorage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), testStorage.ID)

	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	backup := backuping_logical.SeedTestBackup(t, database.ID, testStorage.ID, 100)
	enqueued := EnqueueManualVerificationViaAPI(t, router, owner.Token, backup.ID)

	test_utils.MakeDeleteRequest(
		t, router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	var remainingVerifications int64
	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", enqueued.ID).
		Count(&remainingVerifications).Error)
	assert.Zero(t, remainingVerifications, "verification row must be cascade-deleted with its database")
}

func Test_DeleteBackup_RemovesVerifications_ViaCascade(t *testing.T) {
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

	test_utils.MakeDeleteRequest(
		t, router,
		"/api/v1/backups/"+backup.ID.String(),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	var remainingVerifications int64
	require.NoError(t, storage.GetDb().
		Model(&RestoreVerification{}).
		Where("id = ?", enqueued.ID).
		Count(&remainingVerifications).Error)
	assert.Zero(t, remainingVerifications, "verification row must be cascade-deleted with its backup")
}
