package healthcheck_config

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/features/databases"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func createTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetHealthcheckConfigController(),
	)
	return router
}

func Test_SaveHealthcheckConfig_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can save healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can save healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can save healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer cannot save healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can save healthcheck config",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			request := HealthcheckConfigDTO{
				DatabaseID:                        database.ID,
				IsHealthcheckEnabled:              true,
				IsSentNotificationWhenUnavailable: true,
				IntervalMinutes:                   5,
				AttemptsBeforeConcideredAsDown:    3,
				StoreAttemptsDays:                 7,
			}

			if tt.expectSuccess {
				var response map[string]string
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/healthcheck-config",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
					&response,
				)
				assert.Contains(t, response["message"], "successfully")
			} else {
				testResp := test_utils.MakePostRequest(
					t,
					router,
					"/api/v1/healthcheck-config",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}

			// Cleanup
			databases.RemoveTestDatabase(t.Context(), database)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_SaveHealthcheckConfig_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := HealthcheckConfigDTO{
		DatabaseID:                        database.ID,
		IsHealthcheckEnabled:              true,
		IsSentNotificationWhenUnavailable: true,
		IntervalMinutes:                   5,
		AttemptsBeforeConcideredAsDown:    3,
		StoreAttemptsDays:                 7,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/healthcheck-config",
		"Bearer "+nonMember.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_GetHealthcheckConfig_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can get healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can get healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can get healthcheck config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can get healthcheck config",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get healthcheck config",
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
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				testUserToken = nonMember.Token
			}

			if tt.expectSuccess {
				var response HealthcheckConfig
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/healthcheck-config/"+database.ID.String(),
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
					&response,
				)

				assert.Equal(t, database.ID, response.DatabaseID)
				assert.True(t, response.IsHealthcheckEnabled)
			} else {
				testResp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/healthcheck-config/"+database.ID.String(),
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}

			// Cleanup
			databases.RemoveTestDatabase(t.Context(), database)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_GetHealthcheckConfig_ReturnsDefaultConfigForNewDatabase(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	var response HealthcheckConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/healthcheck-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.True(t, response.IsHealthcheckEnabled)
	assert.True(t, response.IsSentNotificationWhenUnavailable)
	assert.Equal(t, 1, response.IntervalMinutes)
	assert.Equal(t, 3, response.AttemptsBeforeConcideredAsDown)
	assert.Equal(t, 7, response.StoreAttemptsDays)

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
