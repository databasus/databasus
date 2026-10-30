package users_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	t.Run("replaces the name and the address, and the next sign-in uses the new address", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		changedEmail := apptest.NewTestEmail("changed")

		response := updateUser(t, admin.Client, admin.UserID, "Renamed Owner", changedEmail)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, changedEmail, response.JSON200.Email)
		assert.Equal(t, "Renamed Owner", response.JSON200.Name)
		requireCompletedSignIn(t, signIn(t, app, changedEmail, apptest.TestPassword))
		previousAddressSignIn := signIn(t, app, admin.Email, apptest.TestPassword)
		requireProblem(t, previousAddressSignIn.StatusCode(), previousAddressSignIn.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
	})

	t.Run("keeps the session that changed the address", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		updated := updateUser(t, admin.Client, admin.UserID, "Owner", apptest.NewTestEmail("changed"))
		require.Equal(t, http.StatusOK, updated.StatusCode(), string(updated.Body))

		readUser := getUser(t, admin.Client, admin.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
	})

	t.Run("records the old and the new address in the audit log", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		changedEmail := apptest.NewTestEmail("changed")

		response := updateUser(t, admin.Client, admin.UserID, "Test User", changedEmail)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		assert.Contains(t, listGlobalAuditMessages(t, admin),
			"Email changed from '"+admin.Email+"' to '"+changedEmail+"'")
	})

	t.Run("returns 409 email_taken when the address belongs to another user", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		response := updateUser(t, admin.Client, admin.UserID, "Owner", member.Email)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeEmailTaken)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, apptest.TestPassword))
	})

	t.Run("returns 400 naming the email when it is not an address", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := updateUser(t, admin.Client, admin.UserID, "Owner", "not-an-address")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		require.NotNil(t, response.ApplicationproblemJSON400.Errors)
		assert.Equal(t, "email", (*response.ApplicationproblemJSON400.Errors)[0].Field)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, apptest.TestPassword))
	})

	t.Run("keeps an address that is not valid when only the name changes", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.ExecuteSQL(t, "UPDATE users SET email = 'admin' WHERE id = ?", admin.UserID)

		response := updateUser(t, admin.Client, admin.UserID, "Renamed Owner", "admin")

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, "Renamed Owner", response.JSON200.Name)
	})

	t.Run("returns 403 permission_denied for another user's ID, even for an instance admin", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		response := updateUser(t, admin.Client, member.UserID, "Renamed", member.Email)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})
}

func TestChangeUserPassword(t *testing.T) {
	t.Parallel()

	t.Run("signs the user in with the new password and not with the old one", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := changeUserPassword(t, admin.Client, admin.UserID, "new-password-456")

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		requireCompletedSignIn(t, signIn(t, app, admin.Email, "new-password-456"))
		oldPasswordSignIn := signIn(t, app, admin.Email, apptest.TestPassword)
		requireProblem(t, oldPasswordSignIn.StatusCode(), oldPasswordSignIn.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
	})

	t.Run("returns 401 unauthenticated for a token issued before the change", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.Clock.Advance(2 * time.Second)

		changed := changeUserPassword(t, admin.Client, admin.UserID, "new-password-456")
		require.Equal(t, http.StatusNoContent, changed.StatusCode(), string(changed.Body))

		response := getUser(t, admin.Client, admin.UserID)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 400 naming the password when it is shorter than 8 characters", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := changeUserPassword(t, admin.Client, admin.UserID, "short")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, apptest.TestPassword))
	})

	t.Run("returns 403 permission_denied for another user's ID, even for an instance admin", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		response := changeUserPassword(t, admin.Client, member.UserID, "new-password-456")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireCompletedSignIn(t, signIn(t, app, member.Email, apptest.TestPassword))
	})
}

func updateUser(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	userID apiclient.UserId,
	name string,
	email string,
) *apiclient.UpdateUserResponse {
	t.Helper()

	response, err := client.UpdateUserWithResponse(t.Context(), userID, apiclient.UserUpdate{Name: name, Email: email})
	require.NoError(t, err)

	return response
}

func changeUserPassword(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	userID apiclient.UserId,
	newPassword string,
) *apiclient.ChangeUserPasswordResponse {
	t.Helper()

	response, err := client.ChangeUserPasswordWithResponse(t.Context(), userID, apiclient.PasswordChange{
		NewPassword: &newPassword,
	})
	require.NoError(t, err)

	return response
}
