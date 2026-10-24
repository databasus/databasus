package systemhttpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	system_version "databasus-backend/internal/features/system/version"
	apiclient "databasus-backend/internal/lib/apiclient"
)

// Installed verification agents update themselves through this operation with their own parser, so
// its path and its body never change.
func TestGetVersion(t *testing.T) {
	t.Run("returns the running version when the request has no session", func(t *testing.T) {
		testApp := apptest.New(t)
		client := apiclient.New(t, testApp.URL)

		response, err := client.GetVersionWithResponse(t.Context())

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode())
		require.NotNil(t, response.JSON200)
		assert.Equal(t, system_version.GetAppVersion(), response.JSON200.Version)
	})

	t.Run("returns a body with the version field alone", func(t *testing.T) {
		testApp := apptest.New(t)
		client := apiclient.New(t, testApp.URL)

		response, err := client.GetVersionWithResponse(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "application/json", response.HTTPResponse.Header.Get("Content-Type"))
		assert.Equal(t,
			`{"version":"`+system_version.GetAppVersion()+`"}`,
			strings.TrimSuffix(string(response.Body), "\n"),
		)
	})

	t.Run("returns the request ID header with the version", func(t *testing.T) {
		testApp := apptest.New(t)
		client := apiclient.New(t, testApp.URL)

		response, err := client.GetVersionWithResponse(t.Context())

		require.NoError(t, err)
		assert.NotEmpty(t, response.HTTPResponse.Header.Get("X-Request-Id"))
	})
}
