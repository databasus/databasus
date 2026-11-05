package users_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestTwoFactorSignIn(t *testing.T) {
	t.Parallel()

	t.Run("answers a pending sign-in without a token and sends one code when the second factor is on",
		func(t *testing.T) {
			t.Parallel()

			app, admin := newTwoFactorApp(t)

			response := signIn(t, app, admin.Email, apptest.TestPassword)

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			assert.NotContains(t, string(response.Body), "token")

			codeRequired, err := response.JSON200.AsSignInCodeRequired()
			require.NoError(t, err)
			assert.Equal(t, apiclient.SignInCodeRequiredOutcomeCODEREQUIRED, codeRequired.Outcome)
			assert.Equal(t, admin.Email, codeRequired.Email)
			assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 1)
		})

	t.Run("keeps only a hash of the code", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		code := readLatestMailedCode(t, app, admin.Email)

		var hashedCode string
		app.QuerySQL(t, &hashedCode, "SELECT hashed_code FROM users_pending_sign_ins WHERE id = ?",
			pendingSignIn.PendingSignInId)

		assert.NotEqual(t, code, hashedCode)
		assert.NotContains(t, hashedCode, code)
	})

	t.Run("sends a message with the code, its ten minutes and the rule that a new code replaces it",
		func(t *testing.T) {
			t.Parallel()

			app, admin := newTwoFactorApp(t)

			startTwoFactorSignIn(t, app, admin.Email)

			messages := app.Mail.GetMessagesTo(admin.Email)
			require.Len(t, messages, 1)
			assert.Regexp(t, mailedCodePattern, messages[0].HTMLBody)
			assert.Contains(t, messages[0].HTMLBody, "10 minutes")
			assert.Contains(t, messages[0].HTMLBody, "replaces this one")
		})

	t.Run("returns the same pending sign-in and sends nothing when the password step repeats while it is live",
		func(t *testing.T) {
			t.Parallel()

			app, admin := newTwoFactorApp(t)
			firstPendingSignIn := startTwoFactorSignIn(t, app, admin.Email)

			secondPendingSignIn := startTwoFactorSignIn(t, app, admin.Email)

			assert.Equal(t, firstPendingSignIn.PendingSignInId, secondPendingSignIn.PendingSignInId)
			assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 1)
		})

	t.Run("sends one code when two password steps arrive at once", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)

		var waitGroup sync.WaitGroup
		for range 2 {
			waitGroup.Go(func() {
				signIn(t, app, admin.Email, apptest.TestPassword)
			})
		}
		waitGroup.Wait()

		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 1)
	})

	t.Run("sends a fresh code after the password changed during a pending sign-in", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		firstPendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		changed := changeUserPassword(t, admin.Client, admin.UserID, apptest.TestPassword)
		require.Equal(t, http.StatusNoContent, changed.StatusCode(), string(changed.Body))

		secondPendingSignIn := startTwoFactorSignIn(t, app, admin.Email)

		assert.NotEqual(t, firstPendingSignIn.PendingSignInId, secondPendingSignIn.PendingSignInId)
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 2)
	})

	t.Run("returns 429 too_many_sign_in_codes without mail after five codes within an hour", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)

		for range 5 {
			startTwoFactorSignIn(t, app, admin.Email)
			app.Clock.Advance(11 * time.Minute)
		}

		response := signIn(t, app, admin.Email, apptest.TestPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeTooManySignInCodes)
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 5)
	})

	t.Run("counts earlier codes toward the hourly cap after fixed-clock password replacements", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		for range 5 {
			pending := startTwoFactorSignIn(t, app, admin.Email)
			verified := verifySignInCode(t, app, pending.PendingSignInId, readLatestMailedCode(t, app, admin.Email))
			require.Equal(t, http.StatusOK, verified.StatusCode(), string(verified.Body))

			currentClient := app.NewSessionClient(t, verified.JSON200.Token)
			changed := changeUserPassword(t, currentClient, admin.UserID, apptest.TestPassword)
			require.Equal(t, http.StatusNoContent, changed.StatusCode(), string(changed.Body))
		}
		response := signIn(t, app, admin.Email, apptest.TestPassword)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeTooManySignInCodes)
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 5)
	})

	t.Run("returns 503 sign_in_code_not_sent when the instance has no mail server", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithoutMailServer())
		admin := app.CreateAdmin(t)
		requireTwoFactorAfterMailServerLoss(t, app, admin)

		response := signIn(t, app, admin.Email, apptest.TestPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusServiceUnavailable,
			apiclient.ProblemCodeSignInCodeNotSent)
	})

	t.Run(
		"returns 503 sign_in_code_not_sent without a token when the mail server refuses the code",
		func(t *testing.T) {
			t.Parallel()

			app, admin := newTwoFactorApp(t)
			app.Mail.RefuseSends()

			response := signIn(t, app, admin.Email, apptest.TestPassword)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusServiceUnavailable,
				apiclient.ProblemCodeSignInCodeNotSent)
			assert.NotContains(t, string(response.Body), "token")
		},
	)

	t.Run("sends no mail and answers as with one factor when the password is wrong", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)

		response := signIn(t, app, admin.Email, "wrong-password-123")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
		assert.Empty(t, app.Mail.GetMessagesTo(admin.Email))
	})

	t.Run("sends no mail and creates no pending sign-in when the address is unknown", func(t *testing.T) {
		t.Parallel()

		app, _ := newTwoFactorApp(t)
		unknownEmail := apptest.NewTestEmail("unknown")

		response := signIn(t, app, unknownEmail, apptest.TestPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
		assert.Zero(t, app.Mail.GetMessageCount())

		var pendingSignInCount int64
		app.QuerySQL(t, &pendingSignInCount, "SELECT COUNT(*) FROM users_pending_sign_ins")
		assert.Zero(t, pendingSignInCount)
	})

	t.Run("records the sign-in in the audit log only at the code step", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		signedInMessage := "User signed in with email: " + admin.Email

		assert.NotContains(t, listGlobalAuditMessages(t, admin), signedInMessage)

		verifyResponse := verifySignInCode(
			t,
			app,
			pendingSignIn.PendingSignInId,
			readLatestMailedCode(t, app, admin.Email),
		)
		require.Equal(t, http.StatusOK, verifyResponse.StatusCode(), string(verifyResponse.Body))
		assert.Contains(t, listGlobalAuditMessages(t, admin), signedInMessage)
	})
}

func TestVerifySignInCode(t *testing.T) {
	t.Parallel()

	t.Run("returns a session when the code that arrived is submitted", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, readLatestMailedCode(t, app, admin.Email))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, admin.UserID, response.JSON200.UserId)

		readUser := getUser(t, app.NewSessionClient(t, response.JSON200.Token), admin.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
	})

	t.Run("returns 410 pending_sign_in_not_usable when the code already completed a sign-in", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		code := readLatestMailedCode(t, app, admin.Email)
		require.Equal(t, http.StatusOK, verifySignInCode(t, app, pendingSignIn.PendingSignInId, code).StatusCode())

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, code)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("returns 410 pending_sign_in_not_usable when the code is submitted eleven minutes later", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		app.Clock.Advance(11 * time.Minute)

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, readLatestMailedCode(t, app, admin.Email))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("returns 400 sign_in_code_incorrect when the code belongs to another pending sign-in", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		member := app.CreateNonAdminUser(t, admin)
		adminPendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		startTwoFactorSignIn(t, app, member.Email)

		response := verifySignInCode(t, app, adminPendingSignIn.PendingSignInId,
			readLatestMailedCode(t, app, member.Email))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
	})

	t.Run("returns 410 pending_sign_in_not_usable when nobody issued the identifier", func(t *testing.T) {
		t.Parallel()

		app, _ := newTwoFactorApp(t)

		response := verifySignInCode(t, app, uuid.New(), "123456")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("refuses the correct code after five wrong ones", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		code := readLatestMailedCode(t, app, admin.Email)

		for range 5 {
			verifySignInCode(t, app, pendingSignIn.PendingSignInId, getWrongCode(code))
		}

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, code)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("checks no more than five codes when ten wrong codes arrive at once", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		code := readLatestMailedCode(t, app, admin.Email)

		var waitGroup sync.WaitGroup
		for range 10 {
			waitGroup.Go(func() {
				verifySignInCode(t, app, pendingSignIn.PendingSignInId, getWrongCode(code))
			})
		}
		waitGroup.Wait()

		var failedAttemptCount int
		app.QuerySQL(t, &failedAttemptCount, "SELECT failed_attempt_count FROM users_pending_sign_ins WHERE id = ?",
			pendingSignIn.PendingSignInId)
		assert.Equal(t, 5, failedAttemptCount)

		app.Clock.Advance(time.Minute)
		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, code)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("returns one session when the same correct code arrives twice at once", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		code := readLatestMailedCode(t, app, admin.Email)

		statuses := make(chan int, 2)

		var waitGroup sync.WaitGroup
		for range 2 {
			waitGroup.Go(func() {
				statuses <- verifySignInCode(t, app, pendingSignIn.PendingSignInId, code).StatusCode()
			})
		}
		waitGroup.Wait()
		close(statuses)

		sessionCount := 0
		for status := range statuses {
			if status == http.StatusOK {
				sessionCount++
			}
		}

		assert.Equal(t, 1, sessionCount)
	})

	t.Run("returns 410 when the user was deactivated while the code was in flight", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		member := app.CreateNonAdminUser(t, admin)
		pendingSignIn := startTwoFactorSignIn(t, app, member.Email)
		requireDeactivated(t, admin, member.UserID)

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, readLatestMailedCode(t, app, member.Email))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("returns 410 when the password changed while the code was in flight", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		changed := changeUserPassword(t, admin.Client, admin.UserID, apptest.TestPassword)
		require.Equal(t, http.StatusNoContent, changed.StatusCode(), string(changed.Body))

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, readLatestMailedCode(t, app, admin.Email))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("completes the sign-in after the second factor was switched off", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		require.Equal(t, http.StatusOK, updateInstanceSettings(t, admin.Client, defaultSettingsUpdate()).StatusCode())

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, readLatestMailedCode(t, app, admin.Email))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("returns 400 when the path names an email address instead of a pending sign-in", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)

		verifyStatus, _ := sendRawRequest(t, app, http.MethodPost, "/pending-sign-ins/"+admin.Email+"/verify",
			`{"code":"123456"}`)
		resendStatus, _ := sendRawRequest(t, app, http.MethodPost, "/pending-sign-ins/"+admin.Email+"/resend", `{}`)

		assert.Equal(t, http.StatusBadRequest, verifyStatus)
		assert.Equal(t, http.StatusBadRequest, resendStatus)
	})

	t.Run("returns 400 human_verification_failed before checking the code when the challenge is not answered",
		func(t *testing.T) {
			t.Parallel()

			app, admin := newTwoFactorApp(t, apptest.WithTurnstile())
			pendingSignIn := startTwoFactorSignInWithAnswer(t, app, admin.Email)

			response := verifySignInCode(
				t,
				app,
				pendingSignIn.PendingSignInId,
				readLatestMailedCode(t, app, admin.Email),
			)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeHumanVerificationFailed)

			var failedAttemptCount int
			app.QuerySQL(t, &failedAttemptCount, "SELECT failed_attempt_count FROM users_pending_sign_ins WHERE id = ?",
				pendingSignIn.PendingSignInId)
			assert.Zero(t, failedAttemptCount)
		})
}

func TestResendSignInCode(t *testing.T) {
	t.Parallel()

	t.Run("sends a new code that names a new pending sign-in and retires the previous code", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		firstPendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		firstCode := readLatestMailedCode(t, app, admin.Email)

		response := resendSignInCode(t, app, firstPendingSignIn.PendingSignInId)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.NotEqual(t, firstPendingSignIn.PendingSignInId, response.JSON200.PendingSignInId)
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 2)

		firstCodeResponse := verifySignInCode(t, app, firstPendingSignIn.PendingSignInId, firstCode)
		requireProblem(t, firstCodeResponse.StatusCode(), firstCodeResponse.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)

		newCodeResponse := verifySignInCode(t, app, response.JSON200.PendingSignInId,
			readLatestMailedCode(t, app, admin.Email))
		require.Equal(t, http.StatusOK, newCodeResponse.StatusCode(), string(newCodeResponse.Body))
	})

	t.Run("refuses resend and verification after a fixed-clock password replacement", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		pending := startTwoFactorSignIn(t, app, admin.Email)
		code := readLatestMailedCode(t, app, admin.Email)

		changed := changeUserPassword(t, admin.Client, admin.UserID, "replacement-password")
		require.Equal(t, http.StatusNoContent, changed.StatusCode(), string(changed.Body))

		resent := resendSignInCode(t, app, pending.PendingSignInId)
		requireProblem(
			t,
			resent.StatusCode(),
			resent.Body,
			http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable,
		)

		verified := verifySignInCode(t, app, pending.PendingSignInId, code)
		requireProblem(
			t,
			verified.StatusCode(),
			verified.Body,
			http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable,
		)

		freshPending := requireCodeRequired(t, signIn(t, app, admin.Email, "replacement-password"))
		require.NotEqual(t, pending.PendingSignInId, freshPending.PendingSignInId)
		freshVerified := verifySignInCode(
			t,
			app,
			freshPending.PendingSignInId,
			readLatestMailedCode(t, app, admin.Email),
		)
		require.Equal(t, http.StatusOK, freshVerified.StatusCode(), string(freshVerified.Body))
		require.Equal(
			t,
			http.StatusOK,
			getUser(t, app.NewSessionClient(t, freshVerified.JSON200.Token), admin.UserID).StatusCode(),
		)
	})

	t.Run("returns 429 sign_in_code_resent_too_soon without mail when a second resend comes within the minute",
		func(t *testing.T) {
			t.Parallel()

			app, admin := newTwoFactorApp(t)
			pendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
			resentPendingSignIn := resendSignInCode(t, app, pendingSignIn.PendingSignInId)
			require.Equal(t, http.StatusOK, resentPendingSignIn.StatusCode(), string(resentPendingSignIn.Body))

			response := resendSignInCode(t, app, resentPendingSignIn.JSON200.PendingSignInId)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusTooManyRequests,
				apiclient.ProblemCodeSignInCodeResentTooSoon)
			assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 2)
		})

	t.Run("returns 400 human_verification_failed without mail when the challenge is not answered",
		func(t *testing.T) {
			t.Parallel()

			app, admin := newTwoFactorApp(t, apptest.WithTurnstile())
			pendingSignIn := startTwoFactorSignInWithAnswer(t, app, admin.Email)

			response := resendSignInCode(t, app, pendingSignIn.PendingSignInId)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeHumanVerificationFailed)
			assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 1)
		})
}

func TestExpiredCodesCleanup(t *testing.T) {
	t.Parallel()

	t.Run("deletes pending sign-ins older than an hour and keeps younger ones", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)
		member := app.CreateNonAdminUser(t, admin)
		oldPendingSignIn := startTwoFactorSignIn(t, app, admin.Email)
		app.Clock.Advance(50 * time.Minute)
		youngPendingSignIn := startTwoFactorSignIn(t, app, member.Email)
		app.Clock.Advance(11 * time.Minute)

		app.RunJobPass(t, "users_expired_codes_cleanup")

		var remainingIDs []uuid.UUID
		app.QuerySQL(t, &remainingIDs, "SELECT id FROM users_pending_sign_ins")
		assert.NotContains(t, remainingIDs, oldPendingSignIn.PendingSignInId)
		assert.Contains(t, remainingIDs, youngPendingSignIn.PendingSignInId)
	})

	t.Run("keeps the hourly cap counting the codes of the last hour after a pass", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t)

		for range 5 {
			startTwoFactorSignIn(t, app, admin.Email)
			app.Clock.Advance(11 * time.Minute)
		}

		app.RunJobPass(t, "users_expired_codes_cleanup")
		response := signIn(t, app, admin.Email, apptest.TestPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeTooManySignInCodes)
	})
}

// requireTwoFactorAfterMailServerLoss turns the second factor on directly, as on an instance that
// lost its mail server after an administrator turned it on; the API refuses to turn it on then.
func requireTwoFactorAfterMailServerLoss(t *testing.T, app *apptest.App, admin apptest.Session) {
	t.Helper()

	readSettings, err := admin.Client.GetInstanceSettingsWithResponse(t.Context())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, readSettings.StatusCode(), string(readSettings.Body))

	app.ExecuteSQL(t, "UPDATE users_settings SET is_two_factor_auth_required = TRUE")
}

// newTwoFactorApp builds an instance whose administrator turned the second factor on.
func newTwoFactorApp(t *testing.T, options ...apptest.Option) (*apptest.App, apptest.Session) {
	t.Helper()

	app := apptest.New(t, options...)
	admin := app.CreateAdmin(t)
	twoFactorUpdate := defaultSettingsUpdate()
	twoFactorUpdate.IsTwoFactorAuthRequired = true

	response := updateInstanceSettings(t, admin.Client, twoFactorUpdate)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return app, admin
}

func startTwoFactorSignIn(t *testing.T, app *apptest.App, email string) apiclient.SignInCodeRequired {
	t.Helper()

	return requireCodeRequired(t, signIn(t, app, email, apptest.TestPassword))
}

func startTwoFactorSignInWithAnswer(t *testing.T, app *apptest.App, email string) apiclient.SignInCodeRequired {
	t.Helper()

	password := apptest.TestPassword
	answer := apptest.TurnstileValidAnswer

	response, err := app.NewClient(t).SignInWithResponse(t.Context(), apiclient.SignInRequest{
		Email:                    email,
		Password:                 &password,
		CloudflareTurnstileToken: &answer,
	})
	require.NoError(t, err)

	return requireCodeRequired(t, response)
}

func requireCodeRequired(t *testing.T, response *apiclient.SignInResponse) apiclient.SignInCodeRequired {
	t.Helper()

	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	codeRequired, err := response.JSON200.AsSignInCodeRequired()
	require.NoError(t, err)
	require.Equal(t, apiclient.SignInCodeRequiredOutcomeCODEREQUIRED, codeRequired.Outcome, string(response.Body))

	return codeRequired
}

// getWrongCode returns a six-digit code that differs from the correct one.
func getWrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}

	return "000000"
}

func verifySignInCode(
	t *testing.T,
	app *apptest.App,
	pendingSignInID uuid.UUID,
	code string,
) *apiclient.VerifySignInCodeResponse {
	t.Helper()

	response, err := app.NewClient(t).VerifySignInCodeWithResponse(t.Context(), pendingSignInID,
		apiclient.VerifySignInCodeRequest{Code: &code, CloudflareTurnstileToken: nil})
	require.NoError(t, err)

	return response
}

func resendSignInCode(t *testing.T, app *apptest.App, pendingSignInID uuid.UUID) *apiclient.ResendSignInCodeResponse {
	t.Helper()

	response, err := app.NewClient(t).ResendSignInCodeWithResponse(t.Context(), pendingSignInID,
		apiclient.ResendSignInCodeRequest{CloudflareTurnstileToken: nil})
	require.NoError(t, err)

	return response
}
