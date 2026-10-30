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

const resetPassword = "reset-password-456"

func TestRequestPasswordReset(t *testing.T) {
	t.Parallel()

	t.Run("mails a code that replaces the password of the bootstrap administrator", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		requested := requestPasswordReset(t, app, admin.Email)
		require.Equal(t, http.StatusNoContent, requested.StatusCode(), string(requested.Body))

		completed := completePasswordReset(
			t,
			app,
			admin.Email,
			readLatestMailedCode(t, app, admin.Email),
			resetPassword,
		)
		require.Equal(t, http.StatusNoContent, completed.StatusCode(), string(completed.Body))
		requireCompletedSignIn(t, signIn(t, app, admin.Email, resetPassword))
		oldPasswordSignIn := signIn(t, app, admin.Email, apptest.TestPassword)
		requireProblem(t, oldPasswordSignIn.StatusCode(), oldPasswordSignIn.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
	})

	t.Run("answers alike and mails nothing for an address without a user", func(t *testing.T) {
		app := apptest.New(t)
		app.CreateAdmin(t)
		unknownEmail := apptest.NewTestEmail("unknown")

		response := requestPasswordReset(t, app, unknownEmail)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, app.Mail.GetMessagesTo(unknownEmail))
	})

	t.Run("answers alike and mails nothing for a deactivated user", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		requireDeactivated(t, admin, member.UserID)

		response := requestPasswordReset(t, app, member.Email)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, app.Mail.GetMessagesTo(member.Email))
	})

	t.Run("answers alike and mails nothing for an invited user", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		invitedEmail := inviteThroughWorkspace(t, admin)
		mailCountBeforeReset := len(app.Mail.GetMessagesTo(invitedEmail))

		response := requestPasswordReset(t, app, invitedEmail)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Len(t, app.Mail.GetMessagesTo(invitedEmail), mailCountBeforeReset)
	})

	t.Run("answers alike and mails nothing for the placeholder address", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.ExecuteSQL(t, "UPDATE users SET email = 'admin' WHERE id = ?", admin.UserID)

		response := requestPasswordReset(t, app, "admin")

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Zero(t, app.Mail.GetMessageCount())
	})

	t.Run("returns 429 rate_limited on the fourth request in an hour, for any address", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		unknownEmail := apptest.NewTestEmail("unknown")

		for range 3 {
			require.Equal(t, http.StatusNoContent, requestPasswordReset(t, app, admin.Email).StatusCode())
			require.Equal(t, http.StatusNoContent, requestPasswordReset(t, app, unknownEmail).StatusCode())
		}

		userResponse := requestPasswordReset(t, app, admin.Email)
		unknownResponse := requestPasswordReset(t, app, unknownEmail)

		requireProblem(t, userResponse.StatusCode(), userResponse.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeRateLimited)
		requireProblem(t, unknownResponse.StatusCode(), unknownResponse.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeRateLimited)
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 3)
	})

	t.Run("returns 400 human_verification_failed without the challenge answer", func(t *testing.T) {
		app := apptest.New(t, apptest.WithTurnstile())
		admin := app.CreateAdmin(t)

		response := requestPasswordReset(t, app, admin.Email)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeHumanVerificationFailed)
		assert.Empty(t, app.Mail.GetMessagesTo(admin.Email))
	})
}

func TestCompletePasswordReset(t *testing.T) {
	t.Parallel()

	t.Run("ends the sessions issued before the reset", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)

		completed := completePasswordReset(t, app, admin.Email, code, resetPassword)
		require.Equal(t, http.StatusNoContent, completed.StatusCode(), string(completed.Body))

		response := getUser(t, admin.Client, admin.UserID)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("invalidates a pending code after a password reset at the same clock instant", func(t *testing.T) {
		app, admin := newTwoFactorApp(t)
		pending := startTwoFactorSignIn(t, app, admin.Email)
		pendingCode := readLatestMailedCode(t, app, admin.Email)
		resetCode := requestResetCode(t, app, admin.Email)
		completed := completePasswordReset(t, app, admin.Email, resetCode, resetPassword)
		require.Equal(t, http.StatusNoContent, completed.StatusCode(), string(completed.Body))
		verified := verifySignInCode(t, app, pending.PendingSignInId, pendingCode)
		requireProblem(
			t,
			verified.StatusCode(),
			verified.Body,
			http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable,
		)
		freshPending := requireCodeRequired(t, signIn(t, app, admin.Email, resetPassword))
		freshVerified := verifySignInCode(
			t,
			app,
			freshPending.PendingSignInId,
			readLatestMailedCode(t, app, admin.Email),
		)
		require.Equal(t, http.StatusOK, freshVerified.StatusCode(), string(freshVerified.Body))
	})

	t.Run("returns 400 invalid_reset_code for a wrong code and keeps the right one usable", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)

		wrongCodeResponse := completePasswordReset(t, app, admin.Email, getWrongCode(code), resetPassword)
		rightCodeResponse := completePasswordReset(t, app, admin.Email, code, resetPassword)

		requireProblem(t, wrongCodeResponse.StatusCode(), wrongCodeResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
		require.Equal(t, http.StatusNoContent, rightCodeResponse.StatusCode(), string(rightCodeResponse.Body))
	})

	t.Run("returns 400 invalid_reset_code for a used code", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)
		completed := completePasswordReset(t, app, admin.Email, code, resetPassword)
		require.Equal(t, http.StatusNoContent, completed.StatusCode(), string(completed.Body))

		response := completePasswordReset(t, app, admin.Email, code, "another-password-789")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, resetPassword))
	})

	t.Run("returns 400 invalid_reset_code for a code older than an hour", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)
		app.Clock.Advance(time.Hour + time.Second)

		response := completePasswordReset(t, app, admin.Email, code, resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	})

	t.Run("returns 400 invalid_reset_code for an address without a user", func(t *testing.T) {
		app := apptest.New(t)
		app.CreateAdmin(t)

		response := completePasswordReset(t, app, apptest.NewTestEmail("unknown"), "123456", resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	})

	t.Run("returns 400 naming the password when it is shorter than 8 characters", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)

		response := completePasswordReset(t, app, admin.Email, code, "short")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		require.NotNil(t, response.ApplicationproblemJSON400.Errors)
		assert.Equal(t, "newPassword", (*response.ApplicationproblemJSON400.Errors)[0].Field)
	})

	t.Run("deletes the codes of the past hour in the cleanup pass", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		requestResetCode(t, app, admin.Email)
		app.Clock.Advance(time.Hour + time.Second)

		app.RunJobPass(t, "users_expired_codes_cleanup")

		var resetCodeCount int
		app.QuerySQL(t, &resetCodeCount, "SELECT COUNT(*) FROM users_password_reset_codes WHERE user_id = ?",
			admin.UserID)
		assert.Zero(t, resetCodeCount)
	})
}

func requestResetCode(t *testing.T, app *apptest.App, email string) string {
	t.Helper()

	response := requestPasswordReset(t, app, email)
	require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))

	return readLatestMailedCode(t, app, email)
}

func requestPasswordReset(t *testing.T, app *apptest.App, email string) *apiclient.RequestPasswordResetResponse {
	t.Helper()

	response, err := app.NewClient(t).RequestPasswordResetWithResponse(t.Context(), apiclient.PasswordResetRequest{
		Email:                    email,
		CloudflareTurnstileToken: nil,
	})
	require.NoError(t, err)

	return response
}

func completePasswordReset(
	t *testing.T,
	app *apptest.App,
	email string,
	code string,
	newPassword string,
) *apiclient.CompletePasswordResetResponse {
	t.Helper()

	response, err := app.NewClient(t).CompletePasswordResetWithResponse(t.Context(), apiclient.PasswordResetCompletion{
		Email:       email,
		Code:        code,
		NewPassword: &newPassword,
	})
	require.NoError(t, err)

	return response
}
