package users_test

import (
	"net/http"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSignUp(t *testing.T) {
	t.Parallel()

	t.Run("makes the user an active admin when the instance has no user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		admin := app.SignUp(t, apptest.NewTestEmail("owner"))

		readUser := getUser(t, admin.Client, admin.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserRoleADMIN, readUser.JSON200.Role)
		assert.Equal(t, apiclient.UserStatusACTIVE, readUser.JSON200.Status)
	})

	t.Run("makes the user a member when the instance already has a user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.CreateAdmin(t)

		member := app.SignUp(t, apptest.NewTestEmail("member"))

		readUser := getUser(t, member.Client, member.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserRoleMEMBER, readUser.JSON200.Role)
	})

	t.Run("returns a session that signs the user in", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		email := apptest.NewTestEmail("owner")

		session := app.SignUp(t, email)

		assert.Equal(t, email, session.Email)
		readUser := getUser(t, session.Client, session.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, email, readUser.JSON200.Email)
	})

	t.Run("returns 409 email_taken when the address belongs to another user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := signUp(t, app, admin.Email, apptest.TestPassword, "Second User")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeEmailTaken)
	})

	t.Run("returns 400 naming the password when it is shorter than 8 characters", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		response := signUp(t, app, apptest.NewTestEmail("owner"), "short", "Owner")

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
		require.NotNil(t, response.ApplicationproblemJSON400.Errors)
		assert.Equal(t, "password", (*response.ApplicationproblemJSON400.Errors)[0].Field)
	})

	t.Run("returns 400 naming the name when it is empty", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		response := signUp(t, app, apptest.NewTestEmail("owner"), apptest.TestPassword, " ")

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
		require.NotNil(t, response.ApplicationproblemJSON400.Errors)
		assert.Equal(t, "name", (*response.ApplicationproblemJSON400.Errors)[0].Field)
	})

	t.Run("returns 400 and creates no user when the address is not an email address", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		status, body := sendRawRequest(t, app, http.MethodPost, "/users",
			`{"email":"not-an-address","password":"test-password-123","name":"Owner"}`)

		requireProblem(t, status, body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
		assert.False(t, getSignInOptions(t, app).HasAnyUser)
	})
}

func signUp(t *testing.T, app *apptest.App, email, password, name string) *apiclient.SignUpResponse {
	t.Helper()

	response, err := app.NewClient(t).SignUpWithResponse(t.Context(), apiclient.SignUpRequest{
		Email:                    openapi_types.Email(email),
		Password:                 &password,
		Name:                     name,
		CloudflareTurnstileToken: nil,
	})
	require.NoError(t, err)

	return response
}
