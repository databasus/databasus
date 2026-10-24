// Package testdb gives a test a database of its own on the shared test PostgreSQL server. A migrated
// database is copied from a template that is built once per set of migrations, so a test never
// replays the migrations and worktrees with different migrations never share a schema.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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
		require.NoError(tb, systemDb.WithContext(tb.Context()).Exec("CREATE DATABASE "+databaseName).Error)
	})

	return keepOnFailureOrDrop(tb, testDsn, databaseName)
}

// OpenForProcess creates a migrated database for code that reaches the internal database through
// process-wide state. It lives as long as the process; the abandoned-database cleanup of a later
// process removes it.
func OpenForProcess(ctx context.Context) (string, error) {
	testDsn, err := readTestDsnVariable()
	if err != nil {
		return "", err
	}

	databaseName, err := generateTestDatabaseName(time.Now())
	if err != nil {
		return "", err
	}

	systemDb, err := database.Open(ctx, replaceDatabaseName(testDsn, systemDatabaseName))
	if err != nil {
		return "", fmt.Errorf("connect to the test server: %w", err)
	}
	defer closeDatabase(systemDb)

	err = createCopyOfMigrations(ctx, systemDb, testDsn, migrations.Files, databaseName)
	if err != nil {
		return "", err
	}

	return replaceDatabaseName(testDsn, databaseName), nil
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

// Drop removes a test database by name, for a database a test cannot hand to Open, such as the one
// a child test process got from OpenForProcess. FORCE ends the connections still open to it.
func Drop(tb testing.TB, databaseName string) {
	tb.Helper()

	// Drop is usually called from a cleanup, when the test context is already cancelled.
	require.NoError(tb, dropDatabase(context.WithoutCancel(tb.Context()), readTestDsn(tb), databaseName))
}

// GetDatabaseName returns the dbname of a keyword/value connection string, or "" when it has none.
func GetDatabaseName(dsn string) string {
	for dsnPart := range strings.FieldsSeq(dsn) {
		if databaseName, isDatabaseName := strings.CutPrefix(dsnPart, "dbname="); isDatabaseName {
			return databaseName
		}
	}

	return ""
}

func dropDatabase(ctx context.Context, testDsn, databaseName string) error {
	systemDb, err := database.Open(ctx, replaceDatabaseName(testDsn, systemDatabaseName))
	if err != nil {
		return fmt.Errorf("connect to the test server: %w", err)
	}
	defer closeDatabase(systemDb)

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

func newTestDatabaseName(tb testing.TB) string {
	tb.Helper()

	databaseName, err := generateTestDatabaseName(time.Now())
	require.NoError(tb, err)

	return databaseName
}

// pg_database stores no creation time, so the name carries it for the abandoned-database cleanup.
func generateTestDatabaseName(createdAt time.Time) (string, error) {
	randomBytes := make([]byte, 4)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("generate a test database name: %w", err)
	}

	return testDatabasePrefix + strconv.FormatInt(createdAt.Unix(), 10) + "_" + hex.EncodeToString(randomBytes), nil
}

func replaceDatabaseName(dsn, databaseName string) string {
	dsnParts := strings.Fields(dsn)
	for partIndex, dsnPart := range dsnParts {
		if strings.HasPrefix(dsnPart, "dbname=") {
			dsnParts[partIndex] = "dbname=" + databaseName
		}
	}

	return strings.Join(dsnParts, " ")
}

func removePassword(dsn string) string {
	dsnParts := strings.Fields(dsn)
	keptParts := make([]string, 0, len(dsnParts))

	for _, dsnPart := range dsnParts {
		if !strings.HasPrefix(dsnPart, "password=") {
			keptParts = append(keptParts, dsnPart)
		}
	}

	return strings.Join(keptParts, " ")
}

func closeDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

// A process builds or finds each template once, however many tests ask for it.
var (
	templatesMutex sync.Mutex
	templates      = map[string]*templateResult{}
)

type templateResult struct {
	once         sync.Once
	templateName string
	err          error
}

func ensureTemplateOncePerProcess(ctx context.Context, systemDb *gorm.DB, testDsn string, files fs.FS) (string, error) {
	migrationsHash, err := hashMigrations(files)
	if err != nil {
		return "", err
	}

	templatesMutex.Lock()
	result, isKnown := templates[migrationsHash]
	if !isKnown {
		result = &templateResult{templateName: templateDatabasePrefix + migrationsHash}
		templates[migrationsHash] = result
	}
	templatesMutex.Unlock()

	result.once.Do(func() {
		result.err = buildTemplate(ctx, systemDb, testDsn, files, result.templateName)
	})

	if result.err != nil {
		return "", result.err
	}

	return result.templateName, nil
}
