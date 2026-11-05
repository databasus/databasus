// Package testdb gives a test a database of its own on the shared test PostgreSQL server. A migrated
// database is copied from a template that is built once per set of migrations, so a test never
// replays the migrations and worktrees with different migrations never share a schema.
package testdb

import (
	"context"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/database"
	"databasus-backend/migrations"
)

const (
	testDatabaseDsnVariable = "TEST_DATABASE_DSN"
	systemDatabaseName      = "postgres"
	testDatabasePrefix      = "databasus_test_"
	templateDatabasePrefix  = "databasus_test_template_"
)

// Open creates a database migrated to the checkout's migrations and returns its connection string.
// A passing test drops it; a failing test keeps it and logs its address.
func Open(tb testing.TB) string {
	tb.Helper()

	return openCopyOfMigrations(tb, migrations.Files)
}

// OpenEmpty creates a database without migrations, for tests of the migrations themselves, and
// returns its connection string. A passing test drops it; a failing test keeps it and logs its address.
func OpenEmpty(tb testing.TB) string {
	tb.Helper()

	testDsn := readTestDsn(tb)
	databaseName := newTestDatabaseName(tb)

	withSystemDatabase(tb, testDsn, func(systemDb *gorm.DB) {
		require.NoError(tb, removeAbandonedDatabasesOncePerProcess(tb.Context(), systemDb))
		// The database name is generated from a fixed prefix, timestamp and hexadecimal random suffix.
		// nosemgrep: go.lang.security.audit.database.string-formatted-query.string-formatted-query
		require.NoError(tb, systemDb.WithContext(tb.Context()).Exec("CREATE DATABASE "+databaseName).Error)
	})

	return keepOnFailureOrDrop(tb, testDsn, databaseName)
}

func openCopyOfMigrations(tb testing.TB, files fs.FS) string {
	tb.Helper()

	testDsn := readTestDsn(tb)
	databaseName := newTestDatabaseName(tb)

	withSystemDatabase(tb, testDsn, func(systemDb *gorm.DB) {
		require.NoError(tb, createCopyOfMigrations(tb.Context(), systemDb, testDsn, files, databaseName))
	})

	return keepOnFailureOrDrop(tb, testDsn, databaseName)
}

func createCopyOfMigrations(
	ctx context.Context,
	systemDb *gorm.DB,
	testDsn string,
	files fs.FS,
	databaseName string,
) error {
	err := removeAbandonedDatabasesOncePerProcess(ctx, systemDb)
	if err != nil {
		return err
	}

	templateName, err := ensureTemplateOncePerProcess(ctx, systemDb, testDsn, files)
	if err != nil {
		return err
	}

	// Both identifiers are generated internally from fixed prefixes and hexadecimal hashes or suffixes.
	// nosemgrep: go.lang.security.audit.database.string-formatted-query.string-formatted-query
	err = systemDb.WithContext(ctx).Exec("CREATE DATABASE " + databaseName + " TEMPLATE " + templateName).Error
	if err != nil {
		return fmt.Errorf("copy the template %s: %w", templateName, err)
	}

	return nil
}

func keepOnFailureOrDrop(tb testing.TB, testDsn, databaseName string) string {
	tb.Helper()

	databaseDsn := replaceDatabaseName(testDsn, databaseName)

	tb.Cleanup(func() {
		if tb.Failed() {
			tb.Logf("kept the test database for inspection: %s", removePassword(databaseDsn))

			return
		}

		// The test context is already cancelled when cleanups run.
		err := dropDatabase(context.WithoutCancel(tb.Context()), testDsn, databaseName)
		if err != nil {
			tb.Logf("drop the test database %s: %v", databaseName, err)
		}
	})

	return databaseDsn
}

// Drop removes a test-owned database, including connections left by a child test process.
func Drop(tb testing.TB, databaseName string) {
	tb.Helper()

	// Drop is usually called from a cleanup, when the test context is already cancelled.
	require.NoError(tb, dropDatabase(context.WithoutCancel(tb.Context()), readTestDsn(tb), databaseName))
}

func dropDatabase(ctx context.Context, testDsn, databaseName string) error {
	systemDb, err := database.Open(ctx, replaceDatabaseName(testDsn, systemDatabaseName))
	if err != nil {
		return fmt.Errorf("connect to the test server: %w", err)
	}
	defer closeDatabase(systemDb)

	// Only test-owned internally generated database names reach this cleanup.
	// nosemgrep: go.lang.security.audit.database.string-formatted-query.string-formatted-query
	return systemDb.WithContext(ctx).Exec("DROP DATABASE IF EXISTS " + databaseName + " WITH (FORCE)").Error
}

func withSystemDatabase(tb testing.TB, testDsn string, work func(systemDb *gorm.DB)) {
	tb.Helper()

	systemDb, err := database.Open(tb.Context(), replaceDatabaseName(testDsn, systemDatabaseName))
	require.NoError(tb, err, "connect to the test server")
	defer closeDatabase(systemDb)

	work(systemDb)
}

func readTestDsn(tb testing.TB) string {
	tb.Helper()

	testDsn, err := readTestDsnVariable()
	require.NoError(tb, err)

	return testDsn
}

func readTestDsnVariable() (string, error) {
	variables, _, err := config.ReadVariables(config.LoadSpec{
		IsDatabaseRequired:       false,
		RootDirectory:            "",
		PublishedDatabaseDsnPath: "",
	})
	if err != nil {
		return "", fmt.Errorf("read the test settings: %w", err)
	}

	testDsn := variables[testDatabaseDsnVariable]
	if testDsn == "" {
		return "", fmt.Errorf("%s is not set; copy .env.example to .env", testDatabaseDsnVariable)
	}

	return testDsn, nil
}

func closeDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}
