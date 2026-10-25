package users_controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/audit_logs"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_GetUsersList_WhenUserIsAdmin_ReturnsUsers(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var response users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.NotNil(t, response.Users)
	assert.GreaterOrEqual(t, response.Total, int64(1)) // At least the test user should exist
}

func Test_GetUsersList_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users",
		"Bearer "+testUser.Token,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "permissions")
}

func Test_GetUsersList_WithPagination_RespectsLimits(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var response users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users?limit=5&offset=0",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.NotNil(t, response.Users)
	assert.LessOrEqual(t, len(response.Users), 5) // Should respect limit
}

func Test_GetUsersList_WithBeforeDateFilter_ReturnsFilteredUsers(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Test with beforeDate filter
	beforeDate := "2024-01-01T00:00:00Z"
	var response users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users?beforeDate="+beforeDate,
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.NotNil(t, response.Users)
	// All returned users should have been created before the specified date
	for _, user := range response.Users {
		assert.True(t, user.CreatedAt.Before(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
	}
}

func Test_GetUsersList_WithInvalidDateFilter_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Test with invalid date format
	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users?beforeDate=invalid-date",
		"Bearer "+testUser.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid query parameters")
}

func Test_GetUsersList_WithSearchQuery_ReturnsFilteredUsers(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Create test users with specific emails and names
	user1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// Test searching by email (partial match)
	emailPart := user1.Email[:5]
	var emailResponse users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users?query="+emailPart,
		"Bearer "+adminUser.Token,
		http.StatusOK,
		&emailResponse,
	)

	assert.NotNil(t, emailResponse.Users)
	found := false
	for _, u := range emailResponse.Users {
		if u.ID == user1.UserID {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected user1 to be in search results")

	// Test case-insensitive search
	upperEmailPart := user2.Email[:5]
	var caseInsensitiveResponse users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users?query="+upperEmailPart,
		"Bearer "+adminUser.Token,
		http.StatusOK,
		&caseInsensitiveResponse,
	)

	assert.NotNil(t, caseInsensitiveResponse.Users)
	found = false
	for _, u := range caseInsensitiveResponse.Users {
		if u.ID == user2.UserID {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected user2 to be in case-insensitive search results")

	// Test searching by name
	var nameResponse users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users?query=Test",
		"Bearer "+adminUser.Token,
		http.StatusOK,
		&nameResponse,
	)

	assert.NotNil(t, nameResponse.Users)
	assert.GreaterOrEqual(t, len(nameResponse.Users), 1, "Should find users with 'Test' in name")
}

func Test_GetUsersList_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	router := createManagementTestRouter()

	test_utils.MakeGetRequest(t, router, "/api/v1/users", "", http.StatusUnauthorized)
}

func Test_GetUserProfile_WhenAccessingOwnProfile_ReturnsProfile(t *testing.T) {
	router := createManagementTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	var response users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/"+testUser.UserID.String(),
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, testUser.UserID, response.ID)
	assert.Equal(t, users.RoleMember, response.Role)
}

func Test_GetUserProfile_WhenUserIsAdmin_ReturnsProfile(t *testing.T) {
	router := createManagementTestRouter()

	// Create both admin and regular user
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	regularUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	var response users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/"+regularUser.UserID.String(),
		"Bearer "+adminUser.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, regularUser.UserID, response.ID)
}

func Test_GetUserProfile_WhenAccessingOtherUserAsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String(),
		"Bearer "+user1.Token,
		http.StatusForbidden,
	)
}

func Test_GetUserProfile_WithNonExistentUser_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Try to access non-existent user
	test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/00000000-0000-0000-0000-000000000000",
		"Bearer "+testUser.Token,
		http.StatusForbidden,
	)
}

func Test_GetUserProfile_WithInvalidUserID_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/invalid-uuid",
		"Bearer "+testUser.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid user ID")
}

func Test_DeactivateUser_WhenUserIsAdmin_UserDeactivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/deactivate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User deactivated successfully")
}

func Test_DeactivateUser_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String()+"/deactivate",
		"Bearer "+user1.Token,
		nil,
		http.StatusForbidden,
	)
}

func Test_DeactivateUser_WhenDeactivatingOwnAccount_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminUser.UserID.String()+"/deactivate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot deactivate your own account")
}

func Test_ActivateUser_WhenUserIsAdmin_UserActivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// First deactivate the user
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/deactivate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusOK,
	)

	// Now activate the user
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/activate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User activated successfully")
}

func Test_ActivateUser_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String()+"/activate",
		"Bearer "+user1.Token,
		nil,
		http.StatusForbidden,
	)
}

func Test_ChangeUserRole_WhenUserIsRootAdmin_RoleChanged(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and target user
	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())
	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User role changed successfully")
}

func Test_ChangeUserRole_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleAdmin,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String()+"/role",
		"Bearer "+user1.Token,
		request,
		http.StatusForbidden,
	)
}

func Test_ChangeUserRole_WhenChangingOwnRole_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+adminUser.UserID.String()+"/role",
		"Bearer "+adminUser.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change your own role")
}

func Test_ChangeUserRole_WithInvalidRole_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// Test with invalid JSON structure containing invalid role
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/" + targetUser.UserID.String() + "/role",
		Body:           map[string]string{"role": "INVALID_ROLE"},
		AuthToken:      "Bearer " + adminUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.NotEmpty(t, resp.Body)
}

func Test_ChangeUserRole_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/" + targetUser.UserID.String() + "/role",
		Body:           "invalid json",
		AuthToken:      "Bearer " + adminUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

// Tests for root admin restrictions
func Test_ChangeUserRole_WhenRegularAdminPromotesToAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and target user
	regularAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/role",
		"Bearer "+regularAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(
		t,
		string(resp.Body),
		"only the root admin user can promote users to admin or demote admin users",
	)
}

func Test_ChangeUserRole_WhenRegularAdminDemotesAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and admin target user
	regularAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	adminTargetUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/role",
		"Bearer "+regularAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(
		t,
		string(resp.Body),
		"only the root admin user can promote users to admin or demote admin users",
	)
}

func Test_DeactivateUser_WhenRegularAdminDeactivatesAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and admin target user
	regularAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	adminTargetUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/deactivate",
		"Bearer "+regularAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only the root admin user can deactivate admin accounts")
}

func Test_ActivateUser_WhenRegularAdminActivatesAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and admin target user
	regularAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	adminTargetUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/activate",
		"Bearer "+regularAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only the root admin user can activate admin accounts")
}

func Test_ChangeUserRole_WhenRootAdminPromotesToAdmin_RoleChanged(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and target user
	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())
	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User role changed successfully")
}

func Test_ChangeUserRole_WhenRootAdminDemotesAdmin_RoleChanged(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and admin target user
	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())
	adminTargetUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User role changed successfully")
}

func Test_DeactivateUser_WhenRootAdminDeactivatesAdmin_UserDeactivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and admin target user
	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())
	adminTargetUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User deactivated successfully")
}

func Test_ActivateUser_WhenRootAdminActivatesAdmin_UserActivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and admin target user
	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())
	adminTargetUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// First deactivate the admin user
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)

	// Now activate the admin user
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/activate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User activated successfully")
}

func Test_ChangeUserRole_WhenRootAdminChangesOwnRole_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin
	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users.RoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change your own role")
}

func Test_DeactivateUser_WhenRootAdminDeactivatesOwnAccount_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin
	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot deactivate your own account")
}

func Test_InviteUserToWorkspace_MembershipReceivedAfterSignUp(t *testing.T) {
	router := createInviteWorkspaceTestRouter()
	users_testing.EnableMemberInvitations(t.Context())
	users_testing.EnableExternalRegistrations(t.Context())
	defer users_testing.ResetSettingsToDefaults(t.Context())

	// 1. Create workspace owner and workspace
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace, _ := workspaces_testing.CreateTestWorkspaceViaAPI(t.Context(),
		"Invite Test Workspace",
		owner,
		router,
	)

	// 2. Invite non-existing user to workspace
	inviteEmail := "invited-" + uuid.New().String() + "@example.com"
	inviteResponse := workspaces_testing.InviteMemberToWorkspace(
		workspace,
		inviteEmail,
		users_enums.WorkspaceRoleMember,
		owner.Token,
		router,
	)

	assert.Equal(t, workspaces_dto.AddStatusInvited, inviteResponse.Status)

	// 3. Sign up the invited user (now returns token directly)
	signUpRequest := users_dto.SignUpRequestDTO{
		Email:    inviteEmail,
		Password: "testpassword123",
		Name:     "Invited User",
	}

	var signInResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signup",
		"",
		signUpRequest,
		http.StatusOK,
		&signInResponse,
	)

	assert.NotEmpty(t, signInResponse.Token)
	assert.Equal(t, inviteEmail, signInResponse.Email)

	// 4. Verify user is automatically added as member to workspace
	var membersResponse workspaces_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspace.ID.String()+"/members",
		"Bearer "+signInResponse.Token,
		http.StatusOK,
		&membersResponse,
	)

	var foundMember *workspaces_dto.WorkspaceMemberResponseDTO
	for _, member := range membersResponse.Members {
		if member.UserID == signInResponse.UserID {
			foundMember = &member
			break
		}
	}

	assert.NotNil(
		t,
		foundMember,
		"Invited user should be automatically added as workspace member after sign up",
	)
	assert.Equal(t, users_enums.WorkspaceRoleMember, foundMember.Role)
	assert.Equal(t, inviteEmail, foundMember.Email)
}

func createInviteWorkspaceTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	GetUserController().RegisterRoutes(v1)

	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	GetManagementController().RegisterRoutes(protected.(*gin.RouterGroup))
	GetUserController().RegisterProtectedRoutes(protected.(*gin.RouterGroup))
	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected.(*gin.RouterGroup))
	workspaces_controllers.GetMembershipController().RegisterRoutes(protected.(*gin.RouterGroup))
	audit_logs.SetupDependencies()

	return router
}

func Test_ManageAdmins_WhenBootstrapAdminHasChangedAddress_AllThreeActionsSucceed(t *testing.T) {
	// The management router is built last: it installs the recorder on all three
	// services, and the user router would otherwise leave the management service
	// writing into an earlier one.
	userRouter := createUserTestRouter()
	router := createManagementTestRouter()
	auditLogRecorder := users_testing.GetAuditLogRecorder()

	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())

	changedEmail := "moved" + uuid.New().String() + "@example.com"
	test_utils.MakePutRequest(
		t,
		userRouter,
		"/api/v1/users/me",
		"Bearer "+rootAdmin.Token,
		users_dto.UpdateUserInfoRequestDTO{Email: &changedEmail},
		http.StatusOK,
	)

	targetUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	response := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		users_dto.ChangeUserRoleRequestDTO{Role: users.RoleAdmin},
		http.StatusOK,
	)
	assert.Contains(t, string(response.Body), "User role changed successfully")

	response = test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(response.Body), "User deactivated successfully")

	response = test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/activate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(response.Body), "User activated successfully")

	assert.True(t, auditLogRecorder.HasEntryContaining("User role changed", targetUser.Email))
	assert.True(t, auditLogRecorder.HasEntryContaining("User deactivated", targetUser.Email))
}

// The property here holds by composition of three separate rules and nothing in
// the code states it, so an edit to any one of them could remove it silently.
func Test_ManageAdmins_AgainstTheBootstrapAdmin_IsAlwaysRefused(t *testing.T) {
	router := createManagementTestRouter()

	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())

	recordedRootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, recordedRootAdmin)
	require.Equal(t, rootAdmin.UserID, recordedRootAdmin.ID)

	ordinaryAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	response := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/role",
		"Bearer "+ordinaryAdmin.Token,
		users_dto.ChangeUserRoleRequestDTO{Role: users.RoleMember},
		http.StatusBadRequest,
	)
	assert.Contains(t, string(response.Body), "only the root admin user")

	response = test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/deactivate",
		"Bearer "+ordinaryAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(response.Body), "only the root admin user")

	response = test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		users_dto.ChangeUserRoleRequestDTO{Role: users.RoleMember},
		http.StatusBadRequest,
	)
	assert.Contains(t, string(response.Body), "cannot change your own role")

	response = test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(response.Body), "cannot deactivate your own account")

	unchangedRootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, unchangedRootAdmin)
	assert.Equal(t, rootAdmin.UserID, unchangedRootAdmin.ID)
	assert.Equal(t, users.RoleAdmin, unchangedRootAdmin.Role)
	assert.Equal(t, users_enums.UserStatusActive, unchangedRootAdmin.Status)
}
