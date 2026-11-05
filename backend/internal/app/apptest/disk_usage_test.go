package apptest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSystemDiskUsage(t *testing.T) {
	t.Parallel()

	t.Run("returns byte counts when ordinary users or admins sign in", func(t *testing.T) {
		t.Parallel()

		application := apptest.New(t)
		admin := application.CreateAdmin(t)
		member := application.CreateNonAdminUser(t, admin)

		for _, session := range []apptest.Session{admin, member} {
			response, err := session.Client.GetDiskUsageWithResponse(t.Context())
			require.NoError(t, err)
			require.NotNil(t, response.JSON200, string(response.Body))
			assert.Positive(t, response.JSON200.TotalSpaceBytes)
			assert.GreaterOrEqual(t, response.JSON200.UsedSpaceBytes, int64(0))
			assert.LessOrEqual(t, response.JSON200.FreeSpaceBytes, response.JSON200.TotalSpaceBytes)
		}

		for _, client := range []*apiclient.ClientWithResponses{application.NewClient(t), application.NewSessionClient(t, "invalid")} {
			response, err := client.GetDiskUsageWithResponse(t.Context())
			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
		}

		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			application.URL+"/api/v1/disk/usage",
			nil,
		)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer func() { require.NoError(t, response.Body.Close()) }()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
	})
}
