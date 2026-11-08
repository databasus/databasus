package users_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

type passwordlessSignInScenario struct {
	name            string
	options         []apptest.Option
	hasGoogleLink   bool
	hasGitHubLink   bool
	isPasswordEmpty bool
	code            apiclient.ProblemCode
}

func TestPasswordlessSignIn(t *testing.T) {
	t.Parallel()

	scenarios := []passwordlessSignInScenario{
		{
			name:          "recommends Google when only Google is linked",
			options:       []apptest.Option{apptest.WithGoogle(), apptest.WithGitHub()},
			hasGoogleLink: true,
			code:          apiclient.ProblemCodePasswordNotSetGoogle,
		},
		{
			name:          "recommends GitHub when only GitHub is linked",
			options:       []apptest.Option{apptest.WithGoogle(), apptest.WithGitHub()},
			hasGitHubLink: true,
			code:          apiclient.ProblemCodePasswordNotSetGithub,
		},
		{
			name:          "recommends either provider when both are linked and configured",
			options:       []apptest.Option{apptest.WithGoogle(), apptest.WithGitHub()},
			hasGoogleLink: true,
			hasGitHubLink: true,
			code:          apiclient.ProblemCodePasswordNotSetGoogleGithub,
		},
		{
			name:          "recommends only Google when linked GitHub is disabled",
			options:       []apptest.Option{apptest.WithGoogle()},
			hasGoogleLink: true,
			hasGitHubLink: true,
			code:          apiclient.ProblemCodePasswordNotSetGoogle,
		},
		{
			name:          "recommends only GitHub when linked Google is disabled",
			options:       []apptest.Option{apptest.WithGitHub()},
			hasGoogleLink: true,
			hasGitHubLink: true,
			code:          apiclient.ProblemCodePasswordNotSetGithub,
		},
		{
			name:          "recommends no provider when linked providers are disabled",
			hasGoogleLink: true,
			hasGitHubLink: true,
			code:          apiclient.ProblemCodePasswordNotSet,
		},
		{
			name:    "recommends no provider when configured providers are not linked",
			options: []apptest.Option{apptest.WithGoogle(), apptest.WithGitHub()},
			code:    apiclient.ProblemCodePasswordNotSet,
		},
		{
			name:            "identifies the missing password when its stored value is empty",
			isPasswordEmpty: true,
			code:            apiclient.ProblemCodePasswordNotSet,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			for _, isInactive := range []bool{false, true} {
				accountState := "with an active account"
				if isInactive {
					accountState = "with an inactive account"
				}

				t.Run("keeps the same guidance "+accountState, func(t *testing.T) {
					t.Parallel()

					app := apptest.New(t, scenario.options...)
					admin := app.CreateAdmin(t)
					account := createPasswordlessAccount(t, app, scenario)
					if isInactive {
						requireDeactivated(t, admin, account.UserId)
					}

					response := signIn(t, app, account.Email, apptest.TestPassword)

					requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized, scenario.code)
					assert.NotContains(t, string(response.Body), "deactivated")
					assert.NotContains(t, string(response.Body), "token")
					require.Equal(t, http.StatusOK, getUser(t, admin.Client, admin.UserID).StatusCode())

					twoFactorUpdate := defaultSettingsUpdate()
					twoFactorUpdate.IsTwoFactorAuthRequired = true
					updatedSettings := updateInstanceSettings(t, admin.Client, twoFactorUpdate)
					require.Equal(t, http.StatusOK, updatedSettings.StatusCode(), string(updatedSettings.Body))

					twoFactorResponse := signIn(t, app, account.Email, apptest.TestPassword)

					requireProblem(
						t,
						twoFactorResponse.StatusCode(),
						twoFactorResponse.Body,
						http.StatusUnauthorized,
						scenario.code,
					)
					assert.JSONEq(t, string(response.Body), string(twoFactorResponse.Body))
					assert.Zero(t, app.Mail.GetMessageCount())
					var pendingSignInCount int64
					app.QuerySQL(t, &pendingSignInCount, "SELECT COUNT(*) FROM users_pending_sign_ins")
					assert.Zero(t, pendingSignInCount)
				})
			}
		})
	}

	t.Run("accepts the password when an OAuth account sets its first password through recovery", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGoogle(), apptest.WithGitHub())
		app.CreateAdmin(t)
		account := requireGoogleSession(t, app, apptest.NewGoogleAccount(apptest.NewTestEmail("passwordless")))
		requireGitHubSession(t, app, apptest.NewGitHubAccount(account.Email))
		code := requestResetCode(t, app, account.Email)

		resetResponse := completePasswordReset(t, app, account.Email, code, resetPassword)

		require.Equal(t, http.StatusNoContent, resetResponse.StatusCode(), string(resetResponse.Body))
		completedSignIn := requireCompletedSignIn(t, signIn(t, app, account.Email, resetPassword))
		assert.Equal(t, account.UserId, completedSignIn.UserId)
		wrongPasswordResponse := signIn(t, app, account.Email, apptest.TestPassword)
		requireProblem(t, wrongPasswordResponse.StatusCode(), wrongPasswordResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodePasswordIncorrect)
	})
}

func createPasswordlessAccount(t *testing.T, app *apptest.App, scenario passwordlessSignInScenario) apiclient.Session {
	t.Helper()

	email := apptest.NewTestEmail("passwordless")
	var account apiclient.Session

	switch {
	case scenario.hasGoogleLink && app.Google != nil:
		account = requireGoogleSession(t, app, apptest.NewGoogleAccount(email))
	case scenario.hasGitHubLink && app.GitHub != nil:
		account = requireGitHubSession(t, app, apptest.NewGitHubAccount(email))
	default:
		registeredUser := app.SignUp(t, email)
		account = apiclient.Session{Email: email, UserId: registeredUser.UserID, Token: registeredUser.Token}
		// No API removes a password or links a provider that is disabled on this instance.
		app.ExecuteSQL(t, "UPDATE users SET hashed_password = NULL WHERE id = ?", account.UserId)
	}

	if scenario.hasGoogleLink {
		if app.Google != nil {
			requireGoogleSession(t, app, apptest.NewGoogleAccount(email))
		} else {
			app.ExecuteSQL(
				t,
				"UPDATE users SET google_oauth_id = ? WHERE id = ?",
				"unavailable-google-account",
				account.UserId,
			)
		}
	}

	if scenario.hasGitHubLink {
		if app.GitHub != nil {
			requireGitHubSession(t, app, apptest.NewGitHubAccount(email))
		} else {
			app.ExecuteSQL(
				t,
				"UPDATE users SET github_oauth_id = ? WHERE id = ?",
				"unavailable-github-account",
				account.UserId,
			)
		}
	}

	if scenario.isPasswordEmpty {
		app.ExecuteSQL(t, "UPDATE users SET hashed_password = '' WHERE id = ?", account.UserId)
	}

	return account
}
