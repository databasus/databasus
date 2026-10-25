package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	old_config "databasus-backend/internal/config"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/migrations"
)

// The tests of this package fill the process-wide bridge of the old code, so none of them runs in
// parallel.

func TestNew(t *testing.T) {
	t.Run("feeds every public getter of the old code after New", func(t *testing.T) {
		newTestApp(t)

		for _, getter := range getOldCodeGetters() {
			t.Run(getter.name, func(t *testing.T) {
				requireGetterReturnsValue(t, getter)
			})
		}
	})

	t.Run("returns an error naming the migration and fills no bridge when a migration fails", func(t *testing.T) {
		databaseDsn := testdb.OpenEmpty(t)
		db, err := database.Open(t.Context(), databaseDsn)
		require.NoError(t, err)
		require.NoError(t, db.WithContext(t.Context()).Exec("CREATE TABLE users (id INT)").Error)
		closeDatabase(db)

		environment := newTestEnvironment(t, databaseDsn)

		createdApp, err := New(t.Context(), environment)

		require.Error(t, err)
		assert.Nil(t, createdApp)
		assert.Contains(t, err.Error(), "20250605090323_init.sql")
		assert.NotEqual(t, databaseDsn, old_config.GetEnv().DatabaseDsn)
	})
	t.Run("returns an error naming both versions and fills no bridge when the schema is newer", func(t *testing.T) {
		databaseDsn := testdb.Open(t)
		recordNewerMigration(t, databaseDsn)

		environment := newTestEnvironment(t, databaseDsn)

		createdApp, err := New(t.Context(), environment)

		require.ErrorIs(t, err, database.ErrSchemaNewerThanBinary)
		assert.Nil(t, createdApp)
		assert.Contains(t, err.Error(), strconv.Itoa(newerMigrationVersion))
		assert.Contains(t, err.Error(), getNewestMigrationVersion(t))
		assert.Contains(t, err.Error(), "after an upgrade is not supported")
		assert.NotEqual(t, databaseDsn, old_config.GetEnv().DatabaseDsn)
	})

	t.Run(
		"returns an error naming the key file and keeps it when the key is shorter than 32 bytes",
		func(t *testing.T) {
			environment := newTestEnvironment(t, testdb.Open(t))
			writeKeyFile(t, environment.Config.SecretKeyPath, "short-key")

			createdApp, err := New(t.Context(), environment)

			require.ErrorIs(t, err, crypto.ErrMasterKeyTooShort)
			assert.Nil(t, createdApp)
			assert.Contains(t, err.Error(), environment.Config.SecretKeyPath)
			assertKeyFileContent(t, environment.Config.SecretKeyPath, "short-key")
		},
	)

	t.Run("returns an error naming the key file when its directory refuses a new file", func(t *testing.T) {
		skipUnderRoot(t)
		environment := newTestEnvironmentWithoutLogFile(t, testdb.Open(t))
		makeDataDirectoryReadOnly(t, environment)

		createdApp, err := New(t.Context(), environment)

		require.Error(t, err)
		assert.Nil(t, createdApp)
		assert.Contains(t, err.Error(), environment.Config.SecretKeyPath)
		assert.NoFileExists(t, environment.Config.SecretKeyPath)
	})

	t.Run("returns an error naming the key file and keeps the legacy key when the key file cannot be written",
		func(t *testing.T) {
			skipUnderRoot(t)
			databaseDsn := testdb.Open(t)
			insertLegacyMasterKey(t, databaseDsn, "lib_crypto_secret_keys", legacyMasterKey)
			environment := newTestEnvironmentWithoutLogFile(t, databaseDsn)
			makeDataDirectoryReadOnly(t, environment)

			createdApp, err := New(t.Context(), environment)

			require.Error(t, err)
			assert.Nil(t, createdApp)
			assert.Contains(t, err.Error(), environment.Config.SecretKeyPath)
			assert.Equal(t, []string{legacyMasterKey}, listLegacyMasterKeys(t, databaseDsn))
		})

	t.Run("decrypts a value encrypted with the legacy key after an upgrade from a release without a key file",
		func(t *testing.T) {
			databaseDsn := testdb.OpenEmpty(t)
			migrateBeforeLegacyKeyTableRename(t, databaseDsn)
			insertLegacyMasterKey(t, databaseDsn, "secret_keys", legacyMasterKey)
			legacyKeyFile := filepath.Join(t.TempDir(), "legacy.key")
			writeKeyFile(t, legacyKeyFile, legacyMasterKey)
			legacyKey, err := crypto.OpenOrCreateMasterKey(legacyKeyFile)
			require.NoError(t, err)
			encryptedValue, err := crypto.NewFieldEncryptor(legacyKey).Encrypt("database-password")
			require.NoError(t, err)
			environment := newTestEnvironment(t, databaseDsn)

			createdApp, err := New(t.Context(), environment)

			require.NoError(t, err)
			t.Cleanup(createdApp.Close)
			assertKeyFileContent(t, environment.Config.SecretKeyPath, legacyMasterKey)
			assert.Empty(t, listLegacyMasterKeys(t, databaseDsn))
			decryptedValue, err := encryption.GetFieldEncryptor().Decrypt(encryptedValue)
			require.NoError(t, err)
			assert.Equal(t, "database-password", decryptedValue)
		})
}

func TestLoadEnvironment(t *testing.T) {
	t.Run("returns an error naming the variable when a required one is empty", func(t *testing.T) {
		t.Setenv("ENV_MODE", "")
		t.Setenv("DATABASE_DSN", "host=localhost dbname=databasus")

		_, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, true))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ENV_MODE")
	})
}

func newTestApp(t *testing.T) *App {
	t.Helper()

	createdApp, err := New(t.Context(), newTestEnvironment(t, testdb.Open(t)))
	require.NoError(t, err)

	return createdApp
}

func newTestEnvironment(t *testing.T, databaseDsn string) *Environment {
	t.Helper()

	t.Setenv("ENV_MODE", "development")
	t.Setenv("DATABASE_DSN", databaseDsn)

	environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, true))
	require.NoError(t, err)

	return environment
}

// newIsolatedLoadSpec keeps the data directory and .env of the test apart from the checkout's.
func newIsolatedLoadSpec(t *testing.T, isDatabaseRequired bool) config.LoadSpec {
	t.Helper()

	rootDirectory := t.TempDir()

	return config.LoadSpec{
		IsDatabaseRequired:       isDatabaseRequired,
		RootDirectory:            rootDirectory,
		PublishedDatabaseDsnPath: filepath.Join(rootDirectory, "never-published"),
	}
}

const newerMigrationVersion = 99991231235959

func recordNewerMigration(t *testing.T, databaseDsn string) {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)
	defer closeDatabase(db)

	require.NoError(t, db.WithContext(t.Context()).Exec(
		"INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, TRUE)", newerMigrationVersion,
	).Error)
}

func getNewestMigrationVersion(t *testing.T) string {
	t.Helper()

	migrationFiles, err := migrations.Files.ReadDir(".")
	require.NoError(t, err)

	newestName := migrationFiles[len(migrationFiles)-1].Name()

	newestVersion, _, _ := strings.Cut(newestName, "_")

	return newestVersion
}

const (
	legacyMasterKey          = "fedcba9876543210fedcba9876543210-legacy"
	legacyKeyTableRenameName = "20261025082512_rename_secret_keys_to_lib_crypto_secret_keys.sql"
)

func writeKeyFile(t *testing.T, path, secret string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(secret), 0o600))
}

func assertKeyFileContent(t *testing.T, path, expectedSecret string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expectedSecret, string(content))
}

// newTestEnvironmentWithoutLogFile keeps the log file out of a data directory that the test makes
// read-only, so the logger does not fail its writes and the directory can be removed afterwards.
func newTestEnvironmentWithoutLogFile(t *testing.T, databaseDsn string) *Environment {
	t.Helper()

	t.Setenv("LOG_FILE_IS_ENABLED", "false")

	return newTestEnvironment(t, databaseDsn)
}

// makeDataDirectoryReadOnly creates the folders New ensures first, so only the key file is refused.
func makeDataDirectoryReadOnly(t *testing.T, environment *Environment) {
	t.Helper()

	for _, directory := range []string{environment.Config.TempFolder, environment.Config.DataFolder} {
		require.NoError(t, os.MkdirAll(directory, 0o755))
	}

	dataDirectory := filepath.Dir(environment.Config.SecretKeyPath)
	require.NoError(t, os.Chmod(dataDirectory, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dataDirectory, 0o755) })
}

func skipUnderRoot(t *testing.T) {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
}

// migrateBeforeLegacyKeyTableRename leaves the schema of the release before the table of the legacy key
// was renamed.
func migrateBeforeLegacyKeyTableRename(t *testing.T, databaseDsn string) {
	t.Helper()

	migrationEntries, err := migrations.Files.ReadDir(".")
	require.NoError(t, err)

	olderMigrations := fstest.MapFS{}
	for _, migrationEntry := range migrationEntries {
		if migrationEntry.Name() >= legacyKeyTableRenameName {
			continue
		}

		content, readErr := fs.ReadFile(migrations.Files, migrationEntry.Name())
		require.NoError(t, readErr)
		olderMigrations[migrationEntry.Name()] = &fstest.MapFile{Data: content}
	}

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)
	defer closeDatabase(db)

	_, err = database.Migrate(t.Context(), db, olderMigrations)
	require.NoError(t, err)
}

func insertLegacyMasterKey(t *testing.T, databaseDsn, tableName, secret string) {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)
	defer closeDatabase(db)

	require.NoError(t, db.WithContext(t.Context()).
		Exec("INSERT INTO "+tableName+" (secret) VALUES (?)", secret).Error)
}

func listLegacyMasterKeys(t *testing.T, databaseDsn string) []string {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)
	defer closeDatabase(db)

	var secrets []string
	require.NoError(t, db.WithContext(t.Context()).
		Raw("SELECT secret FROM lib_crypto_secret_keys").Scan(&secrets).Error)

	return secrets
}
