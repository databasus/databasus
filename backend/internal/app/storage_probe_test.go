package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeStorage(t *testing.T) {
	t.Run("saves and deletes the probe without a connection string", func(t *testing.T) {
		t.Setenv("ENV_MODE", "production")
		t.Setenv("DATABASE_DSN", "")
		t.Setenv("LOG_FILE_IS_ENABLED", "false")

		environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(environment.Config.TempFolder, 0o755))
		require.NoError(t, os.MkdirAll(environment.Config.DataFolder, 0o755))

		require.NoError(t, ProbeStorage(t.Context(), environment))

		leftProbes, err := filepath.Glob(filepath.Join(environment.Config.DataFolder, ".databasus-storage-test-*"))
		require.NoError(t, err)
		assert.Empty(t, leftProbes)
	})

	t.Run("names both paths when the data directory cannot be written", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}

		t.Setenv("ENV_MODE", "production")
		t.Setenv("DATABASE_DSN", "")
		t.Setenv("LOG_FILE_IS_ENABLED", "false")

		environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(environment.Config.TempFolder, 0o755))
		require.NoError(t, os.MkdirAll(environment.Config.DataFolder, 0o500))

		err = ProbeStorage(t.Context(), environment)

		var probeError *StorageProbeError
		require.ErrorAs(t, err, &probeError)
		assert.Contains(t, err.Error(), environment.Config.DataFolder)
		assert.Contains(t, err.Error(), environment.Config.TempFolder)
		assert.Contains(t, err.Error(), "save a file through local storage")
		assert.Contains(t, err.Error(), storagePermissionsDocumentationURL)

		leftTemporaryProbes, err := filepath.Glob(
			filepath.Join(environment.Config.TempFolder, ".databasus-storage-test-*"),
		)
		require.NoError(t, err)
		assert.Empty(t, leftTemporaryProbes)
	})
}
