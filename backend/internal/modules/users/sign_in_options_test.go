package users_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestGetSignInOptions(t *testing.T) {
	t.Parallel()

	t.Run("answers that the instance has no user when nobody signed up", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		options := getSignInOptions(t, app)

		assert.False(t, options.HasAnyUser)
	})

	t.Run("answers that the instance has a user after the first sign-up", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.CreateAdmin(t)

		options := getSignInOptions(t, app)

		assert.True(t, options.HasAnyUser)
	})
}

// getSignInOptions calls the public operation without a session.
func getSignInOptions(t *testing.T, app *apptest.App) apiclient.SignInOptions {
	t.Helper()

	response, err := app.NewClient(t).GetSignInOptionsWithResponse(t.Context())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}
