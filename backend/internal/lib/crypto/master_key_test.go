package crypto_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/crypto"
)

const storedMasterKey = "0123456789abcdef0123456789abcdef-stored"

func TestOpenOrCreateMasterKey(t *testing.T) {
	t.Parallel()

	t.Run("creates one key file shared by every opener when openers start at once", func(t *testing.T) {
		t.Parallel()
		masterKeyPath := filepath.Join(t.TempDir(), "data", "secret.key")

		const openerCount = 16
		openedKeys := make([]*crypto.MasterKey, openerCount)
		openErrors := make([]error, openerCount)
		startOpening := make(chan struct{})

		var finishedOpeners sync.WaitGroup
		for openerIndex := range openerCount {
			finishedOpeners.Go(func() {
				<-startOpening
				openedKeys[openerIndex], openErrors[openerIndex] = crypto.OpenOrCreateMasterKey(masterKeyPath)
			})
		}

		close(startOpening)
		finishedOpeners.Wait()

		storedKey, err := os.ReadFile(masterKeyPath)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(storedKey), 32)

		keyDirectoryEntries, err := os.ReadDir(filepath.Dir(masterKeyPath))
		require.NoError(t, err)
		assert.Len(t, keyDirectoryEntries, 1)

		for openerIndex := range openerCount {
			require.NoError(t, openErrors[openerIndex])
			assert.Equal(t, string(storedKey), openedKeys[openerIndex].GetSecret())
		}
	})

	t.Run("returns the stored key and leaves the file untouched when the key file exists", func(t *testing.T) {
		t.Parallel()
		masterKeyPath := writeMasterKeyFile(t, storedMasterKey)
		fileBeforeOpen, err := os.Stat(masterKeyPath)
		require.NoError(t, err)

		openedKey, err := crypto.OpenOrCreateMasterKey(masterKeyPath)

		require.NoError(t, err)
		assert.Equal(t, storedMasterKey, openedKey.GetSecret())
		assertFileContent(t, masterKeyPath, storedMasterKey)
		fileAfterOpen, err := os.Stat(masterKeyPath)
		require.NoError(t, err)
		assert.Equal(t, fileBeforeOpen.ModTime(), fileAfterOpen.ModTime())
	})

	t.Run("fails with the file's path and keeps the file when the key is shorter than 32 bytes", func(t *testing.T) {
		t.Parallel()
		masterKeyPath := writeMasterKeyFile(t, "short-key")

		_, err := crypto.OpenOrCreateMasterKey(masterKeyPath)

		require.ErrorIs(t, err, crypto.ErrMasterKeyTooShort)
		assert.Contains(t, err.Error(), masterKeyPath)
		assertFileContent(t, masterKeyPath, "short-key")
	})

	t.Run("fails with the file's path and keeps the file when the key file cannot be read", func(t *testing.T) {
		t.Parallel()
		skipUnderRoot(t)
		masterKeyPath := writeMasterKeyFile(t, storedMasterKey)
		require.NoError(t, os.Chmod(masterKeyPath, 0o000))
		t.Cleanup(func() { _ = os.Chmod(masterKeyPath, 0o600) })

		_, err := crypto.OpenOrCreateMasterKey(masterKeyPath)

		require.Error(t, err)
		assert.Contains(t, err.Error(), masterKeyPath)
		require.NoError(t, os.Chmod(masterKeyPath, 0o600))
		assertFileContent(t, masterKeyPath, storedMasterKey)
	})

	t.Run("fails with the file's path when the key directory refuses a new file", func(t *testing.T) {
		t.Parallel()
		skipUnderRoot(t)
		masterKeyPath := filepath.Join(newReadOnlyDirectory(t), "secret.key")

		_, err := crypto.OpenOrCreateMasterKey(masterKeyPath)

		require.Error(t, err)
		assert.Contains(t, err.Error(), masterKeyPath)
		assert.NoFileExists(t, masterKeyPath)
	})
}

func writeMasterKeyFile(t *testing.T, secret string) string {
	t.Helper()

	masterKeyPath := filepath.Join(t.TempDir(), "secret.key")
	require.NoError(t, os.WriteFile(masterKeyPath, []byte(secret), 0o600))

	return masterKeyPath
}

func newReadOnlyDirectory(t *testing.T) string {
	t.Helper()

	directory := filepath.Join(t.TempDir(), "read-only")
	require.NoError(t, os.Mkdir(directory, 0o500))
	t.Cleanup(func() { _ = os.Chmod(directory, 0o755) })

	return directory
}

func assertFileContent(t *testing.T, path, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}

func skipUnderRoot(t *testing.T) {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
}
