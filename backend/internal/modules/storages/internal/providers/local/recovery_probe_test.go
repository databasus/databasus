package local

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeBackupFolder(t *testing.T) {
	t.Parallel()

	t.Run("leaves only the existing backups after a successful probe", func(t *testing.T) {
		t.Parallel()
		folder := newFolderWithBackup(t)

		err := probeBackupFolder(folder, getOSProbeFiles())

		require.NoError(t, err)
		assertOnlyBackupLeft(t, folder)
	})

	t.Run("fails and removes the probe when it cannot be read back", func(t *testing.T) {
		t.Parallel()
		folder := newFolderWithBackup(t)
		files := probeFiles{
			readFile: func(string) ([]byte, error) { return nil, errProbeStepFailed },
			remove:   os.Remove,
		}

		err := probeBackupFolder(folder, files)

		require.ErrorIs(t, err, ErrBackupFolderProbeFailed)
		require.ErrorIs(t, err, errProbeStepFailed)
		assertOnlyBackupLeft(t, folder)
	})

	t.Run("fails and retries the removal when the probe cannot be removed", func(t *testing.T) {
		t.Parallel()
		folder := newFolderWithBackup(t)
		var removedPaths []string
		files := probeFiles{
			readFile: os.ReadFile,
			remove: func(path string) error {
				removedPaths = append(removedPaths, path)

				return errProbeStepFailed
			},
		}

		err := probeBackupFolder(folder, files)

		require.ErrorIs(t, err, ErrBackupFolderProbeFailed)
		require.ErrorIs(t, err, errProbeStepFailed)
		require.Len(t, removedPaths, 2)
		assert.Equal(t, removedPaths[0], removedPaths[1])
		assert.Equal(t, folder, filepath.Dir(removedPaths[0]))
		backup, err := os.ReadFile(filepath.Join(folder, "backup.dump"))
		require.NoError(t, err)
		assert.Equal(t, []byte("backup bytes"), backup)
	})
}

var errProbeStepFailed = errors.New("probe step failed")

func newFolderWithBackup(t *testing.T) string {
	t.Helper()

	folder := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(folder, "backup.dump"), []byte("backup bytes"), 0o600))

	return folder
}

func assertOnlyBackupLeft(t *testing.T, folder string) {
	t.Helper()

	entries, err := os.ReadDir(folder)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "backup.dump", entries[0].Name())
	backup, err := os.ReadFile(filepath.Join(folder, "backup.dump"))
	require.NoError(t, err)
	assert.Equal(t, []byte("backup bytes"), backup)
}
