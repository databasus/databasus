package users_test

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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
		t.Parallel()

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

	t.Run("records an audit event when a reset code is sent", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := requestPasswordReset(t, app, admin.Email)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		code := readLatestMailedCode(t, app, admin.Email)
		resetCodeEvents := listResetCodeAuditEvents(t, admin, admin.Email)
		require.Len(t, resetCodeEvents, 1)
		assert.Nil(t, resetCodeEvents[0].Author)
		assert.Nil(t, resetCodeEvents[0].Workspace)
		assert.NotContains(t, resetCodeEvents[0].Message, code)
	})

	t.Run("writes no audit event for an address without a user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		unknownEmail := apptest.NewTestEmail("unknown")

		response := requestPasswordReset(t, app, unknownEmail)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listResetCodeAuditEvents(t, admin, unknownEmail))
	})

	t.Run("writes no audit event when the mail server refuses the code", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.Mail.RefuseSends()

		response := requestPasswordReset(t, app, admin.Email)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Len(t, app.Mail.GetRefusedMessagesTo(admin.Email), 1)
		assert.Empty(t, listResetCodeAuditEvents(t, admin, admin.Email))
	})

	t.Run("answers alike when the audit event cannot be written", func(t *testing.T) {
		t.Parallel()

		logSink := &resetAuditLogSink{}
		app := apptest.New(t, apptest.WithLogSink(logSink))
		admin := app.CreateAdmin(t)
		app.ExecuteSQL(t, `CREATE FUNCTION refuse_reset_code_event() RETURNS trigger AS $$ BEGIN
			IF NEW.message LIKE 'Password reset code%' THEN RAISE EXCEPTION 'audit write refused'; END IF;
			RETURN NEW; END $$ LANGUAGE plpgsql`)
		app.ExecuteSQL(t, `CREATE TRIGGER refuse_reset_code_event BEFORE INSERT ON lib_audit_events
			FOR EACH ROW EXECUTE FUNCTION refuse_reset_code_event()`)

		response := requestPasswordReset(t, app, admin.Email)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 1)
		assert.True(t, logSink.isWriteFailureLogged())
	})

	t.Run("answers alike and mails nothing for an address without a user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.CreateAdmin(t)
		unknownEmail := apptest.NewTestEmail("unknown")

		response := requestPasswordReset(t, app, unknownEmail)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, app.Mail.GetMessagesTo(unknownEmail))
	})

	t.Run("answers alike and mails nothing for a deactivated user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		requireDeactivated(t, admin, member.UserID)

		response := requestPasswordReset(t, app, member.Email)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, app.Mail.GetMessagesTo(member.Email))
	})

	t.Run("answers alike and mails nothing for an invited user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		invitedEmail := inviteThroughWorkspace(t, admin)
		mailCountBeforeReset := len(app.Mail.GetMessagesTo(invitedEmail))

		response := requestPasswordReset(t, app, invitedEmail)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Len(t, app.Mail.GetMessagesTo(invitedEmail), mailCountBeforeReset)
	})

	t.Run("answers alike and mails nothing for the placeholder address", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.ExecuteSQL(t, "UPDATE users SET email = 'admin' WHERE id = ?", admin.UserID)

		response := requestPasswordReset(t, app, "admin")

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Zero(t, app.Mail.GetMessageCount())
	})

	t.Run("returns 429 rate_limited on the fourth request in an hour, for any address", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)

		wrongCodeResponse := completePasswordReset(t, app, admin.Email, getWrongCode(code), resetPassword)
		rightCodeResponse := completePasswordReset(t, app, admin.Email, code, resetPassword)

		requireProblem(t, wrongCodeResponse.StatusCode(), wrongCodeResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
		require.Equal(t, http.StatusNoContent, rightCodeResponse.StatusCode(), string(rightCodeResponse.Body))
	})

	t.Run("refuses the correct code after five wrong attempts", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)
		enterWrongResetCodes(t, app, wrongResetCodeEntries{email: admin.Email, correctCode: code, attemptCount: 5})

		response := completePasswordReset(t, app, admin.Email, code, resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, apptest.TestPassword))
	})

	t.Run("accepts the correct code after four wrong attempts", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)
		enterWrongResetCodes(t, app, wrongResetCodeEntries{email: admin.Email, correctCode: code, attemptCount: 4})

		response := completePasswordReset(t, app, admin.Email, code, resetPassword)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		requireCompletedSignIn(t, signIn(t, app, admin.Email, resetPassword))
	})

	t.Run("keeps the attempt count when the attempt fails", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)

		enterWrongResetCodes(t, app, wrongResetCodeEntries{email: admin.Email, correctCode: code, attemptCount: 1})

		assert.Equal(t, 1, getNewestResetCodeAttemptCount(t, app, admin))
	})

	t.Run("refuses the correct code after eight simultaneous wrong attempts", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)

		var waitGroup sync.WaitGroup
		for range 8 {
			waitGroup.Go(func() {
				completePasswordReset(t, app, admin.Email, getWrongCode(code), resetPassword)
			})
		}
		waitGroup.Wait()

		response := completePasswordReset(t, app, admin.Email, code, resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
		assert.Equal(t, 5, getNewestResetCodeAttemptCount(t, app, admin))
	})

	t.Run("completes one reset when two requests carry the same correct code at once", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)
		newPasswords := []string{resetPassword, "another-password-789"}
		responses := make([]*apiclient.CompletePasswordResetResponse, len(newPasswords))

		var waitGroup sync.WaitGroup
		for index, newPassword := range newPasswords {
			waitGroup.Go(func() {
				responses[index] = completePasswordReset(t, app, admin.Email, code, newPassword)
			})
		}
		waitGroup.Wait()

		acceptedIndex := 0
		if responses[1].StatusCode() == http.StatusNoContent {
			acceptedIndex = 1
		}

		refusedResponse := responses[1-acceptedIndex]
		require.Equal(
			t,
			http.StatusNoContent,
			responses[acceptedIndex].StatusCode(),
			string(responses[acceptedIndex].Body),
		)
		requireProblem(t, refusedResponse.StatusCode(), refusedResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, newPasswords[acceptedIndex]))
	})

	t.Run("refuses an older code while the newest one is usable", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		olderCode := requestResetCode(t, app, admin.Email)
		app.Clock.Advance(time.Second)
		requestResetCode(t, app, admin.Email)

		response := completePasswordReset(t, app, admin.Email, olderCode, resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	})

	t.Run("refuses an older code after the newest one is exhausted", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		olderCode := requestResetCode(t, app, admin.Email)
		app.Clock.Advance(time.Second)
		newestCode := requestResetCode(t, app, admin.Email)
		enterWrongResetCodes(
			t,
			app,
			wrongResetCodeEntries{email: admin.Email, correctCode: newestCode, attemptCount: 5},
		)

		response := completePasswordReset(t, app, admin.Email, olderCode, resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	})

	t.Run("refuses an older code after a completed reset", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		olderCode := requestResetCode(t, app, admin.Email)
		app.Clock.Advance(time.Second)
		newestCode := requestResetCode(t, app, admin.Email)
		completed := completePasswordReset(t, app, admin.Email, newestCode, resetPassword)
		require.Equal(t, http.StatusNoContent, completed.StatusCode(), string(completed.Body))

		response := completePasswordReset(t, app, admin.Email, olderCode, "another-password-789")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, resetPassword))
	})

	t.Run("accepts a new code after four wrong attempts on it when an exhausted one came before", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		exhaustedCode := requestResetCode(t, app, admin.Email)
		enterWrongResetCodes(
			t,
			app,
			wrongResetCodeEntries{email: admin.Email, correctCode: exhaustedCode, attemptCount: 5},
		)
		app.Clock.Advance(time.Second)
		newCode := requestResetCode(t, app, admin.Email)
		enterWrongResetCodes(t, app, wrongResetCodeEntries{email: admin.Email, correctCode: newCode, attemptCount: 4})

		response := completePasswordReset(t, app, admin.Email, newCode, resetPassword)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		requireCompletedSignIn(t, signIn(t, app, admin.Email, resetPassword))
	})

	t.Run("returns invalid_reset_code for a wrong code with a password too long to hash", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)

		response := completePasswordReset(t, app, admin.Email, getWrongCode(code), strings.Repeat("a", 73))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	})

	t.Run("answers alike for an unknown address, a deactivated user and a wrong code", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		memberCode := requestResetCode(t, app, member.Email)
		requireDeactivated(t, admin, member.UserID)
		code := requestResetCode(t, app, admin.Email)

		wrongCodeResponse := completePasswordReset(t, app, admin.Email, getWrongCode(code), resetPassword)
		unknownResponse := completePasswordReset(t, app, apptest.NewTestEmail("unknown"), code, resetPassword)
		deactivatedResponse := completePasswordReset(t, app, member.Email, memberCode, resetPassword)

		require.Equal(t, http.StatusBadRequest, wrongCodeResponse.StatusCode(), string(wrongCodeResponse.Body))
		assert.Equal(t, wrongCodeResponse.StatusCode(), unknownResponse.StatusCode())
		assert.Equal(t, wrongCodeResponse.StatusCode(), deactivatedResponse.StatusCode())
		assert.Equal(
			t,
			wrongCodeResponse.ApplicationproblemJSON400.Code,
			unknownResponse.ApplicationproblemJSON400.Code,
		)
		assert.Equal(
			t,
			wrongCodeResponse.ApplicationproblemJSON400.Code,
			deactivatedResponse.ApplicationproblemJSON400.Code,
		)
		assert.Equal(
			t,
			wrongCodeResponse.ApplicationproblemJSON400.Detail,
			unknownResponse.ApplicationproblemJSON400.Detail,
		)
		assert.Equal(
			t,
			wrongCodeResponse.ApplicationproblemJSON400.Detail,
			deactivatedResponse.ApplicationproblemJSON400.Detail,
		)

		var isMemberCodeUsed bool
		app.QuerySQL(
			t,
			&isMemberCodeUsed,
			"SELECT is_used FROM users_password_reset_codes WHERE user_id = ?",
			member.UserID,
		)
		assert.False(t, isMemberCodeUsed)
	})

	t.Run("returns 400 naming the password without spending an attempt when a wrong code comes with a short password",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			code := requestResetCode(t, app, admin.Email)

			response := completePasswordReset(t, app, admin.Email, getWrongCode(code), "short")

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			require.NotNil(t, response.ApplicationproblemJSON400.Errors)
			assert.Equal(t, "newPassword", (*response.ApplicationproblemJSON400.Errors)[0].Field)
			assert.Zero(t, getNewestResetCodeAttemptCount(t, app, admin))
		})

	t.Run("returns 400 invalid_reset_code for a used code", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		code := requestResetCode(t, app, admin.Email)
		app.Clock.Advance(time.Hour + time.Second)

		response := completePasswordReset(t, app, admin.Email, code, resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	})

	t.Run("returns 400 invalid_reset_code for an address without a user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.CreateAdmin(t)

		response := completePasswordReset(t, app, apptest.NewTestEmail("unknown"), "123456", resetPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	})

	t.Run("returns 400 naming the password when it is shorter than 8 characters", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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

type wrongResetCodeEntries struct {
	email        string
	correctCode  string
	attemptCount int
}

func enterWrongResetCodes(t *testing.T, app *apptest.App, entries wrongResetCodeEntries) {
	t.Helper()

	for range entries.attemptCount {
		response := completePasswordReset(t, app, entries.email, getWrongCode(entries.correctCode), resetPassword)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidResetCode)
	}
}

func getNewestResetCodeAttemptCount(t *testing.T, app *apptest.App, user apptest.Session) int {
	t.Helper()

	var attemptCount int
	app.QuerySQL(
		t,
		&attemptCount,
		"SELECT failed_attempt_count FROM users_password_reset_codes WHERE user_id = ? ORDER BY created_at DESC LIMIT 1",
		user.UserID,
	)

	return attemptCount
}

func requestResetCode(t *testing.T, app *apptest.App, email string) string {
	t.Helper()

	response := requestPasswordReset(t, app, email)
	require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))

	return readLatestMailedCode(t, app, email)
}

// listResetCodeAuditEvents reads the global audit log and keeps the events about the address's code.
func listResetCodeAuditEvents(t *testing.T, admin apptest.Session, email string) []apiclient.AuditEvent {
	t.Helper()

	var resetCodeEvents []apiclient.AuditEvent

	for _, event := range apptest.ListAllAuditEvents(t, admin, apiclient.ListAuditEventsParams{}) {
		if strings.HasPrefix(event.Message, "Password reset code sent to "+email) {
			resetCodeEvents = append(resetCodeEvents, event)
		}
	}

	return resetCodeEvents
}

// resetAuditLogSink notices the error record of a reset code event that could not be written.
type resetAuditLogSink struct {
	isLogged atomic.Bool
}

func (s *resetAuditLogSink) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelError
}

func (s *resetAuditLogSink) Handle(_ context.Context, record slog.Record) error {
	if strings.Contains(record.Message, "audit event of the reset code") {
		s.isLogged.Store(true)
	}

	return nil
}

func (s *resetAuditLogSink) WithAttrs([]slog.Attr) slog.Handler { return s }

func (s *resetAuditLogSink) WithGroup(string) slog.Handler { return s }

func (s *resetAuditLogSink) isWriteFailureLogged() bool {
	return s.isLogged.Load()
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
