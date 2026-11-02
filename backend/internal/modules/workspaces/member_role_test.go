package workspaces_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestChangeWorkspaceMemberRole(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 and keeps the owner when an admin makes a member the owner", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		_, viewerMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response := changeRole(t, workspaceAdmin, viewerMember.Id, apiclient.AssignableWorkspaceRole("WORKSPACE_OWNER"))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		listedMembers := listAllMembers(t, tm.owner.Client, tm.workspace.Id)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, findMemberOfUser(t, listedMembers, tm.owner.UserID).Role)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEVIEWER,
			findMemberOfUser(t, listedMembers, viewerMember.UserId).Role)
	})

	t.Run("returns 409 and keeps the owner's role when an instance admin demotes the owner", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		ownerMember := tm.getOwnerMember(t)

		response := changeRole(t, tm.admin, ownerMember.Id, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeOwnerRequiresTransfer)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, tm.getOwnerMember(t).Role)
	})

	t.Run("returns 403 and keeps the role when an admin demotes another admin", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		_, otherAdminMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := changeRole(t, workspaceAdmin, otherAdminMember.Id, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEADMIN, getMemberRole(t, tm, otherAdminMember.UserId))
	})

	t.Run("returns 403 and keeps the role when an admin promotes a member to admin", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		_, memberOfTeam := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		response := changeRole(t, workspaceAdmin, memberOfTeam.Id, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEMEMBER, getMemberRole(t, tm, memberOfTeam.UserId))
	})

	t.Run("makes a member an admin when the owner promotes them", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		_, memberOfTeam := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		response := changeRole(t, tm.owner, memberOfTeam.Id, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEADMIN, response.JSON200.Role)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEADMIN, getMemberRole(t, tm, memberOfTeam.UserId))
	})

	t.Run("lets an admin change the role of a viewer", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		_, viewerMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response := changeRole(t, workspaceAdmin, viewerMember.Id, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEMEMBER, getMemberRole(t, tm, viewerMember.UserId))
	})

	t.Run("returns 409 and keeps the role when the owner changes their own role", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		ownerMember := tm.getOwnerMember(t)

		response := changeRole(t, tm.owner, ownerMember.Id, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeCannotChangeOwnRole)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, tm.getOwnerMember(t).Role)
	})

	t.Run("returns 403 and keeps the role when an admin demotes themselves", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, adminMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := changeRole(t, workspaceAdmin, adminMember.Id, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEADMIN, getMemberRole(t, tm, adminMember.UserId))
	})

	t.Run("returns 404 and keeps the role when the actor is not a member of the workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		_, viewerMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		stranger := tm.app.CreateNonAdminUser(t, tm.admin)

		response := changeRole(t, stranger, viewerMember.Id, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEVIEWER, getMemberRole(t, tm, viewerMember.UserId))
	})

	t.Run("returns 404 when the member does not exist", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response := changeRole(t, tm.owner, uuid.New(), apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

func TestRemoveWorkspaceMember(t *testing.T) {
	t.Parallel()

	t.Run("returns 409 and keeps the owner when an instance admin removes the owner", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		ownerMember := tm.getOwnerMember(t)

		response := removeMember(t, tm.admin, ownerMember.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeOwnerRequiresTransfer)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, tm.getOwnerMember(t).Role)
	})

	t.Run("returns 403 and keeps the admin when an admin removes another admin", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		_, otherAdminMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := removeMember(t, workspaceAdmin, otherAdminMember.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEADMIN, getMemberRole(t, tm, otherAdminMember.UserId))
	})

	t.Run("ends the removed user's access and keeps their account", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		removedUser, removedMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		response := removeMember(t, workspaceAdmin, removedMember.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		readResponse := getWorkspace(t, removedUser, tm.workspace.Id)
		requireProblem(t, readResponse.StatusCode(), readResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		signInResponse, err := tm.app.NewClient(t).SignInWithResponse(t.Context(), apiclient.SignInRequest{
			Email:                    removedUser.Email,
			Password:                 new(apptest.TestPassword),
			CloudflareTurnstileToken: nil,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, signInResponse.StatusCode(), string(signInResponse.Body))
	})

	t.Run("returns 403 and keeps the member when a member removes a viewer", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		memberOfTeam, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		_, viewerMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response := removeMember(t, memberOfTeam, viewerMember.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Len(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), 3)
	})
}

func TestTransferWorkspaceOwnership(t *testing.T) {
	t.Parallel()

	t.Run("makes the admin the owner and the previous owner an admin", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		_, adminMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := transferOwnership(t, tm.owner, tm.workspace.Id, adminMember.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		listedMembers := listAllMembers(t, tm.owner.Client, tm.workspace.Id)
		assert.Equal(
			t,
			apiclient.WorkspaceRoleWORKSPACEOWNER,
			findMemberOfUser(t, listedMembers, adminMember.UserId).Role,
		)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEADMIN, findMemberOfUser(t, listedMembers, tm.owner.UserID).Role)
		assert.Equal(t, 1, countOwners(listedMembers))
	})

	t.Run("changes nothing when the owner names their own member", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		_, viewerMember := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		membersBefore := listAllMembers(t, tm.owner.Client, tm.workspace.Id)

		response := transferOwnership(t, tm.owner, tm.workspace.Id, tm.getOwnerMember(t).Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Equal(t, membersBefore, listAllMembers(t, tm.owner.Client, tm.workspace.Id))
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEVIEWER, getMemberRole(t, tm, viewerMember.UserId))
	})

	t.Run("returns 403 and keeps the owner when an admin transfers ownership", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		_, memberOfTeam := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		response := transferOwnership(t, workspaceAdmin, tm.workspace.Id, memberOfTeam.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, tm.getOwnerMember(t).Role)
	})

	t.Run("returns 404 and keeps the owner when the member belongs to another workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Other")
		otherUser := tm.app.SignUp(t, apptest.NewTestEmail("other"))
		otherMember := apptest.AddWorkspaceMember(t, tm.owner, otherWorkspace.Id, otherUser.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := transferOwnership(t, tm.owner, tm.workspace.Id, otherMember.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, tm.getOwnerMember(t).Role)
	})
}

func changeRole(
	t *testing.T,
	actor apptest.Session,
	memberID uuid.UUID,
	role apiclient.AssignableWorkspaceRole,
) *apiclient.ChangeWorkspaceMemberRoleResponse {
	t.Helper()

	response, err := actor.Client.ChangeWorkspaceMemberRoleWithResponse(t.Context(), memberID,
		apiclient.WorkspaceMemberRoleChange{Role: role})
	require.NoError(t, err)

	return response
}

func removeMember(t *testing.T, actor apptest.Session, memberID uuid.UUID) *apiclient.RemoveWorkspaceMemberResponse {
	t.Helper()

	response, err := actor.Client.RemoveWorkspaceMemberWithResponse(t.Context(), memberID)
	require.NoError(t, err)

	return response
}

func transferOwnership(
	t *testing.T,
	actor apptest.Session,
	workspaceID uuid.UUID,
	newOwnerMemberID uuid.UUID,
) *apiclient.TransferWorkspaceOwnershipResponse {
	t.Helper()

	response, err := actor.Client.TransferWorkspaceOwnershipWithResponse(t.Context(), workspaceID,
		apiclient.OwnershipTransfer{NewOwnerMemberId: newOwnerMemberID})
	require.NoError(t, err)

	return response
}

func getMemberRole(t *testing.T, tm team, userID uuid.UUID) apiclient.WorkspaceRole {
	t.Helper()

	return findMemberOfUser(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), userID).Role
}
