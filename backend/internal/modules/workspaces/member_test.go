package workspaces_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestAddWorkspaceMember(t *testing.T) {
	t.Parallel()

	t.Run("adds an existing user with the requested role", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		user := tm.app.SignUp(t, apptest.NewTestEmail("user"))

		response := addMember(
			t,
			workspaceAdmin,
			tm.workspace.Id,
			user.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		)

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.AddedWorkspaceMemberOutcomeADDED, response.JSON201.Outcome)
		assert.False(t, response.JSON201.IsInvited)
		addedMember := findMemberOfUser(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), user.UserID)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEMEMBER, addedMember.Role)
		assert.Equal(t, response.JSON201.Id, addedMember.Id)
	})

	t.Run("returns 409 and keeps the role when the user is already a member", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response := addMember(
			t,
			tm.owner,
			tm.workspace.Id,
			viewer.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeAlreadyMember)
		viewerMember := findMemberOfUser(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), viewer.UserID)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEVIEWER, viewerMember.Role)
	})

	t.Run("invites an address without a user when the instance allows member invitations", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		invitedEmail := apptest.NewTestEmail("invited")

		response := addMember(
			t,
			workspaceAdmin,
			tm.workspace.Id,
			invitedEmail,
			apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
		)

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.AddedWorkspaceMemberOutcomeINVITED, response.JSON201.Outcome)
		invitedMember := findMemberOfUser(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id),
			response.JSON201.UserId)
		assert.True(t, invitedMember.IsInvited)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEVIEWER, invitedMember.Role)
		assert.Len(t, tm.app.Mail.GetMessagesTo(invitedEmail), 1)
	})

	t.Run("returns 403 and invites nobody when the instance does not allow member invitations", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		closeMemberInvitations(t, tm.admin)
		invitedEmail := apptest.NewTestEmail("invited")

		response := addMember(
			t,
			tm.owner,
			tm.workspace.Id,
			invitedEmail,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, 0, countUsersWithEmail(t, tm.app, invitedEmail))
		assert.Empty(t, tm.app.Mail.GetMessagesTo(invitedEmail))
	})

	t.Run("returns 400 and adds nobody when the role is the owner role", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		user := tm.app.SignUp(t, apptest.NewTestEmail("user"))

		response := addMember(
			t,
			tm.owner,
			tm.workspace.Id,
			user.Email,
			apiclient.AssignableWorkspaceRole("WORKSPACE_OWNER"),
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.Len(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), 1)
	})

	roleCases := []struct {
		name string
		role apiclient.AssignableWorkspaceRole
	}{
		{
			name: "returns 403 and adds nobody when a member adds a member",
			role: apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		},
		{
			name: "returns 403 and adds nobody when a viewer adds a member",
			role: apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
		},
	}
	for _, roleCase := range roleCases {
		t.Run(roleCase.name, func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			lowRoleUser, _ := tm.addMember(t, roleCase.role)
			user := tm.app.SignUp(t, apptest.NewTestEmail("user"))

			response := addMember(
				t,
				lowRoleUser,
				tm.workspace.Id,
				user.Email,
				apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
			)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Len(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), 2)
		})
	}

	t.Run("returns 403 when an admin adds a user as an admin", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		user := tm.app.SignUp(t, apptest.NewTestEmail("user"))

		response := addMember(
			t,
			workspaceAdmin,
			tm.workspace.Id,
			user.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEADMIN,
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("returns 404 when the user is not a member of the workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.CreateNonAdminUser(t, tm.admin)

		response := addMember(
			t,
			stranger,
			tm.workspace.Id,
			stranger.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("keeps the invited member when the mail server refuses the invitation email", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		tm.app.Mail.RefuseSends()
		invitedEmail := apptest.NewTestEmail("invited")

		response := addMember(
			t,
			tm.owner,
			tm.workspace.Id,
			invitedEmail,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		)

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.AddedWorkspaceMemberOutcomeINVITED, response.JSON201.Outcome)
		invitedMember := findMemberOfUser(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id),
			response.JSON201.UserId)
		assert.True(t, invitedMember.IsInvited)
		assert.Equal(t, invitedEmail, invitedMember.Email)
		assert.Len(t, tm.app.Mail.GetRefusedMessagesTo(invitedEmail), 1)
	})

	t.Run("returns 400 and adds nobody when the address is not an email address", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		status, body := sendRawRequest(t, tm.app, tm.owner, http.MethodPost, "/workspace-members",
			`{"workspaceId":"`+tm.workspace.Id.String()+`","email":"not-an-address","role":"WORKSPACE_MEMBER"}`)

		requireProblem(t, status, body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
		assert.Len(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), 1)
	})

	t.Run("leaves no user, member or email when adding the member fails after the invitation", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		refuseMemberInserts(t, tm.app)
		invitedEmail := apptest.NewTestEmail("invited")

		response := addMember(
			t,
			tm.owner,
			tm.workspace.Id,
			invitedEmail,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusInternalServerError,
			apiclient.ProblemCodeInternal)
		assert.Equal(t, 0, countUsersWithEmail(t, tm.app, invitedEmail))
		assert.Len(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), 1)
		assert.Empty(t, tm.app.Mail.GetMessagesTo(invitedEmail))
	})
}

func TestInvitationEmail(t *testing.T) {
	t.Parallel()

	publicURLs := []string{"https://databasus.example.com", "https://databasus.example.com/"}
	for _, publicURL := range publicURLs {
		t.Run("links to the public address as configured and names the inviter with "+publicURL,
			func(t *testing.T) {
				t.Parallel()

				app := apptest.New(t, apptest.WithPublicURL(publicURL))
				admin := app.CreateAdmin(t)
				renameUser(t, admin, "Grace Inviter")
				workspace := apptest.CreateWorkspace(t, admin, "Invitations")
				invitedEmail := apptest.NewTestEmail("invited")

				response := addMember(t, admin, workspace.Id, invitedEmail,
					apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

				require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
				invitationEmails := app.Mail.GetMessagesTo(invitedEmail)
				require.Len(t, invitationEmails, 1)
				assert.Contains(t, invitationEmails[0].HTMLBody, `href="`+publicURL+`"`)
				assert.Contains(t, invitationEmails[0].HTMLBody, "Grace Inviter")
			})
	}
}

func TestListWorkspaceMembers(t *testing.T) {
	t.Parallel()

	t.Run("lists an invited member as invited after the members who joined earlier", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
		invitedEmail := apptest.NewTestEmail("invited")
		tm.app.Clock.Advance(time.Second)
		addResponse := addMember(t, workspaceAdmin, tm.workspace.Id, invitedEmail,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		require.Equal(t, http.StatusCreated, addResponse.StatusCode(), string(addResponse.Body))

		listedMembers := listAllMembers(t, workspaceAdmin.Client, tm.workspace.Id)

		require.Len(t, listedMembers, 3)
		assert.Equal(t, tm.owner.UserID, listedMembers[0].UserId)
		assert.Equal(t, workspaceAdmin.UserID, listedMembers[1].UserId)
		assert.Equal(t, invitedEmail, listedMembers[2].Email)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEMEMBER, listedMembers[2].Role)
		assert.True(t, listedMembers[2].IsInvited)
		assert.False(t, listedMembers[0].IsInvited)
	})

	t.Run("lists every member once in the order they joined when following the cursor", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		joinedUserIDs := []uuid.UUID{tm.owner.UserID}
		for range 4 {
			addedUser, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			joinedUserIDs = append(joinedUserIDs, addedUser.UserID)
		}

		var listedUserIDs []uuid.UUID

		var after *string

		for {
			response, err := tm.owner.Client.ListWorkspaceMembersWithResponse(t.Context(),
				&apiclient.ListWorkspaceMembersParams{WorkspaceId: tm.workspace.Id, Limit: new(2), After: after})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			require.LessOrEqual(t, len(response.JSON200.Items), 2)

			for _, listedMember := range response.JSON200.Items {
				listedUserIDs = append(listedUserIDs, listedMember.UserId)
			}

			if response.JSON200.NextCursor == nil {
				break
			}

			after = response.JSON200.NextCursor
		}

		assert.Equal(t, joinedUserIDs, listedUserIDs)
	})

	t.Run("returns 400 when the cursor is not one of this list", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.owner.Client.ListWorkspaceMembersWithResponse(t.Context(),
			&apiclient.ListWorkspaceMembersParams{WorkspaceId: tm.workspace.Id, Limit: nil, After: new("not-a-cursor")})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})

	t.Run("returns 404 when the user is not a member of the workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.CreateNonAdminUser(t, tm.admin)

		response, err := stranger.Client.ListWorkspaceMembersWithResponse(t.Context(),
			&apiclient.ListWorkspaceMembersParams{WorkspaceId: tm.workspace.Id, Limit: nil, After: nil})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

func addMember(
	t *testing.T,
	actor apptest.Session,
	workspaceID uuid.UUID,
	email string,
	role apiclient.AssignableWorkspaceRole,
) *apiclient.AddWorkspaceMemberResponse {
	t.Helper()

	response, err := actor.Client.AddWorkspaceMemberWithResponse(t.Context(), apiclient.WorkspaceMemberAddition{
		WorkspaceId: workspaceID,
		Email:       openapi_types.Email(email),
		Role:        role,
	})
	require.NoError(t, err)

	return response
}

func closeMemberInvitations(t *testing.T, admin apptest.Session) {
	t.Helper()

	response, err := admin.Client.UpdateInstanceSettingsWithResponse(t.Context(), apiclient.InstanceSettingsUpdate{
		IsAllowExternalRegistrations:      true,
		IsAllowMemberInvitations:          false,
		IsMemberAllowedToCreateWorkspaces: true,
		IsTwoFactorAuthRequired:           false,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

// countUsersWithEmail reads the users table, because no operation finds a user by address.
func countUsersWithEmail(t *testing.T, app *apptest.App, email string) int {
	t.Helper()

	var userCount int
	app.QuerySQL(t, &userCount, "SELECT COUNT(*) FROM users WHERE email = ?", email)

	return userCount
}

// refuseMemberInserts makes the database refuse every new member, a failure the API cannot cause.
func refuseMemberInserts(t *testing.T, app *apptest.App) {
	t.Helper()

	app.ExecuteSQL(t, `CREATE FUNCTION refuse_member_insert() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'member inserts are refused by the test';
		END;
		$$ LANGUAGE plpgsql`)
	app.ExecuteSQL(t, `CREATE TRIGGER refuse_member_insert
		BEFORE INSERT ON workspaces_members
		FOR EACH ROW EXECUTE FUNCTION refuse_member_insert()`)
}

func renameUser(t *testing.T, user apptest.Session, name string) {
	t.Helper()

	response, err := user.Client.UpdateUserWithResponse(t.Context(), user.UserID, apiclient.UserUpdate{
		Name:  name,
		Email: user.Email,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}
