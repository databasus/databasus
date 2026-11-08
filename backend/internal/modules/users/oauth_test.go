package users_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const oauthRedirectURI = "http://localhost:3000/auth/callback"

func TestSignInWithGitHub(t *testing.T) {
	t.Parallel()

	t.Run("makes the user an active admin when the instance has no user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		email := apptest.NewTestEmail("owner")

		session := requireGitHubSession(t, app, apptest.NewGitHubAccount(email))

		assert.Equal(t, email, session.Email)
		readUser := getUser(t, app.NewSessionClient(t, session.Token), session.UserId)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserRoleADMIN, readUser.JSON200.Role)
		assert.Equal(t, apiclient.UserStatusACTIVE, readUser.JSON200.Status)
		assert.Equal(t, "GitHub User", readUser.JSON200.Name)
	})

	t.Run("makes the user a member when the instance already has a user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		app.CreateAdmin(t)

		session := requireGitHubSession(t, app, apptest.NewGitHubAccount(apptest.NewTestEmail("member")))

		readUser := getUser(t, app.NewSessionClient(t, session.Token), session.UserId)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserRoleMEMBER, readUser.JSON200.Role)
	})

	t.Run("signs in the existing user with the same address and links the account", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		admin := app.CreateAdmin(t)
		gitHubAccount := apptest.NewGitHubAccount(admin.Email)

		firstSession := requireGitHubSession(t, app, gitHubAccount)

		assert.Equal(t, admin.UserID, firstSession.UserId)

		gitHubAccount.PublicEmail = apptest.NewTestEmail("renamed")
		linkedSession := requireGitHubSession(t, app, gitHubAccount)
		assert.Equal(t, admin.UserID, linkedSession.UserId)
		assert.Equal(t, admin.Email, linkedSession.Email)
	})

	t.Run("takes the primary verified address when the profile shows none", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		email := apptest.NewTestEmail("private")
		gitHubAccount := apptest.NewGitHubAccount("")
		gitHubAccount.Emails = []apptest.GitHubEmail{
			{Email: apptest.NewTestEmail("unverified"), IsPrimary: false, IsVerified: false},
			{Email: email, IsPrimary: true, IsVerified: true},
		}

		session := requireGitHubSession(t, app, gitHubAccount)

		assert.Equal(t, email, session.Email)
	})

	t.Run("returns 403 registration_closed for a new user when registration is closed", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		admin := app.CreateAdmin(t)
		closedRegistration := defaultSettingsUpdate()
		closedRegistration.IsAllowExternalRegistrations = false
		updatedSettings := updateInstanceSettings(t, admin.Client, closedRegistration)
		require.Equal(t, http.StatusOK, updatedSettings.StatusCode(), string(updatedSettings.Body))

		response := signInWithGitHub(
			t,
			app,
			app.GitHub.Authorize(apptest.NewGitHubAccount(apptest.NewTestEmail("new"))),
		)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodeRegistrationClosed)
	})

	t.Run("returns 403 user_deactivated for a deactivated user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		admin := app.CreateAdmin(t)
		gitHubAccount := apptest.NewGitHubAccount(apptest.NewTestEmail("member"))
		member := requireGitHubSession(t, app, gitHubAccount)
		requireDeactivated(t, admin, member.UserId)

		response := signInWithGitHub(t, app, app.GitHub.Authorize(gitHubAccount))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodeUserDeactivated)
	})

	t.Run("returns 403 user_deactivated when the address belongs to a deactivated user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		requireDeactivated(t, admin, member.UserID)

		response := signInWithGitHub(t, app, app.GitHub.Authorize(apptest.NewGitHubAccount(member.Email)))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodeUserDeactivated)
	})

	t.Run("returns 400 oauth_failed when the account has no verified address", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		gitHubAccount := apptest.NewGitHubAccount("")
		gitHubAccount.Emails = []apptest.GitHubEmail{
			{Email: apptest.NewTestEmail("unverified"), IsPrimary: true, IsVerified: false},
		}

		response := signInWithGitHub(t, app, app.GitHub.Authorize(gitHubAccount))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeOauthFailed)
		assert.False(t, getSignInOptions(t, app).HasAnyUser)
	})

	t.Run("returns 400 oauth_failed when GitHub refuses the code", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())

		response := signInWithGitHub(t, app, "code-github-never-issued")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeOauthFailed)
	})

	t.Run("returns 409 oauth_not_configured when the instance has no GitHub client", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		response := signInWithGitHub(t, app, "any-code")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeOauthNotConfigured)
	})

	t.Run("issues the session directly when the instance requires the second factor", func(t *testing.T) {
		t.Parallel()

		app, admin := newTwoFactorApp(t, apptest.WithGitHub())

		session := requireGitHubSession(t, app, apptest.NewGitHubAccount(admin.Email))

		assert.Equal(t, admin.UserID, session.UserId)
		assert.Zero(t, app.Mail.GetMessageCount())
	})

	t.Run("identifies the missing password and GitHub when the user was created through GitHub", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())
		email := apptest.NewTestEmail("owner")
		requireGitHubSession(t, app, apptest.NewGitHubAccount(email))

		response := signIn(t, app, email, apptest.TestPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodePasswordNotSetGithub)
		assert.True(t, getSignInOptions(t, app).HasAnyUser)
	})
}

func TestSignInWithGoogle(t *testing.T) {
	t.Parallel()

	t.Run("makes the user an active admin when the instance has no user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGoogle())
		email := apptest.NewTestEmail("owner")

		session := requireGoogleSession(t, app, apptest.NewGoogleAccount(email))

		assert.Equal(t, email, session.Email)
		readUser := getUser(t, app.NewSessionClient(t, session.Token), session.UserId)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserRoleADMIN, readUser.JSON200.Role)
		assert.Equal(t, "Google User", readUser.JSON200.Name)
	})

	t.Run("signs in the existing user with the same address and links the account", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGoogle())
		admin := app.CreateAdmin(t)
		googleAccount := apptest.NewGoogleAccount(admin.Email)

		requireGoogleSession(t, app, googleAccount)

		googleAccount.Email = apptest.NewTestEmail("renamed")
		linkedSession := requireGoogleSession(t, app, googleAccount)
		assert.Equal(t, admin.UserID, linkedSession.UserId)
	})

	t.Run("returns 400 oauth_failed when the address is not verified", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGoogle())
		admin := app.CreateAdmin(t)
		googleAccount := apptest.NewGoogleAccount(admin.Email)
		googleAccount.IsEmailVerified = false

		response := signInWithGoogle(t, app, app.Google.Authorize(googleAccount))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeOauthFailed)
	})

	t.Run("returns 400 oauth_failed when the account has no address", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGoogle())

		response := signInWithGoogle(t, app, app.Google.Authorize(apptest.NewGoogleAccount("")))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest, apiclient.ProblemCodeOauthFailed)
	})

	t.Run("returns 409 oauth_not_configured when the instance has no Google client", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithGitHub())

		response := signInWithGoogle(t, app, "any-code")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeOauthNotConfigured)
	})
}

func signInWithGitHub(t *testing.T, app *apptest.App, code string) *apiclient.SignInWithGitHubResponse {
	t.Helper()

	response, err := app.NewClient(t).SignInWithGitHubWithResponse(t.Context(), apiclient.OAuthSignInRequest{
		Code:        code,
		RedirectUri: oauthRedirectURI,
	})
	require.NoError(t, err)

	return response
}

func requireGitHubSession(t *testing.T, app *apptest.App, account apptest.GitHubAccount) apiclient.Session {
	t.Helper()

	response := signInWithGitHub(t, app, app.GitHub.Authorize(account))
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func signInWithGoogle(t *testing.T, app *apptest.App, code string) *apiclient.SignInWithGoogleResponse {
	t.Helper()

	response, err := app.NewClient(t).SignInWithGoogleWithResponse(t.Context(), apiclient.OAuthSignInRequest{
		Code:        code,
		RedirectUri: oauthRedirectURI,
	})
	require.NoError(t, err)

	return response
}

func requireGoogleSession(t *testing.T, app *apptest.App, account apptest.GoogleAccount) apiclient.Session {
	t.Helper()

	response := signInWithGoogle(t, app, app.Google.Authorize(account))
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}
