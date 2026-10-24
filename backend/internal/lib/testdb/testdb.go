// Package testdb gives a test an empty database of its own on the shared test PostgreSQL server.
package testdb

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/database"
)

// CreateDatabase creates an empty database on the server TEST_DATABASE_DSN points to, drops it
// after the test and returns its connection string.
func CreateDatabase(t *testing.T) string {
	t.Helper()

	variables, _, err := config.ReadVariables(config.LoadSpec{
		IsDatabaseRequired:       false,
		RootDirectory:            "",
		PublishedDatabaseDsnPath: "",
	})
	require.NoError(t, err)

	testDsn := variables["TEST_DATABASE_DSN"]
	require.NotEmpty(t, testDsn, "TEST_DATABASE_DSN is not set")

	systemDb, err := database.Open(t.Context(), ReplaceDatabaseName(testDsn, "postgres"))
	require.NoError(t, err)

	randomBytes := make([]byte, 6)
	_, err = rand.Read(randomBytes)
	require.NoError(t, err)

	databaseName := "databasus_test_" + hex.EncodeToString(randomBytes)
	require.NoError(t, systemDb.WithContext(t.Context()).Exec("CREATE DATABASE "+databaseName).Error)

	t.Cleanup(func() {
		_ = systemDb.Exec("DROP DATABASE IF EXISTS " + databaseName + " WITH (FORCE)").Error

		systemSQLDB, sqlErr := systemDb.DB()
		if sqlErr == nil {
			_ = systemSQLDB.Close()
		}
	})

	return ReplaceDatabaseName(testDsn, databaseName)
}

// ReplaceDatabaseName swaps the dbname of a keyword/value connection string.
func ReplaceDatabaseName(dsn, databaseName string) string {
	dsnParts := strings.Fields(dsn)
	for partIndex, dsnPart := range dsnParts {
		if strings.HasPrefix(dsnPart, "dbname=") {
			dsnParts[partIndex] = "dbname=" + databaseName
		}
	}

	return strings.Join(dsnParts, " ")
}
