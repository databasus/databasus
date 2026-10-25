package workspaces_controllers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/email"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
)

// ListMembers Tests

func Test_GetWorkspaceMembers_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can view members",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can view members",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can view members",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can view members",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot view members",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "global admin can view members",
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					member,
					*tt.workspaceRole,
					router,
				)
				testUserToken = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				testUserToken = nonMember.Token
			}

			if tt.expectSuccess {
				var response workspaces_dto.GetMembersResponseDTO
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
					&response,
				)

				assert.GreaterOrEqual(t, len(response.Members), 1)
			} else {
				resp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
				)
				assert.Contains(
					t,
					string(resp.Body),
					"insufficient permissions to view workspace members",
				)
			}
		})
	}
}

func Test_GetWorkspaceMembers_WithInvalidWorkspaceID_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/invalid-uuid/members",
		"Bearer "+user.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid workspace ID")
}

// AddMember Tests

func Test_AddMemberToWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can add member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can add member",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can add member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member cannot add member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace viewer cannot add member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			newMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			request := workspaces_dto.AddMemberRequestDTO{
				Email: newMember.Email,
				Role:  users_enums.WorkspaceRoleViewer,
			}

			if tt.expectSuccess {
				var response workspaces_dto.AddMemberResponseDTO
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
					&response,
				)

				assert.True(t, response.Status == workspaces_dto.AddStatusAdded)
			} else {
				resp := test_utils.MakePostRequest(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(resp.Body), "insufficient permissions to manage members")
			}
		})
	}
}

func Test_AddMemberToWorkspace_WhenUserIsAlreadyMember_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
		workspace,
		member,
		users_enums.WorkspaceRoleViewer,
		router,
	)

	// Try to add the same user again
	request := workspaces_dto.AddMemberRequestDTO{
		Email: member.Email,
		Role:  users_enums.WorkspaceRoleViewer,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "user is already a member of this workspace")
}

func Test_AddMemberToWorkspace_WithNonExistentUser_ReturnsInvited(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	request := workspaces_dto.AddMemberRequestDTO{
		Email: uuid.New().String() + "@example.com", // Non-existent user
		Role:  users_enums.WorkspaceRoleViewer,
	}

	var response workspaces_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == workspaces_dto.AddStatusInvited)
}

func Test_AddMemberToWorkspace_WithNonExistentUser_InvitationLinksToConfiguredUrl(t *testing.T) {
	for _, databasusURL := range []string{
		"https://databasus.example.com",
		"https://databasus.example.com/",
	} {
		t.Run(databasusURL, func(t *testing.T) {
			mockEmailSender := useMockMembershipEmailSender(t)
			useDatabasusURL(t, databasusURL)

			router := workspaces_testing.CreateTestRouter(
				GetWorkspaceController(),
				GetMembershipController(),
			)
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			request := workspaces_dto.AddMemberRequestDTO{
				Email: uuid.New().String() + "@example.com",
				Role:  users_enums.WorkspaceRoleViewer,
			}

			test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
				"Bearer "+owner.Token,
				request,
				http.StatusOK,
			)

			assert.Len(t, mockEmailSender.SendEmailCalls, 1)
			assert.Equal(t, request.Email, mockEmailSender.SendEmailCalls[0].To)
			assert.Contains(t, mockEmailSender.SendEmailCalls[0].Body, `href="`+databasusURL+`"`)
		})
	}
}

func useMockMembershipEmailSender(t *testing.T) *workspaces_testing.MockEmailSender {
	t.Helper()

	mockEmailSender := workspaces_testing.NewMockEmailSender()
	workspaces_services.GetMembershipService().SetEmailSender(mockEmailSender)
	t.Cleanup(func() {
		workspaces_services.GetMembershipService().SetEmailSender(email.GetInstanceSender())
	})

	return mockEmailSender
}

func useDatabasusURL(t *testing.T, databasusURL string) {
	t.Helper()

	originalDatabasusURL := config.GetEnv().DatabasusURL
	config.GetEnv().DatabasusURL = databasusURL
	t.Cleanup(func() {
		config.GetEnv().DatabasusURL = originalDatabasusURL
	})
}

func Test_AddMemberToWorkspace_WhenWorkspaceAdminTriesToAddAdmin_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspaceAdmin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
		workspace,
		workspaceAdmin,
		users_enums.WorkspaceRoleAdmin,
		router,
	)

	// Workspace admin tries to add another admin (should fail)
	request := workspaces_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.WorkspaceRoleAdmin,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+workspaceAdmin.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "only workspace owner can add/manage admins")
}

func Test_AddMemberToWorkspace_WhenWorkspaceAdminTriesToAddWorkspaceAdmin_ReturnsBadRequest(
	t *testing.T,
) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspaceAdmin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
		workspace,
		workspaceAdmin,
		users_enums.WorkspaceRoleAdmin,
		router,
	)

	// WorkspaceAdmin tries to add another WorkspaceAdmin (should fail)
	request := workspaces_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.WorkspaceRoleAdmin,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+workspaceAdmin.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "only workspace owner can add/manage admins")
}

func Test_AddWorkspaceAdmin_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can add workspace admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can add workspace admin",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin cannot add workspace admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace member cannot add workspace admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace viewer cannot add workspace admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			newMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			request := workspaces_dto.AddMemberRequestDTO{
				Email: newMember.Email,
				Role:  users_enums.WorkspaceRoleAdmin,
			}

			if tt.expectSuccess {
				var response workspaces_dto.AddMemberResponseDTO
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
					&response,
				)

				assert.True(t, response.Status == workspaces_dto.AddStatusAdded)
			} else {
				resp := test_utils.MakePostRequest(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(resp.Body), "only workspace owner can add/manage admins")
			}
		})
	}
}

func Test_InviteMemberToWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can invite member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can invite member",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can invite member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member cannot invite member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace viewer cannot invite member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
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
			users_testing.EnableMemberInvitations(t.Context())
			defer users_testing.ResetSettingsToDefaults(t.Context())

			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			request := workspaces_dto.AddMemberRequestDTO{
				Email: fmt.Sprintf("invite-%s@example.com", uuid.New().String()),
				Role:  users_enums.WorkspaceRoleViewer,
			}

			if tt.expectSuccess {
				var response workspaces_dto.AddMemberResponseDTO
				test_utils.MakePostRequestAndUnmarshal(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
					&response,
				)

				assert.True(t, response.Status == workspaces_dto.AddStatusInvited)
			} else {
				resp := test_utils.MakePostRequest(
					t,
					router,
					"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
					"Bearer "+testUserToken,
					request,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(resp.Body), "insufficient permissions to manage members")
			}
		})
	}
}

// ChangeMemberRole Tests

func Test_ChangeMemberRole_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can change member role",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can change member role",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can change member role",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member cannot change member role",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace viewer cannot change member role",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			targetMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
				workspace,
				targetMember,
				users_enums.WorkspaceRoleViewer,
				router,
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			request := workspaces_dto.ChangeMemberRoleRequestDTO{
				Role: users_enums.WorkspaceRoleViewer,
			}

			resp := test_utils.MakePutRequest(
				t,
				router,
				fmt.Sprintf(
					"/api/v1/workspaces/memberships/%s/members/%s/role",
					workspace.ID.String(),
					targetMember.UserID.String(),
				),
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(resp.Body), "Member role changed successfully")
			} else {
				assert.Contains(t, string(resp.Body), "insufficient permissions to manage members")
			}
		})
	}
}

func Test_ChangeMemberRoleToAdmin_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can promote to admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can promote to admin",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin cannot promote to admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace member cannot promote to admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			targetMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
				workspace,
				targetMember,
				users_enums.WorkspaceRoleViewer,
				router,
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			request := workspaces_dto.ChangeMemberRoleRequestDTO{
				Role: users_enums.WorkspaceRoleAdmin,
			}

			resp := test_utils.MakePutRequest(
				t,
				router,
				fmt.Sprintf(
					"/api/v1/workspaces/memberships/%s/members/%s/role",
					workspace.ID.String(),
					targetMember.UserID.String(),
				),
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(resp.Body), "Member role changed successfully")
			} else {
				assert.Contains(t, string(resp.Body), "only workspace owner can add/manage admins")
			}
		})
	}
}

func Test_ChangeMemberRole_WhenChangingOwnRole_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	request := workspaces_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.WorkspaceRoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf(
			"/api/v1/workspaces/memberships/%s/members/%s/role",
			workspace.ID.String(),
			owner.UserID.String(),
		),
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change your own role")
}

func Test_ChangeMemberRole_WhenChangingOwnerRole_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	request := workspaces_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.WorkspaceRoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf(
			"/api/v1/workspaces/memberships/%s/members/%s/role",
			workspace.ID.String(),
			owner.UserID.String(),
		),
		"Bearer "+admin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change owner role")
}

// RemoveMember Tests

func Test_RemoveMemberFromWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can remove member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can remove member",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can remove member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member cannot remove member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace viewer cannot remove member",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			targetMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
				workspace,
				targetMember,
				users_enums.WorkspaceRoleViewer,
				router,
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
				Method: "DELETE",
				URL: fmt.Sprintf(
					"/api/v1/workspaces/memberships/%s/members/%s",
					workspace.ID.String(),
					targetMember.UserID.String(),
				),
				AuthToken:      "Bearer " + testUserToken,
				ExpectedStatus: tt.expectedStatusCode,
			})

			if tt.expectSuccess {
				assert.Contains(t, string(resp.Body), "Member removed successfully")
			} else {
				assert.Contains(t, string(resp.Body), "insufficient permissions to remove members")
			}
		})
	}
}

func Test_RemoveMemberFromWorkspace_WhenRemovingOwner_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "DELETE",
		URL: fmt.Sprintf(
			"/api/v1/workspaces/memberships/%s/members/%s",
			workspace.ID.String(),
			owner.UserID.String(),
		),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "cannot remove workspace owner, transfer ownership first")
}

func Test_RemoveWorkspaceAdmin_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can remove workspace admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can remove workspace admin",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin cannot remove workspace admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace member cannot remove workspace admin",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			targetAdmin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
				workspace,
				targetAdmin,
				users_enums.WorkspaceRoleAdmin,
				router,
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
				Method: "DELETE",
				URL: fmt.Sprintf(
					"/api/v1/workspaces/memberships/%s/members/%s",
					workspace.ID.String(),
					targetAdmin.UserID.String(),
				),
				AuthToken:      "Bearer " + testUserToken,
				ExpectedStatus: tt.expectedStatusCode,
			})

			if tt.expectSuccess {
				assert.Contains(t, string(resp.Body), "Member removed successfully")
			} else {
				// Workspace admins get specific error about removing admins
				// Members/viewers get generic insufficient permissions error
				body := string(resp.Body)
				assert.True(t,
					strings.Contains(body, "only workspace owner can remove admins") ||
						strings.Contains(body, "insufficient permissions to remove members"),
					"Expected permission error, got: %s", body)
			}
		})
	}
}

// TransferOwnership Tests

func Test_TransferWorkspaceOwnership_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		requesterRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can transfer ownership",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can transfer ownership",
			requesterRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member cannot transfer ownership",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "workspace admin cannot transfer ownership",
			requesterRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
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
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			newOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
				"Test Workspace",
				owner,
				router,
			)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
				workspace,
				newOwner,
				users_enums.WorkspaceRoleMember,
				router,
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.requesterRole != nil && *tt.requesterRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.requesterRole != nil {
				requester := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
					workspace,
					requester,
					*tt.requesterRole,
					router,
				)
				testUserToken = requester.Token
			}

			request := workspaces_dto.TransferOwnershipRequestDTO{
				NewOwnerEmail: newOwner.Email,
			}

			resp := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/transfer-ownership",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(resp.Body), "Ownership transferred successfully")
			} else {
				assert.Contains(
					t,
					string(resp.Body),
					"only workspace owner or admin can transfer ownership",
				)
			}
		})
	}
}

func Test_TransferWorkspaceOwnership_WhenNewOwnerIsNotMember_ReturnsBadRequest(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	request := workspaces_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: nonMember.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/transfer-ownership",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "new owner must be a workspace member")
}

func Test_TransferWorkspaceOwnership_ThereIsOnlyOneOwner_OldOwnerBecomeAdmin(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	workspaces_testing.AddMemberToWorkspaceViaOwner(t.Context(),
		workspace,
		member,
		users_enums.WorkspaceRoleMember,
		router,
	)

	// Transfer ownership to the member
	request := workspaces_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: member.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/transfer-ownership",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Ownership transferred successfully")

	// Get all members using the new owner's token
	var membersResponse workspaces_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+member.Token,
		http.StatusOK,
		&membersResponse,
	)

	// Verify there is only one owner
	ownerCount := 0
	var currentOwner *workspaces_dto.WorkspaceMemberResponseDTO
	for _, m := range membersResponse.Members {
		if m.Role == users_enums.WorkspaceRoleOwner {
			ownerCount++
			currentOwner = &m
		}
	}

	assert.Equal(t, 1, ownerCount, "There should be exactly one owner")
	assert.NotNil(t, currentOwner, "Owner should exist")
	assert.Equal(
		t,
		member.UserID,
		currentOwner.UserID,
		"The new owner should be the member we transferred to",
	)
	assert.Equal(
		t,
		member.Email,
		currentOwner.Email,
		"Owner email should match the transferred member",
	)

	// verify previous owner is now an admin
	for _, m := range membersResponse.Members {
		if m.UserID == owner.UserID {
			assert.Equal(
				t,
				users_enums.WorkspaceRoleAdmin,
				m.Role,
				"Previous owner should now be admin",
			)
			break
		}
	}
}
