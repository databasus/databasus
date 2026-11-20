package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testdb"
)

const recoveryKeyText = "recovery key that must never reach the log 0123456789"

func TestStartupRecoveryRefusal(t *testing.T) {
	t.Parallel()

	t.Run("refuses mounted-file recovery on a target with an account and leaves it unchanged", func(t *testing.T) {
		t.Parallel()
		host := newOperatorHost(t)
		writeWorkingPostgresqlClients(t, host)
		databaseDsn := testdb.Open(t)
		executeStatement(
			t,
			databaseDsn,
			`INSERT INTO users (email, password_creation_time, role, name, status, is_root_admin)
			VALUES ('root@target.local', now(), 'ADMIN', 'Root', 'ACTIVE', true)`,
		)
		accountsBefore := listAccountRows(t, databaseDsn)

		serverRun := runBinary(t, host, newRecoveryEnvironment(t, host, databaseDsn, []byte("any archive bytes")))

		assert.Equal(t, 1, serverRun.exitCode, serverRun.output)
		assert.Contains(t, serverRun.output, "startup recovery did not run")
		assert.Contains(t, serverRun.output, "EXISTING_ACCOUNTS")
		assert.Contains(t, serverRun.output, "current_installation=unchanged")
		assert.Contains(t, serverRun.output, "root-confirmed import")
		assert.NotContains(t, serverRun.output, recoveryKeyText)
		assert.NotContains(t, serverRun.output, "running database migrations")
		assert.NotContains(t, serverRun.output, "http server listening")
		assert.Equal(t, accountsBefore, listAccountRows(t, databaseDsn))
	})

	t.Run("refuses a file that is not a Databasus backup without migrating a fresh database", func(t *testing.T) {
		t.Parallel()
		host := newOperatorHost(t)
		writeWorkingPostgresqlClients(t, host)
		databaseDsn := testdb.OpenEmpty(t)

		serverRun := runBinary(t, host, newRecoveryEnvironment(t, host, databaseDsn, []byte("PGDMP plain dump")))

		assert.Equal(t, 1, serverRun.exitCode, serverRun.output)
		assert.Contains(t, serverRun.output, "INVALID_ARCHIVE")
		assert.NotContains(t, serverRun.output, recoveryKeyText)
		requireNoMigrationRan(t, databaseDsn, serverRun)
		_, err := os.Stat(filepath.Join(host.dataDirectory, "secret.key"))
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("refuses a backup path without a key path before touching the database", func(t *testing.T) {
		t.Parallel()
		host := newOperatorHost(t)
		writeWorkingPostgresqlClients(t, host)
		databaseDsn := testdb.OpenEmpty(t)
		environment := newDatabaseEnvironment(databaseDsn)
		environment["RECOVERY_DUMP_PATH"] = filepath.Join(host.dataDirectory, "recovery", "backup.dbbackup")

		serverRun := runBinary(t, host, environment)

		assert.NotEqual(t, 0, serverRun.exitCode, serverRun.output)
		assert.Contains(t, serverRun.output, "RECOVERY_DUMP_PATH and RECOVERY_MASTER_KEY_PATH must be set together")
		requireNoMigrationRan(t, databaseDsn, serverRun)
	})

	t.Run("runs a console command without a recovery while recovery files are configured", func(t *testing.T) {
		t.Parallel()
		host := newOperatorHost(t)
		databaseDsn := testdb.Open(t)

		consoleRun := runBinary(t, host, newRecoveryEnvironment(t, host, databaseDsn, []byte("any archive bytes")),
			"--list-admins")

		assert.Equal(t, 0, consoleRun.exitCode, consoleRun.output)
		assert.NotContains(t, consoleRun.output, "startup recovery")
	})
}

// newRecoveryEnvironment puts an archive and a key file into the data volume, the way an owner adds
// them before a restart, and configures both paths.
func newRecoveryEnvironment(t *testing.T, host operatorHost, databaseDsn string, archive []byte) map[string]string {
	t.Helper()

	recoveryDirectory := filepath.Join(host.dataDirectory, "recovery")
	require.NoError(t, os.MkdirAll(recoveryDirectory, 0o700))
	archivePath := filepath.Join(recoveryDirectory, "backup.dbbackup")
	keyPath := filepath.Join(recoveryDirectory, "master.key")
	require.NoError(t, os.WriteFile(archivePath, archive, 0o600))
	require.NoError(t, os.WriteFile(keyPath, []byte(recoveryKeyText), 0o600))

	environment := newDatabaseEnvironment(databaseDsn)
	environment["RECOVERY_DUMP_PATH"] = archivePath
	environment["RECOVERY_MASTER_KEY_PATH"] = keyPath

	return environment
}
