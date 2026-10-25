package workspaces_controllers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_enums "databasus-backend/internal/features/users/enums"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_CreateWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name                  string
		userRole              users.Role
		memberCreationEnabled bool
		expectSuccess         bool
		expectedStatusCode    int
		expectedWorkspaceName string
	}{
		{
			name:                  "member can create workspace when enabled",
			userRole:              users.RoleMember,
			memberCreationEnabled: true,
			expectSuccess:         true,
			expectedStatusCode:    http.StatusOK,
			expectedWorkspaceName: "Test Workspace",
		},
		{
			name:                  "member cannot create workspace when disabled",
			userRole:              users.RoleMember,
			memberCreationEnabled: false,
			expectSuccess:         false,
			expectedStatusCode:    http.StatusForbidden,
		},
		{
			name:                  "admin can create workspace when member creation disabled",
			userRole:              users.RoleAdmin,
			memberCreationEnabled: false,
			expectSuccess:         true,
			expectedStatusCode:    http.StatusOK,
			expectedWorkspaceName: "Admin Workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := workspaces_testing.CreateTestRouter(
				GetWorkspaceController(),
				GetMembershipController(),
			)

			if tt.memberCreationEnabled {
				users_testing.EnableMemberWorkspaceCreation(t.Context())
			} else {
				users_testing.DisableMemberWorkspaceCreation(t.Context())
			}
			defer users_testing.ResetSettingsToDefaults(t.Context())

			user := users_testing.CreateTestUser(t.Context(), tt.userRole)

			uniqueID := uuid.New().String()[:8]
			workspaceName := tt.expectedWorkspaceName + " " + uniqueID
			request := workspaces_dto.CreateWorkspaceRequestDTO{
				Name: workspaceName,
			}

			if tt.expectSuccess {
				var response workspaces_dto.WorkspaceResponseDTO
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/workspaces",
					"Bearer "+user.Token,
					request,
					tt.expectedStatusCode,
					&response,
				)

				assert.Equal(t, workspaceName, response.Name)
				assert.NotEqual(t, uuid.Nil, response.ID)
				assert.Equal(t, users_enums.WorkspaceRoleOwner, *response.UserRole)

				// Cleanup created workspace
				workspace := &workspaces_models.Workspace{ID: response.ID}
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			} else {
				resp := test_utils.MakePostRequest(
					t,
					router,
					"/api/v1/workspaces",
					"Bearer "+user.Token,
					request,
					tt.expectedStatusCode,
				)
				assert.Contains(
					t,
					string(resp.Body),
					"insufficient permissions to create workspaces",
				)
			}
		})
	}
}

func Test_CreateWorkspace_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/workspaces",
		Body:           "invalid json",
		AuthToken:      "Bearer " + user.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_CreateWorkspace_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)

	request := workspaces_dto.CreateWorkspaceRequestDTO{
		Name: "Test Workspace",
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces",
		"",
		request,
		http.StatusUnauthorized,
	)
}

func Test_GetUserWorkspaces_WhenUserHasWorkspaces_ReturnsWorkspacesList(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	workspace1, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
		"Workspace 1",
		user.Token,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1, router)

	workspace2, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
		"Workspace 2",
		user.Token,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2, router)

	var response workspaces_dto.ListWorkspacesResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces",
		"Bearer "+user.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.Workspaces), 2)

	workspaceNames := make([]string, len(response.Workspaces))
	for i, w := range response.Workspaces {
		workspaceNames[i] = w.Name
	}
	assert.Contains(t, workspaceNames, workspace1.Name)
	assert.Contains(t, workspaceNames, workspace2.Name)
}

func Test_GetUserWorkspaces_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	test_utils.MakeGetRequest(t, router, "/api/v1/workspaces", "", http.StatusUnauthorized)
}

func Test_GetSingleWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can get workspace",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can get workspace",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get workspace",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can get workspace",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get workspace",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "global admin can get workspace",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := workspaces_testing.CreateTestRouter(
				GetWorkspaceController(),
				GetMembershipController(),
			)
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
				"Test Workspace",
				owner.Token,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

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
				var response workspaces_models.Workspace
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/workspaces/"+workspace.ID.String(),
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
					&response,
				)

				assert.Equal(t, workspace.ID, response.ID)
				assert.Equal(t, "Test Workspace", response.Name)
			} else {
				resp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/workspaces/"+workspace.ID.String(),
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(resp.Body), "insufficient permissions to view workspace")
			}
		})
	}
}

func Test_GetSingleWorkspace_WithInvalidWorkspaceID_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/workspaces/invalid-uuid",
		"Bearer "+user.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid workspace ID")
}

func Test_UpdateWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      users_enums.WorkspaceRole
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can update workspace",
			workspaceRole:      users_enums.WorkspaceRoleOwner,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can update workspace",
			workspaceRole:      users_enums.WorkspaceRoleAdmin,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member cannot update workspace",
			workspaceRole:      users_enums.WorkspaceRoleMember,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace viewer cannot update workspace",
			workspaceRole:      users_enums.WorkspaceRoleViewer,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := workspaces_testing.CreateTestRouter(
				GetWorkspaceController(),
				GetMembershipController(),
			)
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
				"Original Name",
				owner.Token,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			var testUserToken string
			if tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else {
				member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			updateRequest := workspaces_models.Workspace{
				Name: "Updated Name",
			}

			if tt.expectSuccess {
				var response workspaces_models.Workspace
				test_utils.MakePutRequestAndUnmarshal(
					t,
					router,
					"/api/v1/workspaces/"+workspace.ID.String(),
					"Bearer "+testUserToken,
					updateRequest,
					tt.expectedStatusCode,
					&response,
				)

				assert.Equal(t, workspace.ID, response.ID)
				assert.Equal(t, "Updated Name", response.Name)
			} else {
				resp := test_utils.MakePutRequest(
					t,
					router,
					"/api/v1/workspaces/"+workspace.ID.String(),
					"Bearer "+testUserToken,
					updateRequest,
					tt.expectedStatusCode,
				)
				assert.Contains(
					t,
					string(resp.Body),
					"insufficient permissions to update workspace",
				)
			}
		})
	}
}

func Test_DeleteWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can delete workspace",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can delete workspace",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member cannot delete workspace",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace admin cannot delete workspace",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := workspaces_testing.CreateTestRouter(
				GetWorkspaceController(),
				GetMembershipController(),
			)
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
				"Test Workspace",
				owner.Token,
				router,
			)
			// Only cleanup if the test doesn't successfully delete the workspace
			if !tt.expectSuccess {
				defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}

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

			resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
				Method:         "DELETE",
				URL:            "/api/v1/workspaces/" + workspace.ID.String(),
				AuthToken:      "Bearer " + testUserToken,
				ExpectedStatus: tt.expectedStatusCode,
			})

			if tt.expectSuccess {
				assert.Contains(t, string(resp.Body), "Workspace deleted successfully")
			} else {
				assert.Contains(
					t,
					string(resp.Body),
					"only workspace owner or admin can delete workspace",
				)
			}
		})
	}
}

func Test_GetWorkspaceAuditLogs_WhenUserIsWorkspaceAdmin_ReturnsAuditLogs(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspaceAdmin := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	uniqueID := uuid.New()
	workspaceName := fmt.Sprintf("WorkspaceAdmin Test %s", uniqueID.String()[:8])
	workspace, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
		workspaceName,
		owner.Token,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	workspaces_testing.AddMemberToWorkspace(
		workspace,
		workspaceAdmin,
		users_enums.WorkspaceRoleMember,
		owner.Token,
		router,
	)
	var response audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/workspaces/"+workspace.ID.String()+"/audit-logs",
		"Bearer "+workspaceAdmin.Token, http.StatusOK, &response)

	assert.GreaterOrEqual(t, len(response.AuditLogs), 2) // Create + Add member
	for _, log := range response.AuditLogs {
		assert.Equal(t, &workspace.ID, log.WorkspaceID)
	}
}

func TestGetWorkspaceAuditLogs(t *testing.T) {
	t.Run("shows the author's email and name from the event when the user changed both afterwards",
		func(t *testing.T) {
			router := workspaces_testing.CreateTestRouter(
				GetWorkspaceController(),
				GetMembershipController(),
			)
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			ownerBeforeChange, err := users_repositories.GetUserRepository().GetUserByID(t.Context(), owner.UserID)
			require.NoError(t, err)
			workspace, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
				"Snapshot Test "+uuid.NewString()[:8], owner.Token, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			changedName := "Renamed " + uuid.NewString()
			changedEmail := "renamed-" + uuid.NewString() + "@example.com"
			require.NoError(t, users_repositories.GetUserRepository().UpdateUserInfo(
				owner.UserID, &changedName, &changedEmail,
			))

			var listedLogs audit_logs.GetAuditLogsResponse
			test_utils.MakeGetRequestAndUnmarshal(t, router,
				"/api/v1/workspaces/"+workspace.ID.String()+"/audit-logs",
				"Bearer "+owner.Token, http.StatusOK, &listedLogs)

			require.NotEmpty(t, listedLogs.AuditLogs)
			for _, listedLog := range listedLogs.AuditLogs {
				require.Equal(t, &owner.UserID, listedLog.UserID)
				assert.Equal(t, &ownerBeforeChange.Email, listedLog.UserEmail)
				assert.Equal(t, &ownerBeforeChange.Name, listedLog.UserName)
			}
		},
	)
}

func Test_GetWorkspaceAuditLogs_WithMultipleWorkspaces_ReturnsOnlyWorkspaceSpecificLogs(
	t *testing.T,
) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	owner2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	uniqueID1 := uuid.New()
	uniqueID2 := uuid.New()
	workspaceName1 := fmt.Sprintf("Workspace Test %s", uniqueID1.String()[:8])
	workspaceName2 := fmt.Sprintf("Workspace Test %s", uniqueID2.String()[:8])

	workspace1, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
		workspaceName1,
		owner1.Token,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1, router)

	workspace2, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
		workspaceName2,
		owner2.Token,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2, router)

	updateWorkspace1 := workspaces_models.Workspace{
		Name: "Updated " + workspace1.Name,
	}
	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/workspaces/"+workspace1.ID.String(),
		"Bearer "+owner1.Token,
		updateWorkspace1,
		http.StatusOK,
	)

	updateWorkspace2 := workspaces_models.Workspace{
		Name: "Updated " + workspace2.Name,
	}
	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/workspaces/"+workspace2.ID.String(),
		"Bearer "+owner2.Token,
		updateWorkspace2,
		http.StatusOK,
	)

	var workspace1Response audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/workspaces/"+workspace1.ID.String()+"/audit-logs?limit=50",
		"Bearer "+owner1.Token, http.StatusOK, &workspace1Response)

	var workspace2Response audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/workspaces/"+workspace2.ID.String()+"/audit-logs?limit=50",
		"Bearer "+owner2.Token, http.StatusOK, &workspace2Response)

	assert.GreaterOrEqual(t, len(workspace1Response.AuditLogs), 2)
	for _, log := range workspace1Response.AuditLogs {
		assert.Equal(t, &workspace1.ID, log.WorkspaceID)
		assert.Contains(t, log.Message, workspaceName1)
	}

	assert.GreaterOrEqual(t, len(workspace2Response.AuditLogs), 2)
	for _, log := range workspace2Response.AuditLogs {
		assert.Equal(t, &workspace2.ID, log.WorkspaceID)
		assert.Contains(t, log.Message, workspaceName2)
	}

	workspace1Messages := extractAuditLogMessages(workspace1Response.AuditLogs)
	workspace2Messages := extractAuditLogMessages(workspace2Response.AuditLogs)

	for _, msg := range workspace1Messages {
		assert.NotContains(
			t,
			msg,
			workspaceName2,
			"Workspace1 logs should not contain Workspace2 name",
		)
	}

	for _, msg := range workspace2Messages {
		assert.NotContains(
			t,
			msg,
			workspaceName1,
			"Workspace2 logs should not contain Workspace1 name",
		)
	}
}

func Test_GetWorkspaceAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	globalAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	uniqueID := uuid.New()
	workspaceName := fmt.Sprintf("Audit Test Workspace %s", uniqueID.String()[:8])
	workspace, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
		workspaceName,
		owner.Token,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	workspaces_testing.AddMemberToWorkspace(
		workspace,
		member,
		users_enums.WorkspaceRoleViewer,
		owner.Token,
		router,
	)
	var ownerResponse audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/workspaces/"+workspace.ID.String()+"/audit-logs",
		"Bearer "+owner.Token, http.StatusOK, &ownerResponse)

	assert.GreaterOrEqual(t, len(ownerResponse.AuditLogs), 2)
	var memberResponse audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/workspaces/"+workspace.ID.String()+"/audit-logs",
		"Bearer "+member.Token, http.StatusOK, &memberResponse)

	assert.GreaterOrEqual(t, len(memberResponse.AuditLogs), 2)

	var globalAdminResponse audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/workspaces/"+workspace.ID.String()+"/audit-logs",
		"Bearer "+globalAdmin.Token, http.StatusOK, &globalAdminResponse)

	assert.GreaterOrEqual(t, len(globalAdminResponse.AuditLogs), 2)

	resp := test_utils.MakeGetRequest(t, router,
		"/api/v1/workspaces/"+workspace.ID.String()+"/audit-logs",
		"Bearer "+nonMember.Token, http.StatusForbidden)

	assert.Contains(t, string(resp.Body), "insufficient permissions to view workspace audit logs")
}

func Test_GetWorkspaceAuditLogs_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceWithToken(t.Context(),
		"Test Workspace",
		owner.Token,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	test_utils.MakeGetRequest(t, router,
		"/api/v1/workspaces/"+workspace.ID.String()+"/audit-logs",
		"", http.StatusUnauthorized)
}

func extractAuditLogMessages(logs []*audit_logs.AuditLogDTO) []string {
	messages := make([]string, len(logs))
	for i, log := range logs {
		messages[i] = log.Message
	}
	return messages
}

type AuditLogWriterStub struct{}

func (a *AuditLogWriterStub) WriteAuditLog(context.Context, audit_logs_models.AuditEntry) {
}
