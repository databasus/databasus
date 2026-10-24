package testdb

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	t.Run("gives each of two tests a database where an account with the same email is created", func(t *testing.T) {
		t.Parallel()

		for _, testName := range []string{"first test", "second test"} {
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				db := openDatabase(t, Open(t))

				err := db.WithContext(t.Context()).Exec(
					"INSERT INTO users (email, password_creation_time, role, name, status) VALUES (?, now(), 'MEMBER', 'Same', 'ACTIVE')",
					"same@example.com",
				).
					Error

				require.NoError(t, err)
				assert.Equal(t, int64(1), countRows(t, db, "users"))
			})
		}
	})

	t.Run("applies a migration added to the checkout without a preparation step", func(t *testing.T) {
		t.Parallel()
		initialMigrations := newMigrations(t, 1)
		changedMigrations := newMigrations(t, 2)

		initialDb := openDatabase(t, openCopyOfMigrations(t, initialMigrations))
		changedDb := openDatabase(t, openCopyOfMigrations(t, changedMigrations))

		assert.True(t, hasTable(t, initialDb, "first_table"))
		assert.False(t, hasTable(t, initialDb, "second_table"))
		assert.True(t, hasTable(t, changedDb, "second_table"))
	})

	t.Run("builds a template per migration set when two sets are opened at once", func(t *testing.T) {
		t.Parallel()
		withoutExtraMigration := newMigrations(t, 1)
		withExtraMigration := newMigrations(t, 2)

		for _, testCase := range []struct {
			name            string
			files           fstest.MapFS
			isTableExpected bool
		}{
			{name: "worktree with the extra migration", files: withExtraMigration, isTableExpected: true},
			{name: "worktree without the extra migration", files: withoutExtraMigration, isTableExpected: false},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()
				db := openDatabase(t, openCopyOfMigrations(t, testCase.files))

				assert.Equal(t, testCase.isTableExpected, hasTable(t, db, "second_table"))
			})
		}
	})

	t.Run("drops the database when the test passes", func(t *testing.T) {
		t.Parallel()
		passingTest := &recordingTB{TB: t}

		databaseName := GetDatabaseName(Open(passingTest))
		passingTest.runCleanups()

		assert.False(t, isDatabasePresent(t, databaseName))
		assert.Empty(t, passingTest.logs)
	})

	t.Run("keeps the database and logs its address without the password when the test fails", func(t *testing.T) {
		t.Parallel()
		failingTest := &recordingTB{TB: t, isFailed: true}

		databaseDsn := Open(failingTest)
		databaseName := GetDatabaseName(databaseDsn)
		t.Cleanup(func() { Drop(t, databaseName) })
		failingTest.runCleanups()

		assert.True(t, isDatabasePresent(t, databaseName))
		require.Len(t, failingTest.logs, 1)
		loggedAddress := failingTest.logs[0]
		for _, addressPart := range []string{"host=", "port=", "user=", "dbname=" + databaseName} {
			assert.Contains(t, loggedAddress, addressPart)
		}
		assert.NotContains(t, loggedAddress, "password")
		assert.Contains(t, databaseDsn, "password=")
	})
}

func TestOpenEmpty(t *testing.T) {
	t.Parallel()

	t.Run("creates a database without tables", func(t *testing.T) {
		t.Parallel()
		db := openDatabase(t, OpenEmpty(t))

		var tableCount int64
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'",
		).Scan(&tableCount).Error)

		assert.Zero(t, tableCount)
	})
}

func TestOpenForProcess(t *testing.T) {
	t.Parallel()

	t.Run("gives each call its own migrated database", func(t *testing.T) {
		t.Parallel()

		firstDsn, err := OpenForProcess(t.Context())
		require.NoError(t, err)
		secondDsn, err := OpenForProcess(t.Context())
		require.NoError(t, err)
		t.Cleanup(func() {
			Drop(t, GetDatabaseName(firstDsn))
			Drop(t, GetDatabaseName(secondDsn))
		})

		assert.NotEqual(t, GetDatabaseName(firstDsn), GetDatabaseName(secondDsn))
		assert.True(t, hasTable(t, openDatabase(t, firstDsn), "users"))
	})
}

// The cleanup tests name their databases with the current time and run the cleanup with a clock two
// hours ahead, limited to their own databases. Other processes see the databases as fresh and leave
// them alone, so only the cleanup under test can drop them.
const cleanupClockShift = 2 * time.Hour

func TestRemoveAbandonedDatabases(t *testing.T) {
	t.Parallel()

	t.Run("drops only old test databases without connections", func(t *testing.T) {
		t.Parallel()
		abandonedName := createNamedDatabase(t, newTestDatabaseNameAt(t, time.Now()))
		connectedName := createNamedDatabase(t, newTestDatabaseNameAt(t, time.Now()))
		openDatabase(t, replaceDatabaseName(readTestDsn(t), connectedName))
		freshName := createNamedDatabase(t, newTestDatabaseNameAt(t, time.Now().Add(cleanupClockShift)))
		templateName := createNamedDatabase(t, templateDatabasePrefix+randomHex(t))
		foreignName := createNamedDatabase(t, "databasus_cleanup_probe_"+randomHex(t))
		scope := cleanupScope{
			now:           time.Now().Add(cleanupClockShift),
			databaseNames: []string{abandonedName, connectedName, freshName, templateName, foreignName},
		}

		require.NoError(t, removeAbandonedDatabases(t.Context(), openSystemDatabase(t), scope))

		assert.False(t, isDatabasePresent(t, abandonedName), "old database without connections")
		assert.True(t, isDatabasePresent(t, connectedName), "old database with a connection")
		assert.True(t, isDatabasePresent(t, freshName), "fresh database")
		assert.True(t, isDatabasePresent(t, templateName), "template")
		assert.True(t, isDatabasePresent(t, foreignName), "database outside the name pattern")
	})
}

var abandonedDatabaseForChild = flag.String(
	"abandoned-database",
	"",
	"the database TestOpenAsChildProcess expects its first Open to remove, for TestOpenInNewProcess",
)

func TestOpenInNewProcess(t *testing.T) {
	t.Parallel()

	t.Run("removes an abandoned database when a new test process asks for a database", func(t *testing.T) {
		t.Parallel()
		abandonedName := createNamedDatabase(t, newTestDatabaseNameAt(t, time.Now()))
		childCommand := exec.CommandContext(
			t.Context(),
			os.Args[0],
			"-test.run=^TestOpenAsChildProcess$",
			"-test.count=1",
			"-test.v",
			"-abandoned-database="+abandonedName,
		)

		childOutput, err := childCommand.CombinedOutput()

		require.NoError(t, err, string(childOutput))
		assert.Contains(t, string(childOutput), "--- PASS: TestOpenAsChildProcess")
		assert.False(t, isDatabasePresent(t, abandonedName))
	})
}

// TestOpenAsChildProcess is the fresh test process TestOpenInNewProcess starts. The database the
// parent names is fresh for every other process, so only this process's cleanup can drop it.
func TestOpenAsChildProcess(t *testing.T) {
	if *abandonedDatabaseForChild == "" {
		t.Skip("runs only as a child process of TestOpenInNewProcess")
	}

	require.True(t, isDatabasePresent(t, *abandonedDatabaseForChild), "the database is gone before Open")
	processCleanupScopeForTest = &cleanupScope{
		now:           time.Now().Add(cleanupClockShift),
		databaseNames: []string{*abandonedDatabaseForChild},
	}

	Open(t)

	assert.False(t, isDatabasePresent(t, *abandonedDatabaseForChild), "Open did not remove the abandoned database")
}

func newTestDatabaseNameAt(t *testing.T, createdAt time.Time) string {
	t.Helper()

	databaseName, err := generateTestDatabaseName(createdAt)
	require.NoError(t, err)

	return databaseName
}

type recordingTB struct {
	testing.TB

	isFailed bool
	cleanups []func()
	logs     []string
}

func (r *recordingTB) Cleanup(cleanup func()) {
	r.cleanups = append(r.cleanups, cleanup)
}

func (r *recordingTB) Failed() bool {
	return r.isFailed
}

func (r *recordingTB) Logf(format string, args ...any) {
	r.logs = append(r.logs, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func (r *recordingTB) runCleanups() {
	for cleanupIndex := len(r.cleanups) - 1; cleanupIndex >= 0; cleanupIndex-- {
		r.cleanups[cleanupIndex]()
	}
}

// newMigrations builds a migration set of its own for each test, so tests do not share its template.
// The content is stable across runs, so the template is reused instead of piling up on the server.
func newMigrations(t *testing.T, migrationCount int) fstest.MapFS {
	t.Helper()

	marker := "-- " + t.Name() + "\n"
	files := fstest.MapFS{
		"00001_create_first_table.sql": newMigration(marker + "CREATE TABLE first_table (id INT);"),
	}

	if migrationCount > 1 {
		files["00002_create_second_table.sql"] = newMigration(marker + "CREATE TABLE second_table (id INT);")
	}

	return files
}

func newMigration(statement string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("-- +goose Up\n" + statement + "\n")}
}

func openDatabase(t *testing.T, databaseDsn string) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)
	t.Cleanup(func() { closeDatabase(db) })

	return db
}

func openSystemDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	return openDatabase(t, replaceDatabaseName(readTestDsn(t), systemDatabaseName))
}

func createNamedDatabase(t *testing.T, databaseName string) string {
	t.Helper()

	require.NoError(t, openSystemDatabase(t).WithContext(t.Context()).Exec("CREATE DATABASE "+databaseName).Error)
	t.Cleanup(func() { Drop(t, databaseName) })

	return databaseName
}

func isDatabasePresent(t *testing.T, databaseName string) bool {
	t.Helper()

	isPresent, err := hasDatabase(openSystemDatabase(t).WithContext(t.Context()), databaseName)
	require.NoError(t, err)

	return isPresent
}

func hasTable(t *testing.T, db *gorm.DB, tableName string) bool {
	t.Helper()

	var tableCount int64
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = ?", tableName,
	).Scan(&tableCount).Error)

	return tableCount > 0
}

func countRows(t *testing.T, db *gorm.DB, tableName string) int64 {
	t.Helper()

	var rowCount int64
	require.NoError(t, db.WithContext(t.Context()).Table(tableName).Count(&rowCount).Error)

	return rowCount
}

func randomHex(t *testing.T) string {
	t.Helper()

	randomBytes := make([]byte, 4)
	_, err := rand.Read(randomBytes)
	require.NoError(t, err)

	return hex.EncodeToString(randomBytes)
}
