package users_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestGetUser(t *testing.T) {
	t.Parallel()

	t.Run("returns the caller's own user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		response := getUser(t, member.Client, member.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, member.UserID, response.JSON200.Id)
		assert.Equal(t, member.Email, response.JSON200.Email)
		assert.Equal(t, "Test User", response.JSON200.Name)
	})

	t.Run("returns another user when the caller is an instance admin", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		response := getUser(t, admin.Client, member.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, member.Email, response.JSON200.Email)
	})

	t.Run("returns 403 permission_denied when a member reads another user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		response := getUser(t, member.Client, admin.UserID)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("returns 404 not_found when the user does not exist", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := getUser(t, admin.Client, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 400 invalid_request when the user id is not a uuid", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, app.URL+"/api/v1/users/not-a-uuid", nil)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+admin.Token)

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		_ = response.Body.Close()

		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	})
}
