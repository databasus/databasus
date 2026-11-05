package apptest_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestDownloadAgentBinary(t *testing.T) {
	t.Parallel()

	t.Run("serves bytes from the configured directory with the server version", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		require.NoError(
			t,
			os.WriteFile(
				filepath.Join(directory, "databasus-verification-agent-linux-"+runtime.GOARCH),
				[]byte("test agent"),
				0o755,
			),
		)

		app := apptest.New(t, apptest.WithAgentBinariesDir(directory))
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			app.URL+"/api/v1/system/verification-agent?arch="+runtime.GOARCH,
			nil,
		)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer func() { require.NoError(t, response.Body.Close()) }()
		contents, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "test agent", string(contents))
		assert.Equal(t, getTestVersion(t, app), response.Header.Get("X-Databasus-Version"))
	})
}

func getTestVersion(t *testing.T, application *apptest.App) string {
	t.Helper()
	response, err := application.NewClient(t).GetVersionWithResponse(t.Context())
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	return response.JSON200.Version
}
