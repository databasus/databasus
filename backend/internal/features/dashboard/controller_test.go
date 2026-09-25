package dashboard

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/walmath"
)

const walSegmentBytes = 16 * 1024 * 1024

type dashboardTestWorkspace struct {
	router    *gin.Engine
	owner     *users_dto.SignInResponseDTO
	workspace *workspaces_models.Workspace
	storage   *storages.Storage
	notifier  *notifiers.Notifier
}

func Test_GetWorkspaceDashboard_WithLogicalAndPhysicalDatabases_ReturnsPerDatabaseTotals(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)
	logicalDatabase := createTestLogicalDatabase(t, testWorkspace)
	physicalDatabase := createTestPhysicalDatabase(t, testWorkspace)

	now := time.Now().UTC()
	for _, status := range []backups_core_logical.BackupStatus{
		backups_core_logical.BackupStatusCompleted,
		backups_core_logical.BackupStatusCompleted,
		backups_core_logical.BackupStatusFailed,
		backups_core_logical.BackupStatusCanceled,
	} {
		backups_controllers_logical.CreateTestBackupWithOptions(
			logicalDatabase.ID,
			testWorkspace.storage.ID,
			backups_controllers_logical.TestBackupOptions{Status: status, CreatedAt: now},
		)
	}

	completedFullBackup := physical_testing.NewTestCompletedFullBackup(
		physicalDatabase.ID, testWorkspace.storage.ID, 1,
		walmath.LSN(0), walmath.LSN(walSegmentBytes),
	)
	completedFullBackup.BackupSizeMb = new(100.0)
	physical_testing.CreateTestFullBackup(t, completedFullBackup)

	failedFullBackup := physical_testing.NewTestInProgressFullBackup(physicalDatabase.ID, testWorkspace.storage.ID, 1)
	failedFullBackup.Status = physical_enums.PhysicalBackupStatusError
	physical_testing.CreateTestFullBackup(t, failedFullBackup)

	physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		physicalDatabase.ID, testWorkspace.storage.ID, 1, "000000010000000000000001",
		walmath.LSN(walSegmentBytes), walmath.LSN(2*walSegmentBytes),
	))

	var dashboard WorkspaceDashboard
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		testWorkspace.router,
		getWorkspaceDashboardURL(testWorkspace.workspace),
		"Bearer "+testWorkspace.owner.Token,
		http.StatusOK,
		&dashboard,
	)

	require.Len(t, dashboard.Databases, 2)

	logicalDashboardDatabase := findDashboardDatabase(t, dashboard, logicalDatabase)
	assert.Equal(t, int64(4), logicalDashboardDatabase.BackupsCount)
	assert.Equal(t, int64(2), logicalDashboardDatabase.CompletedBackupsCount)
	assert.Equal(t, int64(1), logicalDashboardDatabase.FailedBackupsCount)
	assert.InDelta(t, 10.5, logicalDashboardDatabase.MeanBackupSizeMb, 0.001)
	assert.InDelta(t, 21.0, logicalDashboardDatabase.TotalBackupSizeMb, 0.001)
	require.NotNil(t, logicalDashboardDatabase.Storage)
	assert.Equal(t, testWorkspace.storage.ID, logicalDashboardDatabase.Storage.ID)
	assert.Equal(t, testWorkspace.storage.Name, logicalDashboardDatabase.Storage.Name)
	assert.Equal(t, storages.StorageTypeLocal, logicalDashboardDatabase.Storage.Type)

	physicalDashboardDatabase := findDashboardDatabase(t, dashboard, physicalDatabase)
	assert.Equal(t, int64(2), physicalDashboardDatabase.BackupsCount)
	assert.Equal(t, int64(1), physicalDashboardDatabase.CompletedBackupsCount)
	assert.Equal(t, int64(1), physicalDashboardDatabase.FailedBackupsCount)
	assert.InDelta(t, 100.0, physicalDashboardDatabase.MeanBackupSizeMb, 0.001)
	assert.InDelta(t, 116.0, physicalDashboardDatabase.TotalBackupSizeMb, 0.001)
	require.NotNil(t, physicalDashboardDatabase.Storage)
	assert.Equal(t, testWorkspace.storage.ID, physicalDashboardDatabase.Storage.ID)

	assert.Equal(t, int64(2), dashboard.Totals.DatabasesCount)
	assert.Equal(t, int64(6), dashboard.Totals.BackupsCount)
	assert.InDelta(t, 137.0, dashboard.Totals.TotalBackupSizeMb, 0.001)
}

func Test_GetWorkspaceDashboard_WhenUserIsViewer_ReturnsDashboard(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)
	database := createTestLogicalDatabase(t, testWorkspace)

	viewer := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspaces_testing.AddMemberToWorkspace(
		testWorkspace.workspace,
		viewer,
		users_enums.WorkspaceRoleViewer,
		testWorkspace.owner.Token,
		testWorkspace.router,
	)

	var dashboard WorkspaceDashboard
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		testWorkspace.router,
		getWorkspaceDashboardURL(testWorkspace.workspace),
		"Bearer "+viewer.Token,
		http.StatusOK,
		&dashboard,
	)

	require.Len(t, dashboard.Databases, 1)
	assert.Equal(t, database.ID, dashboard.Databases[0].ID)
}

func Test_GetWorkspaceDashboard_WhenUserIsNotMember_ReturnsBadRequest(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)
	createTestLogicalDatabase(t, testWorkspace)

	nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	response := test_utils.MakeGetRequest(
		t,
		testWorkspace.router,
		getWorkspaceDashboardURL(testWorkspace.workspace),
		"Bearer "+nonMember.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "insufficient permissions to access this workspace")
}

func Test_GetWorkspaceDashboard_WhenHealthcheckEnabled_ReturnsLastTenAttemptsNewestFirst(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)
	database := createTestLogicalDatabase(t, testWorkspace)
	healthcheck_config.EnableHealthcheckForTestDatabase(database.ID, testWorkspace.owner.Token, testWorkspace.router)

	newestAttemptTime := time.Now().UTC().Truncate(time.Second)
	for attemptIndex := range 12 {
		status := databases.HealthStatusAvailable
		if attemptIndex == 0 {
			status = databases.HealthStatusUnavailable
		}

		healthcheck_attempt.CreateTestHealthcheckAttempt(
			database.ID,
			status,
			newestAttemptTime.Add(-time.Duration(attemptIndex)*time.Minute),
		)
	}

	var dashboard WorkspaceDashboard
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		testWorkspace.router,
		getWorkspaceDashboardURL(testWorkspace.workspace),
		"Bearer "+testWorkspace.owner.Token,
		http.StatusOK,
		&dashboard,
	)

	require.Len(t, dashboard.Databases, 1)
	recentAttempts := dashboard.Databases[0].RecentHealthcheckAttempts
	require.Len(t, recentAttempts, 10)
	assert.True(t, newestAttemptTime.Equal(recentAttempts[0].CreatedAt))
	assert.Equal(t, databases.HealthStatusUnavailable, recentAttempts[0].Status)
	for attemptIndex := 1; attemptIndex < len(recentAttempts); attemptIndex++ {
		assert.True(t, recentAttempts[attemptIndex-1].CreatedAt.After(recentAttempts[attemptIndex].CreatedAt))
	}
}

func Test_GetWorkspaceDashboard_WhenHealthcheckDisabled_ReturnsEmptyAttempts(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)
	database := createTestLogicalDatabase(t, testWorkspace)

	healthcheck_attempt.CreateTestHealthcheckAttempt(database.ID, databases.HealthStatusAvailable, time.Now().UTC())

	response := test_utils.MakeGetRequest(
		t,
		testWorkspace.router,
		getWorkspaceDashboardURL(testWorkspace.workspace),
		"Bearer "+testWorkspace.owner.Token,
		http.StatusOK,
	)

	assert.Contains(t, string(response.Body), `"recentHealthcheckAttempts":[]`)
}

func Test_GetWorkspaceDashboard_WhenWorkspaceIsEmpty_ReturnsZeroTotals(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)

	response := test_utils.MakeGetRequest(
		t,
		testWorkspace.router,
		getWorkspaceDashboardURL(testWorkspace.workspace),
		"Bearer "+testWorkspace.owner.Token,
		http.StatusOK,
	)

	assert.JSONEq(
		t,
		`{"databases":[],"totals":{"databasesCount":0,"backupsCount":0,"totalBackupSizeMb":0}}`,
		string(response.Body),
	)
}

func Test_GetWorkspaceDashboard_WithoutWorkspaceIdQuery_ReturnsBadRequest(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)

	response := test_utils.MakeGetRequest(
		t,
		testWorkspace.router,
		"/api/v1/dashboard",
		"Bearer "+testWorkspace.owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "workspace_id query parameter is required")
}

func Test_GetInstallationDashboard_WhenUserIsAdmin_ReturnsTotalsAcrossWorkspaces(t *testing.T) {
	firstWorkspace := createDashboardTestWorkspace(t)
	secondWorkspace := createDashboardTestWorkspace(t)
	firstDatabase := createTestLogicalDatabase(t, firstWorkspace)
	secondDatabase := createTestLogicalDatabase(t, secondWorkspace)

	backups_controllers_logical.CreateTestBackup(firstDatabase.ID, firstWorkspace.storage.ID)
	backups_controllers_logical.CreateTestBackup(secondDatabase.ID, secondWorkspace.storage.ID)

	admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)

	var installationTotals DashboardTotals
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		firstWorkspace.router,
		"/api/v1/dashboard/installation",
		"Bearer "+admin.Token,
		http.StatusOK,
		&installationTotals,
	)

	assert.GreaterOrEqual(t, installationTotals.DatabasesCount, int64(2))
	assert.GreaterOrEqual(t, installationTotals.BackupsCount, int64(2))
	assert.GreaterOrEqual(t, installationTotals.TotalBackupSizeMb, 21.0)
}

func Test_GetInstallationDashboard_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	testWorkspace := createDashboardTestWorkspace(t)

	response := test_utils.MakeGetRequest(
		t,
		testWorkspace.router,
		"/api/v1/dashboard/installation",
		"Bearer "+testWorkspace.owner.Token,
		http.StatusForbidden,
	)

	assert.Contains(t, string(response.Body), "Insufficient permissions")
}

func createDashboardTestWorkspace(t *testing.T) *dashboardTestWorkspace {
	t.Helper()

	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		healthcheck_config.GetHealthcheckConfigController(),
		GetDashboardController(),
	)

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Dashboard Test Workspace", owner, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	t.Cleanup(func() {
		storages.RemoveTestStorage(t.Context(), storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	})

	return &dashboardTestWorkspace{
		router:    router,
		owner:     owner,
		workspace: workspace,
		storage:   storage,
		notifier:  notifier,
	}
}

func createTestLogicalDatabase(t *testing.T, testWorkspace *dashboardTestWorkspace) *databases.Database {
	t.Helper()

	database := databases.CreateTestDatabase(testWorkspace.workspace.ID, testWorkspace.storage, testWorkspace.notifier)
	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, testWorkspace.storage)

	t.Cleanup(func() {
		databases.RemoveTestDatabase(t.Context(), database)
	})

	return database
}

func createTestPhysicalDatabase(t *testing.T, testWorkspace *dashboardTestWorkspace) *databases.Database {
	t.Helper()

	database := databases.CreateTestPhysicalPostgresDatabase(testWorkspace.workspace.ID, testWorkspace.notifier, "17")
	backups_config_physical.EnableBackupsForPhysicalTestDatabase(t.Context(), database.ID, testWorkspace.storage)

	t.Cleanup(func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, database.ID)
		databases.RemoveTestDatabase(t.Context(), database)
	})

	return database
}

func getWorkspaceDashboardURL(workspace *workspaces_models.Workspace) string {
	return fmt.Sprintf("/api/v1/dashboard?workspace_id=%s", workspace.ID.String())
}

func findDashboardDatabase(
	t *testing.T,
	dashboard WorkspaceDashboard,
	database *databases.Database,
) DashboardDatabase {
	t.Helper()

	for _, dashboardDatabase := range dashboard.Databases {
		if dashboardDatabase.ID == database.ID {
			return dashboardDatabase
		}
	}

	t.Fatalf("database %s is missing from the dashboard", database.ID)

	return DashboardDatabase{}
}
