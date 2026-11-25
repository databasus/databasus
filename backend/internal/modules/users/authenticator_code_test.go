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

func TestAuthenticatorCode(t *testing.T) {
	t.Parallel()

	t.Run("accepts the code of the previous period for a phone clock slightly behind", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		pendingSignIn := startAppCodeSignIn(t, app, admin.Email)
		app.Clock.Advance(time.Minute)

		response := verifySignInCode(t, app, pendingSignIn.PendingSignInId,
			app.ComputeAuthenticatorCode(t, connectedApp.Secret, -1))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("refuses a code generated two minutes ago", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		app.Clock.Advance(time.Minute)
		oldCode := app.ComputeAuthenticatorCode(t, connectedApp.Secret, 0)
		app.Clock.Advance(2 * time.Minute)

		response := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId, oldCode)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
	})

	t.Run("refuses a code that already completed a sign-in in the same period", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		code := app.GenerateAuthenticatorCode(t, connectedApp.Secret)
		firstResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId, code)
		require.Equal(t, http.StatusOK, firstResponse.StatusCode(), string(firstResponse.Body))

		replayedResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId, code)

		requireProblem(t, replayedResponse.StatusCode(), replayedResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSignInCodeIncorrect)
	})

	t.Run("refuses an app code spent at sign-in when the profile receives it", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		code := app.GenerateAuthenticatorCode(t, connectedApp.Secret)
		signInResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId, code)
		require.Equal(t, http.StatusOK, signInResponse.StatusCode(), string(signInResponse.Body))

		response := regenerateRecoveryCodes(t, admin, &code)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeSecondFactorInvalid)
	})

	t.Run("lets exactly one of two simultaneous requests with one code succeed", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		code := app.GenerateAuthenticatorCode(t, connectedApp.Secret)

		var waitGroup sync.WaitGroup
		responses := make([]*apiclient.RegenerateRecoveryCodesResponse, 2)
		requestErrors := make([]error, 2)
		for index := range responses {
			waitGroup.Go(func() {
				responses[index], requestErrors[index] = admin.Client.RegenerateRecoveryCodesWithResponse(
					t.Context(), admin.UserID, apiclient.SecondFactorProof{Code: &code},
				)
			})
		}
		waitGroup.Wait()

		statuses := make([]int, 0, len(responses))
		for index, response := range responses {
			require.NoError(t, requestErrors[index])
			statuses = append(statuses, response.StatusCode())
		}
		assert.ElementsMatch(t, []int{http.StatusCreated, http.StatusBadRequest}, statuses)
	})

	// A second password step answers the live pending sign-in of the first, so the loser is refused
	// either by the spent code or by the spent pending sign-in, depending on where it lost the race.
	t.Run("issues one session to two sign-ins verified at once with one code", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		pendingSignInIDs := []uuid.UUID{
			startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
		}
		code := app.GenerateAuthenticatorCode(t, connectedApp.Secret)
		client := app.NewClient(t)

		var waitGroup sync.WaitGroup
		responses := make([]*apiclient.VerifySignInCodeResponse, len(pendingSignInIDs))
		requestErrors := make([]error, len(pendingSignInIDs))
		for index, pendingSignInID := range pendingSignInIDs {
			waitGroup.Go(func() {
				responses[index], requestErrors[index] = client.VerifySignInCodeWithResponse(
					t.Context(), pendingSignInID,
					apiclient.VerifySignInCodeRequest{Code: &code, CloudflareTurnstileToken: nil},
				)
			})
		}
		waitGroup.Wait()

		var issuedSessionCount int
		for index, response := range responses {
			require.NoError(t, requestErrors[index])
			if response.StatusCode() == http.StatusOK {
				issuedSessionCount++
				continue
			}

			assert.Contains(t, []int{http.StatusBadRequest, http.StatusGone}, response.StatusCode(),
				string(response.Body))
		}
		assert.Equal(t, 1, issuedSessionCount)
	})

	t.Run("refuses the eleventh code within an hour with 429 even when it is correct", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		spendTenWrongCodes(t, app, admin, connectedApp)

		correctResponse := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			app.GenerateAuthenticatorCode(t, connectedApp.Secret))
		profileResponse := regenerateRecoveryCodes(t, admin, &connectedApp.RecoveryCodes[0])

		requireProblem(t, correctResponse.StatusCode(), correctResponse.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeTooManyWrongCodes)
		requireProblem(t, profileResponse.StatusCode(), profileResponse.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeTooManyWrongCodes)
		assert.Equal(t, 10, countWrongCodes(t, app, admin.UserID))
		assert.Equal(t, http.StatusOK, getUser(t, admin.Client, admin.UserID).StatusCode())
	})

	t.Run("does not count wrong codes confirming a setup toward the limit", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		for range 3 {
			startedSetup := requireStartedSetup(t, startProfileSetup(t, admin, nil))
			code := app.GenerateAuthenticatorCode(t, startedSetup.Secret)
			for range 4 {
				confirmProfileSetup(t, admin, startedSetup.AuthenticatorAppSetupId, getWrongCode(code))
			}
		}

		connectedApp := app.ConnectAuthenticatorApp(t, admin)

		response := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			app.GenerateAuthenticatorCode(t, connectedApp.Secret))
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Zero(t, countWrongCodes(t, app, admin.UserID))
	})

	t.Run("accepts a correct code again once the wrong codes are older than an hour", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		spendTenWrongCodes(t, app, admin, connectedApp)
		app.ExecuteSQL(t, "UPDATE users_second_factor_failures SET failed_at = failed_at - interval '61 minutes'")

		response := verifySignInCode(t, app, startAppCodeSignIn(t, app, admin.Email).PendingSignInId,
			app.GenerateAuthenticatorCode(t, connectedApp.Secret))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})
}

// spendTenWrongCodes submits ten wrong codes, half at sign-in and half in the profile, because the
// limit counts them across both.
func spendTenWrongCodes(
	t *testing.T,
	app *apptest.App,
	session apptest.Session,
	connectedApp apptest.ConnectedAuthenticatorApp,
) {
	t.Helper()

	wrongCode := getWrongCode(app.GenerateAuthenticatorCode(t, connectedApp.Secret))
	pendingSignIn := startAppCodeSignIn(t, app, session.Email)

	for range 5 {
		verifySignInCode(t, app, pendingSignIn.PendingSignInId, wrongCode)
	}

	for range 5 {
		response := regenerateRecoveryCodes(t, session, &wrongCode)
		require.Equal(t, http.StatusBadRequest, response.StatusCode(), string(response.Body))
	}

	require.Equal(t, 10, countWrongCodes(t, app, session.UserID))
}
