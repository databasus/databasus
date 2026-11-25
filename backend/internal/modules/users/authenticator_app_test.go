package users_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestStartAuthenticatorAppSetup(t *testing.T) {
	t.Parallel()

	t.Run("answers a new secret with its otpauth address naming the issuer and the account", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		startedSetup := requireStartedSetup(t, startProfileSetup(t, admin, nil))

		address, err := url.Parse(startedSetup.OtpauthUri)
		require.NoError(t, err)
		assert.Equal(t, "otpauth", address.Scheme)
		assert.Equal(t, "totp", address.Host)
		assert.Equal(t, "/Databasus:"+admin.Email, address.Path)
		assert.Equal(t, startedSetup.Secret, address.Query().Get("secret"))
		assert.Equal(t, "Databasus", address.Query().Get("issuer"))
		assert.Equal(t, "SHA1", address.Query().Get("algorithm"))
		assert.Equal(t, "6", address.Query().Get("digits"))
		assert.Equal(t, "30", address.Query().Get("period"))
		assert.Len(t, startedSetup.Secret, 32)
		assert.WithinDuration(t, app.Clock.Now().Add(30*time.Minute), startedSetup.ExpiresAt, time.Second)
	})

	t.Run("returns 403 permission_denied when the setup is for another user", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)

		response, err := admin.Client.StartAuthenticatorAppSetupWithResponse(
			t.Context(), member.UserID, apiclient.SecondFactorProof{Code: nil},
		)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response, err := app.NewClient(t).StartAuthenticatorAppSetupWithResponse(
			t.Context(), admin.UserID, apiclient.SecondFactorProof{Code: nil},
		)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("leaves sign-in unchanged when the setup is never confirmed", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		requireStartedSetup(t, startProfileSetup(t, member, nil))

		response := signIn(t, app, member.Email, apptest.TestPassword)

		requireCompletedSignIn(t, response)
		assert.Nil(t, getAppState(t, member).AuthenticatorAppConnectedAt)
	})

	t.Run("refuses a code from a setup that was never confirmed at the emailed code step", func(t *testing.T) {
		t.Parallel()
		app, admin := newTwoFactorApp(t)
		member := app.CreateNonAdminUser(t, admin)
		abandonedSetup := requireStartedSetup(t, startProfileSetup(t, member, nil))
		pendingSignIn := startTwoFactorSignIn(t, app, member.Email)

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.GenerateAuthenticatorCode(t, abandonedSetup.Secret))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
	})
}

func TestConfirmAuthenticatorAppSetup(t *testing.T) {
	t.Parallel()

	t.Run("connects the app, answers ten recovery codes and asks the app at the next sign-in", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		startedSetup := requireStartedSetup(t, startProfileSetup(t, member, nil))
		code := app.GenerateAuthenticatorCode(t, startedSetup.Secret)

		response := confirmProfileSetup(t, member, startedSetup.AuthenticatorAppSetupId, code)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		require.Len(t, response.JSON200.RecoveryCodes, 10)
		distinctCodes := map[string]bool{}
		for _, recoveryCode := range response.JSON200.RecoveryCodes {
			assert.Regexp(t, `^[a-z2-7]{5}-[a-z2-7]{5}$`, recoveryCode)
			distinctCodes[recoveryCode] = true
		}
		assert.Len(t, distinctCodes, 10)
		assert.NotContains(t, string(response.Body), startedSetup.Secret)
		startAppCodeSignIn(t, app, member.Email)
		assert.Empty(t, app.Mail.GetMessagesTo(member.Email))
		connectedEvent := findAuditEvent(t, admin, "Authenticator app connected: "+member.Email)
		require.NotNil(t, connectedEvent.Author)
		assert.Equal(t, member.UserID, connectedEvent.Author.UserId)
	})

	t.Run("reads the profile with the connection date and ten codes but never the secret", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)

		response := getUser(t, admin.Client, admin.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		require.NotNil(t, response.JSON200.AuthenticatorAppConnectedAt)
		assert.WithinDuration(t, app.Clock.Now(), *response.JSON200.AuthenticatorAppConnectedAt, time.Second)
		require.NotNil(t, response.JSON200.UnusedRecoveryCodeCount)
		assert.Equal(t, 10, *response.JSON200.UnusedRecoveryCodeCount)
		assert.NotContains(t, string(response.Body), connectedApp.Secret)
		assert.NotContains(t, string(response.Body), "otpauth")
	})

	t.Run("stores the secret encrypted so the stored rows do not reveal it", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		pendingSetup := requireStartedSetup(t, startProfileSetup(t, admin,
			new(app.GenerateAuthenticatorCode(t, connectedApp.Secret))))

		var storedSecrets []string
		app.QuerySQL(t, &storedSecrets, `SELECT encrypted_secret FROM users_authenticator_apps
			UNION ALL SELECT encrypted_secret FROM users_authenticator_setups`)
		var storedRecoveryCodes []string
		app.QuerySQL(t, &storedRecoveryCodes, "SELECT hashed_code FROM users_recovery_codes")

		require.Len(t, storedSecrets, 2)
		for _, storedSecret := range storedSecrets {
			assert.True(t, strings.HasPrefix(storedSecret, "enc:"), storedSecret)
			assert.NotContains(t, storedSecret, connectedApp.Secret)
			assert.NotContains(t, storedSecret, pendingSetup.Secret)
		}
		for _, storedRecoveryCode := range storedRecoveryCodes {
			for _, recoveryCode := range connectedApp.RecoveryCodes {
				assert.NotContains(t, storedRecoveryCode, strings.ReplaceAll(recoveryCode, "-", ""))
			}
		}
	})

	t.Run("returns 400 second_factor_invalid for a wrong code without counting it toward the limit",
		func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			member := app.CreateNonAdminUser(t, admin)
			startedSetup := requireStartedSetup(t, startProfileSetup(t, member, nil))
			code := app.GenerateAuthenticatorCode(t, startedSetup.Secret)

			response := confirmProfileSetup(t, member, startedSetup.AuthenticatorAppSetupId, getWrongCode(code))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeSecondFactorInvalid)
			assert.Nil(t, getAppState(t, member).AuthenticatorAppConnectedAt)
			assert.Zero(t, countWrongCodes(t, app, member.UserID))
			requireCompletedSignIn(t, signIn(t, app, member.Email, apptest.TestPassword))
		})

	t.Run("returns 410 for the correct code after five wrong ones and lets a new setup connect", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		startedSetup := requireStartedSetup(t, startProfileSetup(t, admin, nil))
		code := app.GenerateAuthenticatorCode(t, startedSetup.Secret)
		for range 5 {
			response := confirmProfileSetup(t, admin, startedSetup.AuthenticatorAppSetupId, getWrongCode(code))
			require.Equal(t, http.StatusBadRequest, response.StatusCode(), string(response.Body))
		}

		response := confirmProfileSetup(t, admin, startedSetup.AuthenticatorAppSetupId, code)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodeAuthenticatorSetupNotUsable)
		assert.Nil(t, getAppState(t, admin).AuthenticatorAppConnectedAt)
		app.ConnectAuthenticatorApp(t, admin)
	})

	t.Run("returns 410 for a correct code thirty-one minutes after the setup started", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		startedSetup := requireStartedSetup(t, startProfileSetup(t, admin, nil))
		app.Clock.Advance(31 * time.Minute)
		code := app.GenerateAuthenticatorCode(t, startedSetup.Secret)

		response := confirmProfileSetup(t, admin, startedSetup.AuthenticatorAppSetupId, code)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusGone,
			apiclient.ProblemCodeAuthenticatorSetupNotUsable)
		assert.Nil(t, getAppState(t, admin).AuthenticatorAppConnectedAt)
	})

	t.Run("refuses a code of the first secret after a second setup started", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		firstSetup := requireStartedSetup(t, startProfileSetup(t, admin, nil))
		secondSetup := requireStartedSetup(t, startProfileSetup(t, admin, nil))
		firstSecretCode := app.GenerateAuthenticatorCode(t, firstSetup.Secret)

		replacedResponse := confirmProfileSetup(t, admin, firstSetup.AuthenticatorAppSetupId, firstSecretCode)
		wrongSecretResponse := confirmProfileSetup(t, admin, secondSetup.AuthenticatorAppSetupId, firstSecretCode)
		secondSecretResponse := confirmProfileSetup(t, admin, secondSetup.AuthenticatorAppSetupId,
			app.ComputeAuthenticatorCode(t, secondSetup.Secret, 0))

		requireProblem(t, replacedResponse.StatusCode(), replacedResponse.Body, http.StatusGone,
			apiclient.ProblemCodeAuthenticatorSetupNotUsable)
		requireProblem(t, wrongSecretResponse.StatusCode(), wrongSecretResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSecondFactorInvalid)
		require.Equal(t, http.StatusOK, secondSecretResponse.StatusCode(), string(secondSecretResponse.Body))
	})

	t.Run("returns 403 permission_denied when the setup is confirmed for another user", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		startedSetup := requireStartedSetup(t, startProfileSetup(t, member, nil))
		code := app.GenerateAuthenticatorCode(t, startedSetup.Secret)

		response, err := admin.Client.ConfirmAuthenticatorAppSetupWithResponse(t.Context(), member.UserID,
			startedSetup.AuthenticatorAppSetupId, apiclient.AuthenticatorAppCode{Code: &code})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Nil(t, getAppState(t, member).AuthenticatorAppConnectedAt)
	})
}

func TestReplaceAuthenticatorApp(t *testing.T) {
	t.Parallel()

	t.Run("keeps only the new phone working after a replacement started with the old one", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		oldApp := app.ConnectAuthenticatorApp(t, admin)
		newSetup := requireStartedSetup(t, startProfileSetup(t, admin,
			new(app.GenerateAuthenticatorCode(t, oldApp.Secret))))

		response := confirmProfileSetup(t, admin, newSetup.AuthenticatorAppSetupId,
			app.GenerateAuthenticatorCode(t, newSetup.Secret))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		require.Len(t, response.JSON200.RecoveryCodes, 10)
		oldCodeResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			app.GenerateAuthenticatorCode(t, oldApp.Secret))
		requireProblem(t, oldCodeResponse.StatusCode(), oldCodeResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
		oldRecoveryResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			oldApp.RecoveryCodes[0])
		requireProblem(t, oldRecoveryResponse.StatusCode(), oldRecoveryResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
		newCodeResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			app.GenerateAuthenticatorCode(t, newSetup.Secret))
		require.Equal(t, http.StatusOK, newCodeResponse.StatusCode(), string(newCodeResponse.Body))
		assert.Contains(t, listGlobalAuditMessages(t, admin), "Authenticator app replaced: "+admin.Email)
	})

	t.Run("keeps the old app and its recovery codes working when the replacement is abandoned", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		oldApp := app.ConnectAuthenticatorApp(t, admin)
		requireStartedSetup(t, startProfileSetup(t, admin, new(app.GenerateAuthenticatorCode(t, oldApp.Secret))))

		appCodeResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			app.GenerateAuthenticatorCode(t, oldApp.Secret))
		recoveryCodeResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			oldApp.RecoveryCodes[0])

		require.Equal(t, http.StatusOK, appCodeResponse.StatusCode(), string(appCodeResponse.Body))
		require.Equal(t, http.StatusOK, recoveryCodeResponse.StatusCode(), string(recoveryCodeResponse.Body))
	})

	t.Run("returns 400 second_factor_invalid and starts nothing without a current code", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		wrongCode := getWrongCode(app.GenerateAuthenticatorCode(t, connectedApp.Secret))

		for _, code := range []*string{nil, &wrongCode} {
			response := startProfileSetup(t, admin, code)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeSecondFactorInvalid)
		}
		var setupCount int
		app.QuerySQL(t, &setupCount, "SELECT count(*) FROM users_authenticator_setups")
		assert.Zero(t, setupCount)
		assert.Equal(t, 1, countWrongCodes(t, app, admin.UserID))
	})
}

func TestDisconnectAuthenticatorApp(t *testing.T) {
	t.Parallel()

	t.Run("removes the app and its recovery codes when the request carries a current code", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)

		response := disconnectApp(t, admin, new(app.GenerateAuthenticatorCode(t, connectedApp.Secret)))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Nil(t, response.JSON200.AuthenticatorAppConnectedAt)
		assert.Nil(t, response.JSON200.UnusedRecoveryCodeCount)
		var recoveryCodeCount int
		app.QuerySQL(t, &recoveryCodeCount, "SELECT count(*) FROM users_recovery_codes")
		assert.Zero(t, recoveryCodeCount)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, apptest.TestPassword))
		assert.Contains(t, listGlobalAuditMessages(t, admin), "Authenticator app disconnected: "+admin.Email)
	})

	t.Run("returns 400 second_factor_invalid and keeps the app without a current code", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		wrongCode := getWrongCode(app.GenerateAuthenticatorCode(t, connectedApp.Secret))

		for _, code := range []*string{nil, &wrongCode} {
			response := disconnectApp(t, admin, code)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeSecondFactorInvalid)
		}
		appState := getAppState(t, admin)
		assert.NotNil(t, appState.AuthenticatorAppConnectedAt)
		require.NotNil(t, appState.UnusedRecoveryCodeCount)
		assert.Equal(t, 10, *appState.UnusedRecoveryCodeCount)
	})

	t.Run("returns 409 authenticator_app_required while the instance requires the app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		setTwoFactorRequirement(t, admin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

		response := disconnectApp(t, admin, new(app.GenerateAuthenticatorCode(t, connectedApp.Secret)))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeAuthenticatorAppRequired)
		assert.NotNil(t, getAppState(t, admin).AuthenticatorAppConnectedAt)
	})

	t.Run("returns 403 permission_denied and changes nothing for another user's app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		memberApp := app.ConnectAuthenticatorApp(t, member)
		code := app.GenerateAuthenticatorCode(t, memberApp.Secret)

		disconnectResponse, err := admin.Client.DisconnectAuthenticatorAppWithResponse(
			t.Context(), member.UserID, apiclient.SecondFactorProof{Code: &code},
		)
		require.NoError(t, err)
		replaceResponse, err := admin.Client.StartAuthenticatorAppSetupWithResponse(
			t.Context(), member.UserID, apiclient.SecondFactorProof{Code: &code},
		)
		require.NoError(t, err)
		regenerateResponse, err := admin.Client.RegenerateRecoveryCodesWithResponse(
			t.Context(), member.UserID, apiclient.SecondFactorProof{Code: &code},
		)
		require.NoError(t, err)

		for _, response := range []struct {
			status int
			body   []byte
		}{
			{disconnectResponse.StatusCode(), disconnectResponse.Body},
			{replaceResponse.StatusCode(), replaceResponse.Body},
			{regenerateResponse.StatusCode(), regenerateResponse.Body},
		} {
			requireProblem(
				t,
				response.status,
				response.body,
				http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied,
			)
		}
		assert.NotNil(t, getAppState(t, member).AuthenticatorAppConnectedAt)
	})
}

func TestAuthenticatorAppRecoveryCodes(t *testing.T) {
	t.Parallel()

	t.Run("signs in with an unused recovery code once and shows one code fewer", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		memberApp := app.ConnectAuthenticatorApp(t, member)

		response := verifySignInCode(t, app, startAppCodeSignIn(t, app, member.Email).PendingSignInId,
			strings.ToUpper(memberApp.RecoveryCodes[0]))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		appState := getAppState(t, member)
		require.NotNil(t, appState.UnusedRecoveryCodeCount)
		assert.Equal(t, 9, *appState.UnusedRecoveryCodeCount)
		usedEvent := findAuditEvent(t, admin, "Recovery code used: "+member.Email)
		require.NotNil(t, usedEvent.Author)
		assert.Equal(t, member.UserID, usedEvent.Author.UserId)
		reusedResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, member.Email).PendingSignInId,
			memberApp.RecoveryCodes[0])
		requireProblem(t, reusedResponse.StatusCode(), reusedResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
	})

	t.Run("refuses the earlier set after regeneration and accepts the new one", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)

		response := regenerateRecoveryCodes(t, admin, new(app.GenerateAuthenticatorCode(t, connectedApp.Secret)))

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		require.Len(t, response.JSON201.RecoveryCodes, 10)
		assert.NotContains(t, response.JSON201.RecoveryCodes, connectedApp.RecoveryCodes[0])
		earlierResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			connectedApp.RecoveryCodes[1])
		requireProblem(t, earlierResponse.StatusCode(), earlierResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
		newResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			response.JSON201.RecoveryCodes[0])
		require.Equal(t, http.StatusOK, newResponse.StatusCode(), string(newResponse.Body))
		assert.Contains(t, listGlobalAuditMessages(t, admin), "Recovery codes regenerated: "+admin.Email)
	})

	t.Run("regenerates with a recovery code in place of an app code", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)

		response := regenerateRecoveryCodes(t, admin, &connectedApp.RecoveryCodes[0])

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
	})

	t.Run("returns 400 second_factor_invalid and keeps the codes without a current code", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		wrongCode := getWrongCode(app.GenerateAuthenticatorCode(t, connectedApp.Secret))

		for _, code := range []*string{nil, &wrongCode} {
			response := regenerateRecoveryCodes(t, admin, code)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeSecondFactorInvalid)
		}
		earlierResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			connectedApp.RecoveryCodes[0])
		require.Equal(t, http.StatusOK, earlierResponse.StatusCode(), string(earlierResponse.Body))
	})

	t.Run("returns 400 second_factor_invalid for a user without an app", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := regenerateRecoveryCodes(t, admin, new("abcde-fghij"))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSecondFactorInvalid)
	})
}
