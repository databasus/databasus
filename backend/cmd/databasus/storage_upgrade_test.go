package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestStorageUpgradeRefusal(t *testing.T) {
	t.Parallel()

	for _, providerRows := range []string{"('GOOGLE_DRIVE')", "('RCLONE')", "('GOOGLE_DRIVE'), ('RCLONE')"} {
		for _, command := range []string{"--list-admins", "--disable-2fa"} {
			t.Run(
				"exits before migration when console initializes with "+command+" and "+providerRows,
				func(t *testing.T) {
					t.Parallel()
					databaseDsn := testdb.OpenEmpty(t)
					executeStatement(t, databaseDsn, "CREATE TABLE storages (type TEXT)")
					executeStatement(t, databaseDsn, "INSERT INTO storages VALUES "+providerRows)

					consoleRun := runBinary(t, newOperatorHost(t), newDatabaseEnvironment(databaseDsn), command)

					assert.NotEqual(t, 0, consoleRun.exitCode)
					switch providerRows {
					case "('GOOGLE_DRIVE')":
						assert.Contains(t, consoleRun.output, "Google Drive support ended after Databasus 3.67.0.")
					case "('RCLONE')":
						assert.Contains(t, consoleRun.output, "Databasus 3.67.1 supports rclone storages.")
						assert.Contains(t, consoleRun.output, "or continue using 3.67.1.")
					default:
						assert.Contains(t, consoleRun.output, "remove all Google Drive and rclone storages")
						assert.Contains(t, consoleRun.output, "or continue using 3.67.0.")
					}
					assert.NotContains(t, consoleRun.output, "running database migrations")

					db, err := database.Open(t.Context(), databaseDsn)
					require.NoError(t, err)
					sqlDB, err := db.DB()
					require.NoError(t, err)
					defer func() { require.NoError(t, sqlDB.Close()) }()

					assert.False(t, db.Migrator().HasTable("goose_db_version"))

					var storageCount int
					require.NoError(
						t,
						db.WithContext(t.Context()).Raw("SELECT COUNT(*) FROM storages").Scan(&storageCount).Error,
					)
					assert.Equal(t, 1+strings.Count(providerRows, "),"), storageCount)
				},
			)
		}
	}
}
