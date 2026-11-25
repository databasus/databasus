package users_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestAuthenticatorSignIn(t *testing.T) {
	t.Parallel()

	t.Run("asks for an app code without mail when the user has an app and nothing is required",
		func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			member := app.CreateNonAdminUser(t, admin)
			app.ConnectAuthenticatorApp(t, member)

			startAppCodeSignIn(t, app, member.Email)

			assert.Empty(t, app.Mail.GetMessagesTo(member.Email))
		})

	t.Run("asks for an app code without mail under the emailed code method with the mail server down",
		func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t, apptest.WithoutMailServer())
			admin := app.CreateAdmin(t)
			connectedApp := app.ConnectAuthenticatorApp(t, admin)
			requireTwoFactorAfterMailServerLoss(t, app, admin)

			pendingSignIn := startAppCodeSignIn(t, app, admin.Email)
			response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
				app.GenerateAuthenticatorCode(t, connectedApp.Secret))

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			assert.Equal(t, http.StatusOK, getUser(t, app.NewSessionClient(t, response.JSON200.Token),
				admin.UserID).StatusCode())
		})

	t.Run("emails a code to a user without an app under the emailed code method", func(t *testing.T) {
		t.Parallel()
		app, admin := newTwoFactorApp(t)

		startTwoFactorSignIn(t, app, admin.Email)

		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 1)
	})

	t.Run("asks a user without an app for a setup under the app method", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, member.Email, apptest.TestPassword))

		assert.NotEmpty(t, setupRequired.Secret)
		assert.Contains(t, setupRequired.OtpauthUri, setupRequired.Secret)
		assert.WithinDuration(t, app.Clock.Now().Add(30*time.Minute), setupRequired.ExpiresAt, time.Second)
		assert.Empty(t, app.Mail.GetMessagesTo(member.Email))
	})

	t.Run("starts a setup instead of handing back a live emailed code after the method changed",
		func(t *testing.T) {
			t.Parallel()
			app, admin := newTwoFactorApp(t)
			member := app.CreateNonAdminUser(t, admin)
			emailedSignIn := startTwoFactorSignIn(t, app, member.Email)
			emailedCode := readLatestMailedCode(t, app, member.Email)
			setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

			requireAuthenticatorSetupRequired(t, signIn(t, app, member.Email, apptest.TestPassword))
			response := verifySignInCode(t, app, emailedSignIn.PendingSignInId, emailedCode)

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		})

	t.Run("asks for an app code after the user connected one while an emailed code was live",
		func(t *testing.T) {
			t.Parallel()
			app, admin := newTwoFactorApp(t)
			member := app.CreateNonAdminUser(t, admin)
			startTwoFactorSignIn(t, app, member.Email)
			app.ConnectAuthenticatorApp(t, member)

			startAppCodeSignIn(t, app, member.Email)

			assert.Len(t, app.Mail.GetMessagesTo(member.Email), 1)
		})

	t.Run("hands back the same pending sign-in when the password step repeats within ten minutes",
		func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			connectedApp := app.ConnectAuthenticatorApp(t, admin)
			firstPendingSignIn := startAppCodeSignIn(t, app, admin.Email)
			app.Clock.Advance(5 * time.Minute)

			repeatedPendingSignIn := startAppCodeSignIn(t, app, admin.Email)
			response := verifySignInCode(t, app, repeatedPendingSignIn.PendingSignInId,
				app.GenerateAuthenticatorCode(t, connectedApp.Secret))

			assert.Equal(t, firstPendingSignIn.PendingSignInId, repeatedPendingSignIn.PendingSignInId)
			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		})

	t.Run("refuses a correct app code eleven minutes after the password step", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		pendingSignIn := startAppCodeSignIn(t, app, admin.Email)
		app.Clock.Advance(11 * time.Minute)

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.GenerateAuthenticatorCode(t, connectedApp.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("destroys the pending sign-in after five wrong app codes", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		pendingSignIn := startAppCodeSignIn(t, app, admin.Email)
		code := app.GenerateAuthenticatorCode(t, connectedApp.Secret)
		for range 4 {
			response := verifySignInCode(t, app, pendingSignIn.PendingSignInId, getWrongCode(code))
			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeSignInCodeIncorrect)
		}
		fifthResponse := verifySignInCode(t, app, pendingSignIn.PendingSignInId, getWrongCode(code))

		correctResponse := verifySignInCode(t, app, pendingSignIn.PendingSignInId, code)

		requireProblem(t, fifthResponse.StatusCode(), fifthResponse.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
		requireProblem(t, correctResponse.StatusCode(), correctResponse.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("refuses to resend the code of a pending sign-in that waits for the app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.ConnectAuthenticatorApp(t, admin)
		pendingSignIn := startAppCodeSignIn(t, app, admin.Email)

		response := resendSignInCode(t, app, pendingSignIn.PendingSignInId)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodePendingSignInNotResendable)
		assert.Empty(t, app.Mail.GetMessagesTo(admin.Email))
	})

	t.Run("refuses the correct app code after the account was deactivated", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		memberApp := app.ConnectAuthenticatorApp(t, member)
		pendingSignIn := startAppCodeSignIn(t, app, member.Email)
		require.Equal(t, http.StatusOK, deactivateUser(t, admin.Client, member.UserID).StatusCode())

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.GenerateAuthenticatorCode(t, memberApp.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("refuses the correct app code after the password changed", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		pendingSignIn := startAppCodeSignIn(t, app, admin.Email)
		require.Equal(t, http.StatusNoContent,
			changeUserPassword(t, admin.Client, admin.UserID, "another-password-123").StatusCode())

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.GenerateAuthenticatorCode(t, connectedApp.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("refuses a code of the old app after an administrator reset it", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		memberApp := app.ConnectAuthenticatorApp(t, member)
		pendingSignIn := startAppCodeSignIn(t, app, member.Email)
		require.Equal(t, http.StatusOK, resetAuthenticatorApp(t, admin, member.UserID).StatusCode())

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.GenerateAuthenticatorCode(t, memberApp.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("refuses a code of the old app after the user replaced it", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		oldApp := app.ConnectAuthenticatorApp(t, admin)
		pendingSignIn := startAppCodeSignIn(t, app, admin.Email)
		newSetup := requireStartedSetup(t, startProfileSetup(t, admin,
			new(app.GenerateAuthenticatorCode(t, oldApp.Secret))))
		confirmResponse := confirmProfileSetup(t, admin, newSetup.AuthenticatorAppSetupId,
			app.GenerateAuthenticatorCode(t, newSetup.Secret))
		require.Equal(t, http.StatusOK, confirmResponse.StatusCode(), string(confirmResponse.Body))

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.GenerateAuthenticatorCode(t, oldApp.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})

	t.Run("keeps asking a user with an app for its code after the requirement is switched off",
		func(t *testing.T) {
			t.Parallel()
			app, admin := newTwoFactorApp(t)
			member := app.CreateNonAdminUser(t, admin)
			app.ConnectAuthenticatorApp(t, member)
			setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementOFF)

			startAppCodeSignIn(t, app, member.Email)
		})
}

func TestAuthenticatorSignInSetup(t *testing.T) {
	t.Parallel()

	t.Run("connects the app, answers the recovery codes once and signs the member in", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, member.Email, apptest.TestPassword))

		response := confirmSignInSetup(t, app, setupRequired.PendingSignInId,
			app.GenerateAuthenticatorCode(t, setupRequired.Secret))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Len(t, response.JSON200.RecoveryCodes, 10)
		assert.Equal(t, member.UserID, response.JSON200.UserId)
		appState := getAppState(t, apptest.Session{
			Client: app.NewSessionClient(t, response.JSON200.Token), UserID: member.UserID,
			Email: member.Email, Token: response.JSON200.Token,
		})
		assert.NotNil(t, appState.AuthenticatorAppConnectedAt)
		assert.NotContains(t, string(response.Body), setupRequired.Secret)
		startAppCodeSignIn(t, app, member.Email)
	})

	t.Run("lets every account sign in on an instance without a mail server", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithoutMailServer())
		admin := app.CreateAdmin(t)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, admin.Email, apptest.TestPassword))
		response := confirmSignInSetup(t, app, setupRequired.PendingSignInId,
			app.GenerateAuthenticatorCode(t, setupRequired.Secret))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("shows a new secret after the password step repeats and refuses the earlier one", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		firstSetup := requireAuthenticatorSetupRequired(t, signIn(t, app, admin.Email, apptest.TestPassword))
		secondSetup := requireAuthenticatorSetupRequired(t, signIn(t, app, admin.Email, apptest.TestPassword))
		firstSecretCode := app.GenerateAuthenticatorCode(t, firstSetup.Secret)

		firstPendingResponse := confirmSignInSetup(t, app, firstSetup.PendingSignInId, firstSecretCode)
		wrongSecretResponse := confirmSignInSetup(t, app, secondSetup.PendingSignInId, firstSecretCode)
		secondSecretResponse := confirmSignInSetup(t, app, secondSetup.PendingSignInId,
			app.ComputeAuthenticatorCode(t, secondSetup.Secret, 0))

		assert.NotEqual(t, firstSetup.Secret, secondSetup.Secret)
		requireProblem(t, firstPendingResponse.StatusCode(), firstPendingResponse.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
		requireProblem(t, wrongSecretResponse.StatusCode(), wrongSecretResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
		require.Equal(t, http.StatusOK, secondSecretResponse.StatusCode(), string(secondSecretResponse.Body))
	})

	t.Run("returns 400 human_verification_failed and connects no app when the challenge is not answered",
		func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t, apptest.WithTurnstile())
			admin := app.CreateAdmin(t)
			member := app.CreateNonAdminUser(t, admin)
			setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
			setupRequired := requireAuthenticatorSetupRequired(t,
				signInWithTurnstileAnswer(t, app, member.Email))

			response := confirmSignInSetup(t, app, setupRequired.PendingSignInId,
				app.GenerateAuthenticatorCode(t, setupRequired.Secret))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeHumanVerificationFailed)
			assert.Nil(t, response.JSON200)
			assert.Nil(t, getAppState(t, member).AuthenticatorAppConnectedAt)
		})

	t.Run("issues no session and connects no app when the setup is left unfinished", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

		requireAuthenticatorSetupRequired(t, signIn(t, app, member.Email, apptest.TestPassword))

		assert.Nil(t, getAppState(t, member).AuthenticatorAppConnectedAt)
	})

	t.Run("refuses a correct code thirty-one minutes after the password step", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, admin.Email, apptest.TestPassword))
		app.Clock.Advance(31 * time.Minute)

		response := confirmSignInSetup(t, app, setupRequired.PendingSignInId,
			app.GenerateAuthenticatorCode(t, setupRequired.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
		assert.Nil(t, getAppState(t, admin).AuthenticatorAppConnectedAt)
	})

	t.Run("destroys the pending setup after five wrong codes", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, admin.Email, apptest.TestPassword))
		code := app.GenerateAuthenticatorCode(t, setupRequired.Secret)
		for range 5 {
			confirmSignInSetup(t, app, setupRequired.PendingSignInId, getWrongCode(code))
		}

		response := confirmSignInSetup(t, app, setupRequired.PendingSignInId, code)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
		assert.Zero(t, countWrongCodes(t, app, admin.UserID))
	})

	t.Run("connects one app and issues one session for two simultaneous confirmations", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, admin.Email, apptest.TestPassword))
		code := app.GenerateAuthenticatorCode(t, setupRequired.Secret)
		client := app.NewClient(t)

		var waitGroup sync.WaitGroup
		responses := make([]*apiclient.ConfirmSignInAuthenticatorAppSetupResponse, 2)
		requestErrors := make([]error, 2)
		for index := range responses {
			waitGroup.Go(func() {
				responses[index], requestErrors[index] = client.ConfirmSignInAuthenticatorAppSetupWithResponse(
					t.Context(), setupRequired.PendingSignInId,
					apiclient.VerifySignInCodeRequest{Code: &code, CloudflareTurnstileToken: nil},
				)
			})
		}
		waitGroup.Wait()

		statuses := make([]int, 0, len(responses))
		for index, response := range responses {
			require.NoError(t, requestErrors[index])
			statuses = append(statuses, response.StatusCode())
		}
		assert.ElementsMatch(t, []int{http.StatusOK, http.StatusGone}, statuses)
		var appCount int
		app.QuerySQL(t, &appCount, "SELECT count(*) FROM users_authenticator_apps")
		assert.Equal(t, 1, appCount)
	})

	t.Run("refuses a correct code after the account was deactivated or its password changed", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		deactivatedMember := app.CreateNonAdminUser(t, admin)
		passwordChangedMember := app.CreateNonAdminUser(t, admin)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		deactivatedSetup := requireAuthenticatorSetupRequired(t,
			signIn(t, app, deactivatedMember.Email, apptest.TestPassword))
		passwordChangedSetup := requireAuthenticatorSetupRequired(t,
			signIn(t, app, passwordChangedMember.Email, apptest.TestPassword))
		require.Equal(t, http.StatusOK, deactivateUser(t, admin.Client, deactivatedMember.UserID).StatusCode())
		require.Equal(
			t,
			http.StatusNoContent,
			changeUserPassword(
				t,
				passwordChangedMember.Client,
				passwordChangedMember.UserID,
				"another-password-123",
			).StatusCode(),
		)

		deactivatedResponse := confirmSignInSetup(t, app, deactivatedSetup.PendingSignInId,
			app.GenerateAuthenticatorCode(t, deactivatedSetup.Secret))
		passwordChangedResponse := confirmSignInSetup(t, app, passwordChangedSetup.PendingSignInId,
			app.ComputeAuthenticatorCode(t, passwordChangedSetup.Secret, 0))

		requireProblem(t, deactivatedResponse.StatusCode(), deactivatedResponse.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
		requireProblem(t, passwordChangedResponse.StatusCode(), passwordChangedResponse.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
		var appCount int
		app.QuerySQL(t, &appCount, "SELECT count(*) FROM users_authenticator_apps")
		assert.Zero(t, appCount)
	})

	t.Run("completes the setup after the requirement was switched off", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, member.Email, apptest.TestPassword))
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementOFF)

		response := confirmSignInSetup(t, app, setupRequired.PendingSignInId,
			app.GenerateAuthenticatorCode(t, setupRequired.Secret))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("completes the setup without mail after the requirement was switched to the emailed code",
		func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			member := app.CreateNonAdminUser(t, admin)
			setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
			setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, member.Email, apptest.TestPassword))
			setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementEMAIL)

			response := confirmSignInSetup(t, app, setupRequired.PendingSignInId,
				app.GenerateAuthenticatorCode(t, setupRequired.Secret))

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			var memberAppCount int
			app.QuerySQL(t, &memberAppCount,
				"SELECT count(*) FROM users_authenticator_apps WHERE user_id = ?", member.UserID)
			assert.Equal(t, 1, memberAppCount)
			assert.Empty(t, app.Mail.GetMessagesTo(member.Email))
		})

	t.Run("refuses to complete a pending setup through the code verification", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)
		setupRequired := requireAuthenticatorSetupRequired(t, signIn(t, app, admin.Email, apptest.TestPassword))

		response := verifySignInCode(t, app, setupRequired.PendingSignInId,
			app.GenerateAuthenticatorCode(t, setupRequired.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodePendingSignInNotUsable)
	})
}

func resetAuthenticatorApp(
	t *testing.T,
	admin apptest.Session,
	userID apiclient.UserId,
) *apiclient.ResetAuthenticatorAppResponse {
	t.Helper()

	response, err := admin.Client.ResetAuthenticatorAppWithResponse(t.Context(), userID)
	require.NoError(t, err)

	return response
}
