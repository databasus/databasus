package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestServerLifecycle(t *testing.T) {
	t.Run("checks a target started once normally and stopped before any account as recoverable", func(t *testing.T) {
		databaseDsn := testdb.OpenEmpty(t)
		environment := newTestEnvironment(t, databaseDsn)
		startOrdinaryServerOnce(t, environment)
		generatedKey, err := os.ReadFile(environment.Config.SecretKeyPath)
		require.NoError(t, err)
		appliedBefore := readAppliedMigrations(t, databaseDsn)
		configureRecoveryFiles(t, environment, []byte("PGDMP not a Databasus backup"))

		_, err = OpenServerLifecycle(t.Context(), environment)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a complete, unmodified Databasus backup")
		keyAfter, readErr := os.ReadFile(environment.Config.SecretKeyPath)
		require.NoError(t, readErr)
		assert.Equal(t, generatedKey, keyAfter)
		assert.Equal(t, appliedBefore, readAppliedMigrations(t, databaseDsn))
	})

	t.Run("runs a console command beside a running server", func(t *testing.T) {
		environment := newTestEnvironment(t, testdb.Open(t))
		require.NoError(t, os.MkdirAll(environment.Config.RecoveryStateDirectory, 0o700))
		serverLock, err := OpenServerLifecycle(t.Context(), environment)
		require.NoError(t, err)
		defer func() { _ = serverLock.Close() }()

		consoleLock, err := OpenConsoleLifecycle(environment)

		require.NoError(t, err)
		require.NoError(t, consoleLock.Close())
	})
}

// startOrdinaryServerOnce is an owner's first start without recovery files: migrations run, the key
// is generated and the sign-in page loads. The process then stops.
func startOrdinaryServerOnce(t *testing.T, environment *Environment) {
	t.Helper()

	lifecycleLock, err := OpenServerLifecycle(t.Context(), environment)
	require.NoError(t, err)
	defer func() { require.NoError(t, lifecycleLock.Close()) }()

	application, err := New(t.Context(), environment)
	require.NoError(t, err)
	defer application.Close()

	handler, err := application.NewHandler()
	require.NoError(t, err)

	server := httptest.NewServer(handler)
	defer server.Close()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/v1/sign-in-options", nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)
}

func configureRecoveryFiles(t *testing.T, environment *Environment, archive []byte) {
	t.Helper()

	recoveryDirectory := filepath.Join(environment.Config.DataDirectory, "recovery")
	require.NoError(t, os.MkdirAll(recoveryDirectory, 0o700))
	environment.Config.RecoveryDumpPath = filepath.Join(recoveryDirectory, "backup.dbbackup")
	environment.Config.RecoveryMasterKeyPath = filepath.Join(recoveryDirectory, "master.key")
	require.NoError(t, os.WriteFile(environment.Config.RecoveryDumpPath, archive, 0o600))
	require.NoError(t, os.WriteFile(environment.Config.RecoveryMasterKeyPath,
		[]byte("a source master key of at least thirty-two bytes"), 0o600))
}

func readAppliedMigrations(t *testing.T, databaseDsn string) []int64 {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)
	defer closeDatabase(db)

	versions, err := database.ReadAppliedMigrationVersions(t.Context(), db)
	require.NoError(t, err)

	return versions
}
