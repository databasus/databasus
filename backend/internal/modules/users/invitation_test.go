package users_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

// The contract has no invitation operation: the product invites a user by adding an unknown address
// to a workspace, which addWorkspaceMember serves through Module.InviteUser.
func TestInviteUser(t *testing.T) {
	t.Parallel()

	t.Run("invites an unknown address an instance admin adds to a workspace", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		workspaceID := createWorkspace(t, admin)
		invitedEmail := apptest.NewTestEmail("invited")

		response := addWorkspaceMember(t, admin, workspaceID, invitedEmail)

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.AddedWorkspaceMemberOutcomeINVITED, response.JSON201.Outcome)
		assert.Equal(t, 1, countUsersWithStatus(t, app, invitedEmail, "INVITED"))
		assert.Contains(t, listGlobalAuditMessages(t, admin), "User invited: "+invitedEmail+" for workspace")
	})

	t.Run("lets a member invite when the instance allows member invitations", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		workspaceID := createWorkspace(t, member)
		invitedEmail := apptest.NewTestEmail("invited")

		response := addWorkspaceMember(t, member, workspaceID, invitedEmail)

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, 1, countUsersWithStatus(t, app, invitedEmail, "INVITED"))
	})

	t.Run("refuses a member when the instance does not allow member invitations", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		workspaceID := createWorkspace(t, member)
		closedInvitations := defaultSettingsUpdate()
		closedInvitations.IsAllowMemberInvitations = false
		updatedSettings := updateInstanceSettings(t, admin.Client, closedInvitations)
		require.Equal(t, http.StatusOK, updatedSettings.StatusCode(), string(updatedSettings.Body))
		invitedEmail := apptest.NewTestEmail("invited")

		response := addWorkspaceMember(t, member, workspaceID, invitedEmail)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, 0, countUsersWithStatus(t, app, invitedEmail, "INVITED"))
	})

	t.Run("identifies incomplete registration when an invited user signs in with a password", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		invitedEmail := inviteThroughWorkspace(t, admin)

		invitedResponse := signIn(t, app, invitedEmail, apptest.TestPassword)
		wrongPasswordResponse := signIn(t, app, admin.Email, "wrong-password-123")

		requireProblem(t, invitedResponse.StatusCode(), invitedResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeRegistrationIncomplete)
		requireProblem(t, wrongPasswordResponse.StatusCode(), wrongPasswordResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodePasswordIncorrect)
	})

	t.Run("lets the invited user finish sign-up while registration is closed", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		closedRegistration := defaultSettingsUpdate()
		closedRegistration.IsAllowExternalRegistrations = false
		updatedSettings := updateInstanceSettings(t, admin.Client, closedRegistration)
		require.Equal(t, http.StatusOK, updatedSettings.StatusCode(), string(updatedSettings.Body))
		invitedEmail := inviteThroughWorkspace(t, admin)

		var invitedVersion string
		app.QuerySQL(t, &invitedVersion, "SELECT password_version FROM users WHERE email = ?", invitedEmail)
		require.NotEqual(t, uuid.Nil.String(), invitedVersion)

		invitedUser := app.SignUp(t, invitedEmail)

		var activeVersion string
		app.QuerySQL(t, &activeVersion, "SELECT password_version FROM users WHERE email = ?", invitedEmail)
		require.NotEqual(t, invitedVersion, activeVersion)
		require.NotEqual(t, uuid.Nil.String(), activeVersion)

		readUser := getUser(t, invitedUser.Client, invitedUser.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserStatusACTIVE, readUser.JSON200.Status)
		assert.Equal(t, apiclient.UserRoleMEMBER, readUser.JSON200.Role)
	})

	t.Run("makes the invited user a member of the workspace after sign-up", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		workspaceID := createWorkspace(t, admin)
		invitedEmail := apptest.NewTestEmail("invited")
		addResponse := addWorkspaceMember(t, admin, workspaceID, invitedEmail)
		require.Equal(t, http.StatusCreated, addResponse.StatusCode(), string(addResponse.Body))

		invitedUser := app.SignUp(t, invitedEmail)

		membersResponse, err := invitedUser.Client.ListWorkspaceMembersWithResponse(t.Context(),
			&apiclient.ListWorkspaceMembersParams{WorkspaceId: workspaceID, Limit: nil, After: nil})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, membersResponse.StatusCode(), string(membersResponse.Body))
		require.Len(t, membersResponse.JSON200.Items, 2)

		invitedMember := membersResponse.JSON200.Items[0]
		if invitedMember.UserId != invitedUser.UserID {
			invitedMember = membersResponse.JSON200.Items[1]
		}

		assert.Equal(t, invitedUser.UserID, invitedMember.UserId)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEMEMBER, invitedMember.Role)
		assert.False(t, invitedMember.IsInvited)
	})

	t.Run("lets the invited user finish sign-up through GitHub", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		admin := app.CreateAdmin(t)
		invitedEmail := inviteThroughWorkspace(t, admin)

		session := requireGitHubSession(t, app, apptest.NewGitHubAccount(invitedEmail))

		readUser := getUser(t, app.NewSessionClient(t, session.Token), session.UserId)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserStatusACTIVE, readUser.JSON200.Status)
		assert.Equal(t, "GitHub User", readUser.JSON200.Name)
	})
}

func inviteThroughWorkspace(t *testing.T, inviter apptest.Session) string {
	t.Helper()

	invitedEmail := apptest.NewTestEmail("invited")
	response := addWorkspaceMember(t, inviter, createWorkspace(t, inviter), invitedEmail)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return invitedEmail
}

func createWorkspace(t *testing.T, owner apptest.Session) uuid.UUID {
	t.Helper()

	return apptest.CreateWorkspace(t, owner, "Invitations").Id
}

func addWorkspaceMember(
	t *testing.T,
	actor apptest.Session,
	workspaceID uuid.UUID,
	email string,
) *apiclient.AddWorkspaceMemberResponse {
	t.Helper()

	response, err := actor.Client.AddWorkspaceMemberWithResponse(t.Context(), apiclient.WorkspaceMemberAddition{
		WorkspaceId: workspaceID,
		Email:       openapi_types.Email(email),
		Role:        apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
	})
	require.NoError(t, err)

	return response
}

// countUsersWithStatus reads the users table, because no operation lists invited users yet.
func countUsersWithStatus(t *testing.T, app *apptest.App, email, status string) int {
	t.Helper()

	var userCount int
	app.QuerySQL(t, &userCount, "SELECT COUNT(*) FROM users WHERE email = ? AND status = ?", email, status)

	return userCount
}
