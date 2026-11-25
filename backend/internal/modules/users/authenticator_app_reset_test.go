package users_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestAuthenticatorAppReset(t *testing.T) {
	t.Parallel()

	t.Run("takes a member through a new setup at the next sign-in under the app method", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		app.ConnectAuthenticatorApp(t, member)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

		response := resetAuthenticatorApp(t, admin, member.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Nil(t, response.JSON200.AuthenticatorAppConnectedAt)
		assert.Nil(t, response.JSON200.UnusedRecoveryCodeCount)
		requireAuthenticatorSetupRequired(t, signIn(t, app, member.Email, apptest.TestPassword))
		resetEvent := findAuditEvent(t, admin, "Authenticator app reset: "+member.Email)
		require.NotNil(t, resetEvent.Author)
		assert.Equal(t, admin.UserID, resetEvent.Author.UserId)
		var recoveryCodeCount int
		app.QuerySQL(t, &recoveryCodeCount, "SELECT count(*) FROM users_recovery_codes")
		assert.Zero(t, recoveryCodeCount)
	})

	t.Run("keeps the member's session and ends the pending sign-in that waits for the app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		memberApp := app.ConnectAuthenticatorApp(t, member)
		pendingSignIn := startAppCodeSignIn(t, app, member.Email)

		response := resetAuthenticatorApp(t, admin, member.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, http.StatusOK, getUser(t, member.Client, member.UserID).StatusCode())
		verifyResponse := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.GenerateAuthenticatorCode(t, memberApp.Secret))
		requireProblem(t, verifyResponse.StatusCode(), verifyResponse.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("returns 403 permission_denied and keeps the app when an admin resets their own", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.ConnectAuthenticatorApp(t, admin)

		response := resetAuthenticatorApp(t, admin, admin.UserID)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.NotNil(t, getAppState(t, admin).AuthenticatorAppConnectedAt)
	})

	t.Run("returns 403 permission_denied when another admin resets root admin's app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		rootAdmin := app.CreateAdmin(t)
		app.ConnectAuthenticatorApp(t, rootAdmin)
		otherAdmin := createSecondAdmin(t, app, rootAdmin)

		response := resetAuthenticatorApp(t, otherAdmin, rootAdmin.UserID)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.NotNil(t, getAppState(t, rootAdmin).AuthenticatorAppConnectedAt)
	})

	t.Run("returns 403 permission_denied when an admin who is not root admin resets another admin",
		func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t)
			rootAdmin := app.CreateAdmin(t)
			firstAdmin := createSecondAdmin(t, app, rootAdmin)
			secondAdmin := createSecondAdmin(t, app, rootAdmin)
			app.ConnectAuthenticatorApp(t, secondAdmin)

			response := resetAuthenticatorApp(t, firstAdmin, secondAdmin.UserID)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.NotNil(t, getAppState(t, secondAdmin).AuthenticatorAppConnectedAt)
		})

	t.Run("lets root admin reset another admin's app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		rootAdmin := app.CreateAdmin(t)
		otherAdmin := createSecondAdmin(t, app, rootAdmin)
		app.ConnectAuthenticatorApp(t, otherAdmin)

		response := resetAuthenticatorApp(t, rootAdmin, otherAdmin.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Nil(t, response.JSON200.AuthenticatorAppConnectedAt)
	})

	t.Run("answers the user unchanged and records nothing for a user without an app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		auditMessagesBefore := listGlobalAuditMessages(t, admin)

		response := resetAuthenticatorApp(t, admin, member.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Nil(t, response.JSON200.AuthenticatorAppConnectedAt)
		assert.Len(t, listGlobalAuditMessages(t, admin), len(auditMessagesBefore))
	})

	t.Run("returns 403 permission_denied when a member resets another member", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		otherMember := app.CreateNonAdminUser(t, admin)
		app.ConnectAuthenticatorApp(t, otherMember)

		response := resetAuthenticatorApp(t, member, otherMember.UserID)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("returns 404 for a user that does not exist", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := resetAuthenticatorApp(t, admin, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response, err := app.NewClient(t).ResetAuthenticatorAppWithResponse(t.Context(), admin.UserID)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

func createSecondAdmin(t *testing.T, app *apptest.App, rootAdmin apptest.Session) apptest.Session {
	t.Helper()

	otherAdmin := app.CreateNonAdminUser(t, rootAdmin)
	response := changeUserRole(t, rootAdmin.Client, otherAdmin.UserID, apiclient.UserRoleADMIN)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return otherAdmin
}
