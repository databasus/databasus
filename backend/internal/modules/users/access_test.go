package users_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestAccessMark(t *testing.T) {
	t.Parallel()

	t.Run("returns 401 unauthenticated without running a signed-in operation when no token is sent",
		func(t *testing.T) {
			app := apptest.New(t)
			admin := app.CreateAdmin(t)

			response := getUser(t, app.NewClient(t), admin.UserID)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
				apiclient.ProblemCodeUnauthenticated)
		})

	t.Run("returns 401 unauthenticated when the token is not a session token", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := getUser(t, app.NewSessionClient(t, "not-a-token"), admin.UserID)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("runs a public operation when no token is sent", func(t *testing.T) {
		app := apptest.New(t)

		response, err := app.NewClient(t).GetSignInOptionsWithResponse(t.Context())

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("accepts the raw token without the bearer prefix", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
			app.URL+"/api/v1/users/"+admin.UserID.String(), nil)
		require.NoError(t, err)
		request.Header.Set("Authorization", admin.Token)

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		_ = response.Body.Close()

		require.Equal(t, http.StatusOK, response.StatusCode)
	})
}

func TestPasswordPathsWithoutSession(t *testing.T) {
	t.Parallel()

	t.Run("returns 404 when an anonymous caller asks to read or set a password of a user", func(t *testing.T) {
		app := apptest.New(t)
		app.CreateAdmin(t)

		readStatus, _ := sendRawRequest(t, app, http.MethodGet, "/users/admin/has-password", "")
		setStatus, _ := sendRawRequest(t, app, http.MethodPost, "/users/admin/set-password",
			`{"password":"adminpassword123"}`)

		require.Equal(t, http.StatusNotFound, readStatus)
		require.Equal(t, http.StatusNotFound, setStatus)
	})
}
