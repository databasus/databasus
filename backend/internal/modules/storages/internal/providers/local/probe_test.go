package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestProbe(t *testing.T) {
	t.Parallel()

	t.Run("leaves both folders empty when they are writable", func(t *testing.T) {
		t.Parallel()
		folders := providers.LocalFolders{TempFolder: t.TempDir(), DataFolder: t.TempDir()}

		err := Probe(t.Context(), ProbeSpec{Folders: folders, Logger: providertest.NewDiscardLogger()})

		require.NoError(t, err)
		assert.Empty(t, readFolder(t, folders.TempFolder))
		assert.Empty(t, readFolder(t, folders.DataFolder))
	})

	t.Run("names the save and removes the staging file when the data folder is missing", func(t *testing.T) {
		t.Parallel()
		folders := providers.LocalFolders{
			TempFolder: t.TempDir(),
			DataFolder: filepath.Join(t.TempDir(), "missing", "data"),
		}
		require.NoError(t, os.WriteFile(filepath.Dir(folders.DataFolder), []byte("a file"), 0o600))

		err := Probe(t.Context(), ProbeSpec{Folders: folders, Logger: providertest.NewDiscardLogger()})

		require.ErrorIs(t, err, ErrProbeSaveFailed)
		assert.Empty(t, readFolder(t, folders.TempFolder))
	})
}

func readFolder(t *testing.T, folder string) []os.DirEntry {
	t.Helper()

	entries, err := os.ReadDir(folder)
	require.NoError(t, err)

	return entries
}
