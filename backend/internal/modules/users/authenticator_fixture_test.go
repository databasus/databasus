package users_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func setTwoFactorRequirement(t *testing.T, admin apptest.Session, requirement apiclient.TwoFactorRequirement) {
	t.Helper()

	settingsUpdate := defaultSettingsUpdate()
	settingsUpdate.TwoFactorRequirement = requirement

	response := updateInstanceSettings(t, admin.Client, settingsUpdate)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func startProfileSetup(
	t *testing.T,
	session apptest.Session,
	code *string,
) *apiclient.StartAuthenticatorAppSetupResponse {
	t.Helper()

	response, err := session.Client.StartAuthenticatorAppSetupWithResponse(
		t.Context(), session.UserID, apiclient.SecondFactorProof{Code: code},
	)
	require.NoError(t, err)

	return response
}

func requireStartedSetup(
	t *testing.T,
	response *apiclient.StartAuthenticatorAppSetupResponse,
) apiclient.AuthenticatorAppSetup {
	t.Helper()

	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

func confirmProfileSetup(
	t *testing.T,
	session apptest.Session,
	setupID uuid.UUID,
	code string,
) *apiclient.ConfirmAuthenticatorAppSetupResponse {
	t.Helper()

	response, err := session.Client.ConfirmAuthenticatorAppSetupWithResponse(
		t.Context(), session.UserID, setupID, apiclient.AuthenticatorAppCode{Code: &code},
	)
	require.NoError(t, err)

	return response
}

func disconnectApp(t *testing.T, session apptest.Session, code *string) *apiclient.DisconnectAuthenticatorAppResponse {
	t.Helper()

	response, err := session.Client.DisconnectAuthenticatorAppWithResponse(
		t.Context(), session.UserID, apiclient.SecondFactorProof{Code: code},
	)
	require.NoError(t, err)

	return response
}

func regenerateRecoveryCodes(
	t *testing.T,
	session apptest.Session,
	code *string,
) *apiclient.RegenerateRecoveryCodesResponse {
	t.Helper()

	response, err := session.Client.RegenerateRecoveryCodesWithResponse(
		t.Context(), session.UserID, apiclient.SecondFactorProof{Code: code},
	)
	require.NoError(t, err)

	return response
}

func requireAuthenticatorCodeRequired(
	t *testing.T,
	response *apiclient.SignInResponse,
) apiclient.SignInAuthenticatorCodeRequired {
	t.Helper()

	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	codeRequired, err := response.JSON200.AsSignInAuthenticatorCodeRequired()
	require.NoError(t, err)
	require.Equal(t, apiclient.SignInAuthenticatorCodeRequiredOutcomeAUTHENTICATORCODEREQUIRED, codeRequired.Outcome,
		string(response.Body))

	return codeRequired
}

func requireAuthenticatorSetupRequired(
	t *testing.T,
	response *apiclient.SignInResponse,
) apiclient.SignInAuthenticatorSetupRequired {
	t.Helper()

	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	setupRequired, err := response.JSON200.AsSignInAuthenticatorSetupRequired()
	require.NoError(t, err)
	require.Equal(t, apiclient.SignInAuthenticatorSetupRequiredOutcomeAUTHENTICATORSETUPREQUIRED, setupRequired.Outcome,
		string(response.Body))

	return setupRequired
}

func confirmSignInSetup(
	t *testing.T,
	app *apptest.App,
	pendingSignInID uuid.UUID,
	code string,
) *apiclient.ConfirmSignInAuthenticatorAppSetupResponse {
	t.Helper()

	response, err := app.NewClient(t).ConfirmSignInAuthenticatorAppSetupWithResponse(t.Context(), pendingSignInID,
		apiclient.VerifySignInCodeRequest{Code: &code, CloudflareTurnstileToken: nil})
	require.NoError(t, err)

	return response
}

func startAppCodeSignIn(t *testing.T, app *apptest.App, email string) apiclient.SignInAuthenticatorCodeRequired {
	t.Helper()

	return requireAuthenticatorCodeRequired(t, signIn(t, app, email, apptest.TestPassword))
}

func getAppState(t *testing.T, session apptest.Session) apiclient.User {
	t.Helper()

	response := getUser(t, session.Client, session.UserID)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func countWrongCodes(t *testing.T, app *apptest.App, userID uuid.UUID) int {
	t.Helper()

	var wrongCodeCount int
	app.QuerySQL(t, &wrongCodeCount, "SELECT count(*) FROM users_second_factor_failures WHERE user_id = ?", userID)

	return wrongCodeCount
}

func findAuditEvent(t *testing.T, admin apptest.Session, message string) apiclient.AuditEvent {
	t.Helper()

	for _, event := range apptest.ListAllAuditEvents(t, admin, apiclient.ListAuditEventsParams{}) {
		if event.Message == message {
			return event
		}
	}

	require.Failf(t, "no audit event", "no audit event says %q", message)

	return apiclient.AuditEvent{}
}
