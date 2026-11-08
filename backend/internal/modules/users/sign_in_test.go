package users_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSignIn(t *testing.T) {
	t.Parallel()

	t.Run("returns a session whose token reads the user when the password is correct", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		completedSignIn := requireCompletedSignIn(t, signIn(t, app, admin.Email, apptest.TestPassword))

		assert.Equal(t, admin.UserID, completedSignIn.UserId)
		assert.Equal(t, admin.Email, completedSignIn.Email)

		readUser := getUser(t, app.NewSessionClient(t, completedSignIn.Token), admin.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
	})

	t.Run("identifies the cause when the address is unknown or the password is wrong",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)

			unknownAddressResponse := signIn(t, app, apptest.NewTestEmail("unknown"), apptest.TestPassword)
			wrongPasswordResponse := signIn(t, app, admin.Email, "wrong-password-123")

			requireProblem(t, unknownAddressResponse.StatusCode(), unknownAddressResponse.Body,
				http.StatusUnauthorized, apiclient.ProblemCodeEmailNotFound)
			requireProblem(t, wrongPasswordResponse.StatusCode(), wrongPasswordResponse.Body,
				http.StatusUnauthorized, apiclient.ProblemCodePasswordIncorrect)
		})

	t.Run("mentions neither admin nor the user count when the address is mistyped on a single-user instance",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			app.CreateAdmin(t)

			response := signIn(t, app, apptest.NewTestEmail("mistyped"), apptest.TestPassword)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
				apiclient.ProblemCodeEmailNotFound)
			assert.NotContains(t, response.ApplicationproblemJSON401.Detail, "admin")
			assert.NotContains(t, response.ApplicationproblemJSON401.Detail, "1")
		})

	t.Run("returns 429 rate_limited after ten attempts for one address within a minute", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		for attempt := range 10 {
			response := signIn(t, app, admin.Email, "wrong-password-"+strconv.Itoa(attempt))
			require.Equal(t, http.StatusUnauthorized, response.StatusCode(), string(response.Body))
		}

		refusedResponse := signIn(t, app, admin.Email, apptest.TestPassword)

		requireProblem(t, refusedResponse.StatusCode(), refusedResponse.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeRateLimited)

		app.Clock.Advance(time.Minute)
		requireCompletedSignIn(t, signIn(t, app, admin.Email, apptest.TestPassword))
	})

	t.Run("signs in with the placeholder address admin when an upgraded instance still holds it",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			app.ExecuteSQL(t, "UPDATE users SET email = 'admin' WHERE id = ?", admin.UserID)

			completedSignIn := requireCompletedSignIn(t, signIn(t, app, "admin", apptest.TestPassword))

			assert.Equal(t, admin.UserID, completedSignIn.UserId)
		})
}
