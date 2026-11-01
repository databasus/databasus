package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/diskusage"
)

func TestDiskInspectionFailure(t *testing.T) {
	t.Run("returns a safe internal problem when a signed-in user requests an unreadable volume", func(t *testing.T) {
		application := newTestApp(t)
		application.inspectDisk = func(string) (diskusage.Usage, error) { return diskusage.Usage{}, errors.New("private volume path") }

		handler, err := application.NewHandler()
		require.NoError(t, err)
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		publicClient := apiclient.New(t, server.URL)
		signedUp, err := requestSignUp(t.Context(), publicClient, newConsoleEmail())
		require.NoError(t, err)
		require.NotNil(t, signedUp.JSON201, string(signedUp.Body))

		client := apiclient.New(t, server.URL, apiclient.WithSessionToken(signedUp.JSON201.Token))
		response, err := client.GetDiskUsageWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode())
		assert.NotContains(t, string(response.Body), "private volume path")
	})
}
