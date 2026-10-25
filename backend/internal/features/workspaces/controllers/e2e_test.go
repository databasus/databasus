package workspaces_controllers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_WorkspaceLifecycleE2E_CompletesSuccessfully(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)
	users_testing.EnableMemberWorkspaceCreation(t.Context())
	defer users_testing.ResetSettingsToDefaults(t.Context())

	// 1. Create workspace owner
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// 2. Owner creates workspace
	createRequest := workspaces_dto.CreateWorkspaceRequestDTO{
		Name: "E2E Test Workspace",
	}

	var workspaceResponse workspaces_dto.WorkspaceResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces",
		"Bearer "+owner.Token,
		createRequest,
		http.StatusOK,
		&workspaceResponse,
	)

	assert.Equal(t, "E2E Test Workspace", workspaceResponse.Name)
	assert.Equal(t, users_enums.WorkspaceRoleOwner, *workspaceResponse.UserRole)
	workspaceID := workspaceResponse.ID

	// 3. Owner invites a new user
	inviteRequest := workspaces_dto.AddMemberRequestDTO{
		Email: "invited" + uuid.New().String() + "@example.com",
		Role:  users_enums.WorkspaceRoleViewer,
	}

	var inviteResponse workspaces_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspaceID.String()+"/members",
		"Bearer "+owner.Token,
		inviteRequest,
		http.StatusOK,
		&inviteResponse,
	)

	assert.True(t, inviteResponse.Status == workspaces_dto.AddStatusInvited)

	// 4. Add existing user to workspace
	existingMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	addMemberRequest := workspaces_dto.AddMemberRequestDTO{
		Email: existingMember.Email,
		Role:  users_enums.WorkspaceRoleViewer,
	}

	var addMemberResponse workspaces_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspaceID.String()+"/members",
		"Bearer "+owner.Token,
		addMemberRequest,
		http.StatusOK,
		&addMemberResponse,
	)

	assert.True(t, addMemberResponse.Status == workspaces_dto.AddStatusAdded)

	// 5. List workspace members
	var membersResponse workspaces_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspaceID.String()+"/members",
		"Bearer "+owner.Token,
		http.StatusOK,
		&membersResponse,
	)

	assert.GreaterOrEqual(t, len(membersResponse.Members), 2) // owner + added member

	roles := make([]users_enums.WorkspaceRole, len(membersResponse.Members))
	for i, m := range membersResponse.Members {
		roles[i] = m.Role
	}
	assert.Contains(t, roles, users_enums.WorkspaceRoleOwner)
	assert.Contains(t, roles, users_enums.WorkspaceRoleViewer)

	// 6. Promote member to admin
	promoteRequest := workspaces_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.WorkspaceRoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspaceID.String()+"/members/"+existingMember.UserID.String()+"/role",
		"Bearer "+owner.Token,
		promoteRequest,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Member role changed successfully")

	// 7. Update workspace settings
	updateRequest := workspaces_models.Workspace{
		Name: "Updated E2E Workspace",
	}

	var updateResponse workspaces_models.Workspace
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/"+workspaceID.String(),
		"Bearer "+owner.Token,
		updateRequest,
		http.StatusOK,
		&updateResponse,
	)

	assert.Equal(t, "Updated E2E Workspace", updateResponse.Name)

	// 8. Transfer ownership
	transferRequest := workspaces_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: existingMember.Email,
	}

	resp = test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces/memberships/"+workspaceID.String()+"/transfer-ownership",
		"Bearer "+owner.Token,
		transferRequest,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Ownership transferred successfully")

	// 9. New owner can now manage workspace
	var finalWorkspace workspaces_models.Workspace
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/"+workspaceID.String(),
		"Bearer "+existingMember.Token,
		http.StatusOK,
		&finalWorkspace,
	)

	assert.Equal(t, workspaceID, finalWorkspace.ID)
	assert.Equal(t, "Updated E2E Workspace", finalWorkspace.Name)

	// 10. New owner can delete workspace
	resp = test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/workspaces/" + workspaceID.String(),
		AuthToken:      "Bearer " + existingMember.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Workspace deleted successfully")
}

func Test_AdminWorkspaceManagementE2E_CompletesSuccessfully(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		GetWorkspaceController(),
		GetMembershipController(),
	)

	// 1. Create admin and regular user
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	regularUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// 2. Regular user creates workspace (with member creation disabled)
	users_testing.DisableMemberWorkspaceCreation(t.Context())
	defer users_testing.ResetSettingsToDefaults(t.Context())

	// Regular user cannot create workspace
	createRequest := workspaces_dto.CreateWorkspaceRequestDTO{
		Name: "Regular User Workspace",
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/workspaces",
		"Bearer "+regularUser.Token,
		createRequest,
		http.StatusForbidden,
	)

	// 3. Admin can create workspace regardless of settings
	adminCreateRequest := workspaces_dto.CreateWorkspaceRequestDTO{
		Name: "Admin Workspace",
	}

	var adminWorkspaceResponse workspaces_dto.WorkspaceResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces",
		"Bearer "+admin.Token,
		adminCreateRequest,
		http.StatusOK,
		&adminWorkspaceResponse,
	)

	assert.Equal(t, "Admin Workspace", adminWorkspaceResponse.Name)
	adminWorkspaceID := adminWorkspaceResponse.ID

	// 4. Admin can view any workspace (even not a member)
	regularUser2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	users_testing.EnableMemberWorkspaceCreation(t.Context())

	regularUserCreateRequest := workspaces_dto.CreateWorkspaceRequestDTO{
		Name: "Regular User Workspace 2",
	}

	var regularWorkspaceResponse workspaces_dto.WorkspaceResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces",
		"Bearer "+regularUser2.Token,
		regularUserCreateRequest,
		http.StatusOK,
		&regularWorkspaceResponse,
	)

	regularWorkspaceID := regularWorkspaceResponse.ID

	// Admin can view regular user's workspace
	var adminViewResponse workspaces_models.Workspace
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/workspaces/"+regularWorkspaceID.String(),
		"Bearer "+admin.Token,
		http.StatusOK,
		&adminViewResponse,
	)

	assert.Equal(t, regularWorkspaceID, adminViewResponse.ID)

	// 5. Admin can delete any workspace
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/workspaces/" + regularWorkspaceID.String(),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Workspace deleted successfully")

	// 6. Clean up admin's workspace
	test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/workspaces/" + adminWorkspaceID.String(),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusOK,
	})
}
