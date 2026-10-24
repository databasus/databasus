package secrets

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
)

func TestGetSecretKey(t *testing.T) {
	t.Run("generates one key file shared by every call when calls start at once", func(t *testing.T) {
		secretKeyPath := pointSecretKeyPathToTempDir(t)
		secretKeyService := &SecretKeyService{}

		const callersCount = 16
		returnedKeys := make([]string, callersCount)
		returnedErrors := make([]error, callersCount)

		var startedCallers sync.WaitGroup
		var finishedCallers sync.WaitGroup
		startCalls := make(chan struct{})

		for callerIndex := range callersCount {
			startedCallers.Add(1)
			finishedCallers.Add(1)

			go func() {
				defer finishedCallers.Done()

				startedCallers.Done()
				<-startCalls
				returnedKeys[callerIndex], returnedErrors[callerIndex] = secretKeyService.GetSecretKey()
			}()
		}

		startedCallers.Wait()
		close(startCalls)
		finishedCallers.Wait()

		storedKey, err := os.ReadFile(secretKeyPath)
		require.NoError(t, err)
		require.NotEmpty(t, storedKey)

		keyDirEntries, err := os.ReadDir(filepath.Dir(secretKeyPath))
		require.NoError(t, err)
		require.Len(t, keyDirEntries, 1)

		for callerIndex := range callersCount {
			require.NoError(t, returnedErrors[callerIndex])
			assert.Equal(t, string(storedKey), returnedKeys[callerIndex])
		}
	})

	t.Run("returns stored key and leaves file untouched when key file exists", func(t *testing.T) {
		secretKeyPath := pointSecretKeyPathToTempDir(t)
		require.NoError(t, os.MkdirAll(filepath.Dir(secretKeyPath), 0o755))

		const storedKey = "stored-secret-key"
		require.NoError(t, os.WriteFile(secretKeyPath, []byte(storedKey), 0o600))

		fileBeforeCall, err := os.Stat(secretKeyPath)
		require.NoError(t, err)

		returnedKey, err := (&SecretKeyService{}).GetSecretKey()
		require.NoError(t, err)
		assert.Equal(t, storedKey, returnedKey)

		fileContentAfterCall, err := os.ReadFile(secretKeyPath)
		require.NoError(t, err)
		assert.Equal(t, storedKey, string(fileContentAfterCall))

		fileAfterCall, err := os.Stat(secretKeyPath)
		require.NoError(t, err)
		assert.Equal(t, fileBeforeCall.ModTime(), fileAfterCall.ModTime())
	})
}

// The key path lives in the process-wide config, so these tests do not run in parallel.
func pointSecretKeyPathToTempDir(t *testing.T) string {
	t.Helper()

	env := config.GetEnv()
	originalSecretKeyPath := env.SecretKeyPath
	t.Cleanup(func() { env.SecretKeyPath = originalSecretKeyPath })

	env.SecretKeyPath = filepath.Join(t.TempDir(), "data", "secret.key")

	return env.SecretKeyPath
}
