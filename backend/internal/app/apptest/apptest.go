// Package apptest builds the real application for backend API tests: the routes and middleware of
// the server, an internal database of the test's own and no background loop.
package apptest

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"databasus-backend/internal/app"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/testdb"
)

type App struct {
	// URL is the address of the server without /api/v1, the base URL apiclient.New takes.
	URL string

	databaseDsn string
}

// The old code reaches its configuration and database through process-wide state, so one test
// application at a time can own it. A test that builds a second one while the first is alive fails
// instead of reading the other test's data.
var aliveApp sync.Mutex

// Lazy singletons of the old code may keep a data path from the first configuration they see, so
// every test application of the process shares one data directory, which no test cleanup deletes.
var getProcessRootDirectory = sync.OnceValues(func() (string, error) {
	rootDirectory, err := os.MkdirTemp("", "databasus-apptest-")
	if err != nil {
		return "", fmt.Errorf("create the data directory of the test applications: %w", err)
	}

	err = os.WriteFile(filepath.Join(rootDirectory, ".env"), []byte("ENV_MODE=development\n"), 0o600)
	if err != nil {
		return "", fmt.Errorf("write the .env of the test applications: %w", err)
	}

	return rootDirectory, nil
})

// New builds the application through app.New with a database from testdb.Open and serves it until
// the test ends. The API tests of one package run one at a time while the old code is wired through
// process-wide state, so they never call t.Parallel.
func New(tb testing.TB) *App {
	tb.Helper()

	if !aliveApp.TryLock() {
		tb.Fatalf("another test application is alive in this process: the API tests of one package " +
			"run one at a time, so a test that calls apptest.New must not call t.Parallel")
	}

	tb.Cleanup(aliveApp.Unlock)

	databaseDsn := testdb.Open(tb)

	environment, err := app.NewEnvironment(tb.Context(), newConfig(tb, databaseDsn))
	if err != nil {
		tb.Fatalf("build the environment of the test application: %v", err)
	}

	createdApp, err := app.New(tb.Context(), environment)
	if err != nil {
		tb.Fatalf("build the test application: %v", err)
	}

	tb.Cleanup(createdApp.Close)

	handler, err := createdApp.NewHandler()
	if err != nil {
		tb.Fatalf("build the handler of the test application: %v", err)
	}

	server := httptest.NewServer(handler)
	tb.Cleanup(server.Close)

	return &App{URL: server.URL, databaseDsn: databaseDsn}
}

// newConfig sets the database after loading, so a DATABASE_DSN in the environment of the test run
// can never point a test at another database.
func newConfig(tb testing.TB, databaseDsn string) config.Config {
	tb.Helper()

	rootDirectory, err := getProcessRootDirectory()
	if err != nil {
		tb.Fatalf("%v", err)
	}

	loadedConfig, _, err := config.Load(config.LoadSpec{
		IsDatabaseRequired:       false,
		RootDirectory:            rootDirectory,
		PublishedDatabaseDsnPath: filepath.Join(rootDirectory, "never-published"),
	})
	if err != nil {
		tb.Fatalf("load the configuration of the test application: %v", err)
	}

	loadedConfig.DatabaseDsn = databaseDsn

	return loadedConfig
}
