package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	old_config "databasus-backend/internal/config"
	"databasus-backend/internal/util/encryption"
)

func TestProbeStorage(t *testing.T) {
	t.Run("saves and deletes the probe without a connection string", func(t *testing.T) {
		t.Setenv("ENV_MODE", "production")
		t.Setenv("DATABASE_DSN", "")

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

	t.Run("names the delete operation when the probe cannot be deleted", func(t *testing.T) {
		t.Setenv("ENV_MODE", "production")
		t.Setenv("DATABASE_DSN", "")

		environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
		require.NoError(t, err)
		old_config.Install(environment.Config)

		err = probeStorage(t.Context(), environment, &undeletableStorage{})

		var probeError *StorageProbeError
		require.ErrorAs(t, err, &probeError)
		assert.Contains(t, err.Error(), "delete a file through local storage")
		assert.Contains(t, err.Error(), storagePermissionsDocumentationURL)
	})
}

// undeletableStorage stands in for a volume that refuses deletes, which a test cannot cause on a
// real directory without also refusing the save before it.
type undeletableStorage struct{}

func (*undeletableStorage) SaveFile(
	context.Context, encryption.FieldEncryptor, *slog.Logger, string, io.Reader,
) error {
	return nil
}

func (*undeletableStorage) DeleteFile(context.Context, encryption.FieldEncryptor, *slog.Logger, string) error {
	return errors.New("operation not permitted")
}
