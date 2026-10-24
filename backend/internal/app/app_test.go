package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	old_config "databasus-backend/internal/config"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
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
		databaseDsn := testdb.CreateDatabase(t)
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

	createdApp, err := New(t.Context(), newTestEnvironment(t, testdb.CreateDatabase(t)))
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
