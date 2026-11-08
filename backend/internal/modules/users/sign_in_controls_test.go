package users_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSignInAttemptControls(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		accountKind string
		code        apiclient.ProblemCode
	}{
		{accountKind: "unknown", code: apiclient.ProblemCodeEmailNotFound},
		{accountKind: "active", code: apiclient.ProblemCodePasswordIncorrect},
		{accountKind: "invited", code: apiclient.ProblemCodeRegistrationIncomplete},
		{accountKind: "passwordless", code: apiclient.ProblemCodePasswordNotSetGithub},
		{accountKind: "inactive", code: apiclient.ProblemCodeUserDeactivated},
	} {
		t.Run("checks the challenge and address limit before returning "+string(scenario.code), func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t, apptest.WithTurnstile(), apptest.WithGitHub())
			admin := app.CreateAdmin(t)
			email := admin.Email
			password := "incorrect-password"
			status := http.StatusUnauthorized

			switch scenario.accountKind {
			case "unknown":
				email = apptest.NewTestEmail("unknown")
			case "invited":
				email = inviteThroughWorkspace(t, admin)
			case "passwordless":
				account := requireGitHubSession(t, app, apptest.NewGitHubAccount(apptest.NewTestEmail("oauth")))
				email = account.Email
			case "inactive":
				member := app.SignUp(t, apptest.NewTestEmail("inactive"))
				requireDeactivated(t, admin, member.UserID)
				email = member.Email
				password = apptest.TestPassword
				status = http.StatusForbidden
			case "active":
			}
			mailCount := app.Mail.GetMessageCount()

			challengeResponse := signIn(t, app, email, password)

			requireProblem(t, challengeResponse.StatusCode(), challengeResponse.Body, http.StatusBadRequest,
				apiclient.ProblemCodeHumanVerificationFailed)

			request := apiclient.SignInRequest{
				Email: email, Password: &password, CloudflareTurnstileToken: new(apptest.TurnstileValidAnswer),
			}
			for attempt := range 10 {
				if attempt == 1 {
					twoFactorUpdate := defaultSettingsUpdate()
					twoFactorUpdate.IsTwoFactorAuthRequired = true
					updatedSettings := updateInstanceSettings(t, admin.Client, twoFactorUpdate)
					require.Equal(t, http.StatusOK, updatedSettings.StatusCode(), string(updatedSettings.Body))
				}

				response, err := app.NewClient(t).SignInWithResponse(t.Context(), request)
				require.NoError(t, err)
				requireProblem(t, response.StatusCode(), response.Body, status, scenario.code)
				assert.NotContains(t, string(response.Body), "token")
			}

			rateLimitedResponse, err := app.NewClient(t).SignInWithResponse(t.Context(), request)

			require.NoError(t, err)
			requireProblem(t, rateLimitedResponse.StatusCode(), rateLimitedResponse.Body, http.StatusTooManyRequests,
				apiclient.ProblemCodeRateLimited)
			assert.Equal(t, mailCount, app.Mail.GetMessageCount())
			var pendingSignInCount int64
			app.QuerySQL(t, &pendingSignInCount, "SELECT COUNT(*) FROM users_pending_sign_ins")
			assert.Zero(t, pendingSignInCount)
		})
	}
}
