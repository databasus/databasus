//go:build testhelpers

package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/caarlos0/env/v11"

	lib_config "databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/testdb"
)

// fixturesLockFileName names the lock that guards the shared test PostgreSQL servers. Every test
// binary holds it shared for its whole life, and scripts/test.sh takes it exclusively to reset
// those servers, so the reset waits for running tests and a test that starts during the reset waits
// for it to end.
const fixturesLockFileName = "databasus-test-fixtures.lock"

// fixturesLockFile keeps the shared lock: closing the file, or letting it be garbage-collected,
// releases the lock while the process still uses the servers.
//
//nolint:unused // assigned-only: keeps the shared lock for the process lifetime
var fixturesLockFile *os.File

// loadWithoutApp loads the test configuration: the ports of the shared fixtures and, in a go test
// binary, an internal database of the process's own. cmd/cleanup_test_db is not a test binary, so it
// neither waits for the lock that guards its own reset nor creates a database.
func loadWithoutApp() *EnvVariables {
	spec := lib_config.LoadSpec{IsDatabaseRequired: false, RootDirectory: "", PublishedDatabaseDsnPath: ""}

	loadedConfig, _, err := lib_config.Load(spec)
	if err != nil {
		panic(fmt.Sprintf("load the test configuration: %v", err))
	}

	variables, _, err := lib_config.ReadVariables(spec)
	if err != nil {
		panic(fmt.Sprintf("read the test settings: %v", err))
	}

	testEnv := EnvVariables{Config: loadedConfig, IsTesting: true}

	err = env.ParseWithOptions(&testEnv, env.Options{Environment: variables})
	if err != nil {
		panic(fmt.Sprintf("parse the test settings: %v", err))
	}

	requireTestSettings(&testEnv)

	testEnv.DatabaseDsn = testEnv.TestDatabaseDsn

	if testEnv.TestLocalhost == "" {
		testEnv.TestLocalhost = "localhost"
	}

	if testing.Testing() {
		fixturesLockFile = lockSharedFixtures()

		testEnv.DatabaseDsn, err = testdb.OpenForProcess(context.Background())
		if err != nil {
			panic(fmt.Sprintf("create the internal database of this test process: %v", err))
		}
	}

	return &testEnv
}

func requireTestSettings(testEnv *EnvVariables) {
	requiredSettings := map[string]string{
		"TEST_DATABASE_DSN":              testEnv.TestDatabaseDsn,
		"TEST_LOGICAL_POSTGRES_16_PORT":  testEnv.TestLogicalPostgres16Port,
		"TEST_PHYSICAL_POSTGRES_17_PORT": testEnv.TestPhysicalPostgres17Port,
		"TEST_PHYSICAL_POSTGRES_18_PORT": testEnv.TestPhysicalPostgres18Port,
	}

	for name, value := range requiredSettings {
		if value == "" {
			panic(name + " is empty")
		}
	}
}

// lockSharedFixtures blocks while a full run resets the shared servers. The kernel releases the
// lock when the process dies, however it dies.
func lockSharedFixtures() *os.File {
	lockPath := filepath.Join(os.TempDir(), fixturesLockFileName)

	lockFile, err := os.OpenFile(lockPath, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		panic(fmt.Sprintf("open the test fixtures lock %s: %v", lockPath, err))
	}

	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_SH)
	if err != nil {
		panic(fmt.Sprintf("take the test fixtures lock %s: %v", lockPath, err))
	}

	return lockFile
}
