package users_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

// The contract has no invitation operation: the product invites a user by adding an unknown address
// to a workspace, which the old workspace member route serves through Module.InviteUser.
func TestInviteUser(t *testing.T) {
	t.Run("invites an unknown address an instance admin adds to a workspace", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		workspaceID := createWorkspace(t, app, admin)
		invitedEmail := apptest.NewTestEmail("invited")

		status, body := addWorkspaceMember(t, app, admin, workspaceID, invitedEmail)

		require.Equal(t, http.StatusOK, status, string(body))
		assert.Contains(t, string(body), `"INVITED"`)
		assert.Equal(t, 1, countUsersWithStatus(t, app, invitedEmail, "INVITED"))
		assert.Contains(t, listGlobalAuditMessages(t, app, admin.Token), "User invited: "+invitedEmail+" for workspace")
	})

	t.Run("lets a member invite when the instance allows member invitations", func(t *testing.T) {
		app := apptest.New(t)
		app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		workspaceID := createWorkspace(t, app, member)
		invitedEmail := apptest.NewTestEmail("invited")

		status, body := addWorkspaceMember(t, app, member, workspaceID, invitedEmail)

		require.Equal(t, http.StatusOK, status, string(body))
		assert.Equal(t, 1, countUsersWithStatus(t, app, invitedEmail, "INVITED"))
	})

	t.Run("refuses a member when the instance does not allow member invitations", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		workspaceID := createWorkspace(t, app, member)
		closedInvitations := defaultSettingsUpdate()
		closedInvitations.IsAllowMemberInvitations = false
		updatedSettings := updateInstanceSettings(t, admin.Client, closedInvitations)
		require.Equal(t, http.StatusOK, updatedSettings.StatusCode(), string(updatedSettings.Body))
		invitedEmail := apptest.NewTestEmail("invited")

		status, body := addWorkspaceMember(t, app, member, workspaceID, invitedEmail)

		assert.Equal(t, http.StatusBadRequest, status, string(body))
		assert.Equal(t, 0, countUsersWithStatus(t, app, invitedEmail, "INVITED"))
	})

	t.Run("refuses a password sign-in for the invited user like a wrong password", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		invitedEmail := inviteThroughWorkspace(t, app, admin)

		invitedResponse := signIn(t, app, invitedEmail, apptest.TestPassword)
		wrongPasswordResponse := signIn(t, app, admin.Email, "wrong-password-123")

		requireProblem(t, invitedResponse.StatusCode(), invitedResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
		requireProblem(t, wrongPasswordResponse.StatusCode(), wrongPasswordResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
	})

	t.Run("lets the invited user finish sign-up while registration is closed", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		closedRegistration := defaultSettingsUpdate()
		closedRegistration.IsAllowExternalRegistrations = false
		updatedSettings := updateInstanceSettings(t, admin.Client, closedRegistration)
		require.Equal(t, http.StatusOK, updatedSettings.StatusCode(), string(updatedSettings.Body))
		invitedEmail := inviteThroughWorkspace(t, app, admin)

		invitedUser := app.SignUp(t, invitedEmail)

		readUser := getUser(t, invitedUser.Client, invitedUser.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserStatusACTIVE, readUser.JSON200.Status)
		assert.Equal(t, apiclient.UserRoleMEMBER, readUser.JSON200.Role)
	})

	t.Run("gives the invited user the workspace membership after sign-up", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		workspaceID := createWorkspace(t, app, admin)
		invitedEmail := apptest.NewTestEmail("invited")
		status, body := addWorkspaceMember(t, app, admin, workspaceID, invitedEmail)
		require.Equal(t, http.StatusOK, status, string(body))

		invitedUser := app.SignUp(t, invitedEmail)

		status, body = sendAuthorizedRawRequest(t, app, invitedUser, http.MethodGet,
			"/workspaces/memberships/"+workspaceID.String()+"/members", "")
		require.Equal(t, http.StatusOK, status, string(body))
		var listedMembers struct {
			Members []workspaceMember `json:"members"`
		}
		require.NoError(t, json.Unmarshal(body, &listedMembers))
		assert.Contains(t, listedMembers.Members,
			workspaceMember{UserID: invitedUser.UserID, Role: "WORKSPACE_MEMBER"})
	})

	t.Run("lets the invited user finish sign-up through GitHub", func(t *testing.T) {
		app := apptest.New(t, apptest.WithGitHub())
		admin := app.CreateAdmin(t)
		invitedEmail := inviteThroughWorkspace(t, app, admin)

		session := requireGitHubSession(t, app, apptest.NewGitHubAccount(invitedEmail))

		readUser := getUser(t, app.NewSessionClient(t, session.Token), session.UserId)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserStatusACTIVE, readUser.JSON200.Status)
		assert.Equal(t, "GitHub User", readUser.JSON200.Name)
	})
}

// workspaceMember reads the old members route, which the contract does not describe yet.
type workspaceMember struct {
	UserID uuid.UUID `json:"userId"`
	Role   string    `json:"role"`
}

func inviteThroughWorkspace(t *testing.T, app *apptest.App, inviter apptest.Session) string {
	t.Helper()

	invitedEmail := apptest.NewTestEmail("invited")
	status, body := addWorkspaceMember(t, app, inviter, createWorkspace(t, app, inviter), invitedEmail)
	require.Equal(t, http.StatusOK, status, string(body))

	return invitedEmail
}

// createWorkspace creates a workspace through the old route, which the contract does not describe
// yet.
func createWorkspace(t *testing.T, app *apptest.App, owner apptest.Session) uuid.UUID {
	t.Helper()

	status, body := sendAuthorizedRawRequest(t, app, owner, http.MethodPost, "/workspaces", `{"name":"Invitations"}`)
	require.Equal(t, http.StatusOK, status, string(body))

	var createdWorkspace struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &createdWorkspace))

	return createdWorkspace.ID
}

func addWorkspaceMember(
	t *testing.T,
	app *apptest.App,
	actor apptest.Session,
	workspaceID uuid.UUID,
	email string,
) (int, []byte) {
	t.Helper()

	return sendAuthorizedRawRequest(
		t,
		app,
		actor,
		http.MethodPost,
		"/workspaces/memberships/"+workspaceID.String()+"/members",
		`{"email":"`+email+`","role":"WORKSPACE_MEMBER"}`,
	)
}

// countUsersWithStatus reads the users table, because no operation lists invited users yet.
func countUsersWithStatus(t *testing.T, app *apptest.App, email, status string) int {
	t.Helper()

	var userCount int
	app.QuerySQL(t, &userCount, "SELECT COUNT(*) FROM users WHERE email = ? AND status = ?", email, status)

	return userCount
}
