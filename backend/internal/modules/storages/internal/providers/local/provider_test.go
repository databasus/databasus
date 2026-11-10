package local

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/storages/internal/providers"
)

func TestDeleteFile(t *testing.T) {
	t.Parallel()

	t.Run("removes the staging file when the write was interrupted", func(t *testing.T) {
		t.Parallel()

		provider, folders := newTestProvider(t)
		fileName := "interrupted"
		stagingPath := filepath.Join(folders.TempFolder, fileName)
		require.NoError(t, os.WriteFile(stagingPath, []byte("partial"), 0o600))

		err := provider.DeleteFile(t.Context(), newFileAccess(fileName))

		require.NoError(t, err)
		assert.NoFileExists(t, stagingPath)
	})

	t.Run("removes both paths when the file was published", func(t *testing.T) {
		t.Parallel()

		provider, folders := newTestProvider(t)
		fileName := "published"
		require.NoError(t, provider.SaveFile(t.Context(), newFileAccess(fileName), strings.NewReader("payload")))

		stagingPath := filepath.Join(folders.TempFolder, fileName)
		require.NoError(t, os.WriteFile(stagingPath, []byte("partial"), 0o600))

		err := provider.DeleteFile(t.Context(), newFileAccess(fileName))

		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(folders.DataFolder, fileName))
		assert.NoFileExists(t, stagingPath)
	})

	t.Run("succeeds when nothing exists", func(t *testing.T) {
		t.Parallel()

		provider, _ := newTestProvider(t)

		err := provider.DeleteFile(t.Context(), newFileAccess("never-written"))

		assert.NoError(t, err)
	})

	t.Run("succeeds when called twice", func(t *testing.T) {
		t.Parallel()

		provider, _ := newTestProvider(t)
		fileName := "twice"
		require.NoError(t, provider.SaveFile(t.Context(), newFileAccess(fileName), strings.NewReader("payload")))
		require.NoError(t, provider.DeleteFile(t.Context(), newFileAccess(fileName)))

		assert.NoError(t, provider.DeleteFile(t.Context(), newFileAccess(fileName)))
	})

	t.Run("returns the filesystem error when the removal is refused", func(t *testing.T) {
		t.Parallel()

		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}

		provider, folders := newTestProvider(t)
		fileName := filepath.Join("locked", "artifact")
		directory := filepath.Join(folders.DataFolder, "locked")
		require.NoError(t, os.MkdirAll(directory, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(folders.DataFolder, fileName), []byte("x"), 0o600))
		require.NoError(t, os.Chmod(directory, 0o500))

		t.Cleanup(func() {
			_ = os.Chmod(directory, 0o755)
		})

		err := provider.DeleteFile(t.Context(), newFileAccess(fileName))

		assert.Error(t, err, "an unconfirmed absence must not be reported as a successful deletion")
	})
}

func TestSaveFile(t *testing.T) {
	t.Parallel()

	t.Run("publishes the file into the data folder and leaves no staging copy", func(t *testing.T) {
		t.Parallel()

		provider, folders := newTestProvider(t)
		fileName := filepath.Join("nested", "backup")

		require.NoError(t, provider.SaveFile(t.Context(), newFileAccess(fileName), strings.NewReader("payload")))

		openedFile, err := provider.OpenFile(t.Context(), newFileAccess(fileName))
		require.NoError(t, err)

		defer func() { _ = openedFile.Close() }()

		storedContent, err := io.ReadAll(openedFile)
		require.NoError(t, err)
		assert.Equal(t, "payload", string(storedContent))
		assert.NoFileExists(t, filepath.Join(folders.TempFolder, fileName))
	})
}

func TestTestConnection(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when the temp folder is writable", func(t *testing.T) {
		t.Parallel()

		provider, _ := newTestProvider(t)

		err := provider.TestConnection(t.Context(), providers.Location{StorageID: uuid.New(), Settings: Settings{}})

		assert.NoError(t, err)
	})
}

func newTestProvider(t *testing.T) (*Provider, providers.LocalFolders) {
	t.Helper()

	folders := providers.LocalFolders{
		TempFolder: filepath.Join(t.TempDir(), "temp"),
		DataFolder: filepath.Join(t.TempDir(), "data"),
	}
	require.NoError(t, os.MkdirAll(folders.TempFolder, 0o755))
	require.NoError(t, os.MkdirAll(folders.DataFolder, 0o755))

	return New(Deps{Repositories: Repositories{settings: nil}, Folders: folders}), folders
}

func newFileAccess(fileName string) providers.FileAccess {
	return providers.FileAccess{
		StorageID: uuid.New(), Settings: Settings{},
		FileName: fileName,
		Logger:   slog.New(slog.DiscardHandler),
	}
}
