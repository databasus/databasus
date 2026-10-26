package healthcheck_attempt

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	databases_bridge "databasus-backend/internal/features/databases"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
	test_utils "databasus-backend/internal/util/testing"
)

func createTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		databases_bridge.GetTestRoutes(),
		GetHealthcheckAttemptController(),
	)
	return router
}

func Test_GetAttemptsByDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *workspaces.Role
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can get healthcheck attempts",
			workspaceRole:      func() *workspaces.Role { r := workspaces.RoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can get healthcheck attempts",
			workspaceRole:      func() *workspaces.Role { r := workspaces.RoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get healthcheck attempts",
			workspaceRole:      func() *workspaces.Role { r := workspaces.RoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can get healthcheck attempts",
			workspaceRole:      func() *workspaces.Role { r := workspaces.RoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can get healthcheck attempts",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get healthcheck attempts",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

			database := databases_bridge.CreateTestDatabaseViaAPI(t,
				router,
				owner.Token,
				workspace.ID,
				"Test Database",
			)

			pastTime := time.Now().UTC().Add(-1 * time.Hour)
			createTestHealthcheckAttemptWithTime(
				database.ID,
				databases.HealthStatusAvailable,
				pastTime,
			)
			createTestHealthcheckAttemptWithTime(
				database.ID,
				databases.HealthStatusUnavailable,
				pastTime.Add(-30*time.Minute),
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == workspaces.RoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole)

				testUserToken = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				testUserToken = nonMember.Token
			}

			if tt.expectSuccess {
				var response []*HealthcheckAttempt
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/healthcheck-attempts/"+database.ID.String(),
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
					&response,
				)

				assert.GreaterOrEqual(t, len(response), 2)
			} else {
				testResp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/healthcheck-attempts/"+database.ID.String(),
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(testResp.Body), "forbidden")
			}

			// Cleanup
			databases_bridge.RemoveTestDatabase(t.Context(), database)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
		})
	}
}

func Test_GetAttemptsByDatabase_FiltersByAfterDate(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := databases_bridge.CreateTestDatabaseViaAPI(t, router, owner.Token, workspace.ID, "Test Database")

	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	recentTime := time.Now().UTC().Add(-30 * time.Minute)

	createTestHealthcheckAttemptWithTime(database.ID, databases.HealthStatusAvailable, oldTime)
	createTestHealthcheckAttemptWithTime(database.ID, databases.HealthStatusUnavailable, recentTime)
	createTestHealthcheckAttempt(database.ID, databases.HealthStatusAvailable)

	afterDate := time.Now().UTC().Add(-1 * time.Hour)
	var response []*HealthcheckAttempt
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf(
			"/api/v1/healthcheck-attempts/%s?afterDate=%s",
			database.ID.String(),
			afterDate.Format(time.RFC3339),
		),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, 2, len(response))
	for _, attempt := range response {
		assert.True(t, attempt.CreatedAt.After(afterDate) || attempt.CreatedAt.Equal(afterDate))
	}

	// Cleanup
	databases_bridge.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
}

func Test_GetAttemptsByDatabase_ReturnsEmptyListForNewDatabase(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := databases_bridge.CreateTestDatabaseViaAPI(t, router, owner.Token, workspace.ID, "Test Database")

	var response []*HealthcheckAttempt
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/healthcheck-attempts/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, 0, len(response))

	// Cleanup
	databases_bridge.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
}

func createTestHealthcheckAttempt(databaseID uuid.UUID, status databases.HealthStatus) {
	createTestHealthcheckAttemptWithTime(databaseID, status, time.Now().UTC())
}

func createTestHealthcheckAttemptWithTime(
	databaseID uuid.UUID,
	status databases.HealthStatus,
	createdAt time.Time,
) {
	repo := GetHealthcheckAttemptRepository()
	attempt := &HealthcheckAttempt{
		ID:         uuid.New(),
		DatabaseID: databaseID,
		Status:     status,
		CreatedAt:  createdAt,
	}
	if err := repo.Create(attempt); err != nil {
		panic("Failed to create test healthcheck attempt: " + err.Error())
	}
}
