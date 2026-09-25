package healthcheck_attempt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/features/databases"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
)

func createTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetHealthcheckAttemptController(),
	)
	return router
}

func Test_GetAttemptsByDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can get healthcheck attempts",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can get healthcheck attempts",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get healthcheck attempts",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can get healthcheck attempts",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			pastTime := time.Now().UTC().Add(-1 * time.Hour)
			CreateTestHealthcheckAttempt(
				database.ID,
				databases.HealthStatusAvailable,
				pastTime,
			)
			CreateTestHealthcheckAttempt(
				database.ID,
				databases.HealthStatusUnavailable,
				pastTime.Add(-30*time.Minute),
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
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
			databases.RemoveTestDatabase(t.Context(), database)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_GetAttemptsByDatabase_FiltersByAfterDate(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	recentTime := time.Now().UTC().Add(-30 * time.Minute)

	CreateTestHealthcheckAttempt(database.ID, databases.HealthStatusAvailable, oldTime)
	CreateTestHealthcheckAttempt(database.ID, databases.HealthStatusUnavailable, recentTime)
	CreateTestHealthcheckAttempt(database.ID, databases.HealthStatusAvailable, time.Now().UTC())

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
	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_GetAttemptsByDatabase_ReturnsEmptyListForNewDatabase(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

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
	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func createTestDatabaseViaAPI(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *databases.Database {
	request := databases.Database{
		WorkspaceID:       &workspaceID,
		Name:              name,
		Type:              databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: databases.GetTestPostgresConfig(),
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		panic("Failed to create database")
	}

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}
	return &database
}
