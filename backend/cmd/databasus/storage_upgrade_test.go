package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestStorageUpgradeRefusal(t *testing.T) {
	t.Parallel()

	for _, command := range []string{"--list-admins", "--disable-2fa"} {
		t.Run("exits before migration when console initializes with "+command, func(t *testing.T) {
			t.Parallel()
			databaseDsn := testdb.OpenEmpty(t)
			executeStatement(t, databaseDsn, "CREATE TABLE storages (type TEXT)")
			executeStatement(t, databaseDsn, "INSERT INTO storages VALUES ('GOOGLE_DRIVE')")

			consoleRun := runBinary(t, newOperatorHost(t), newDatabaseEnvironment(databaseDsn), command)

			assert.NotEqual(t, 0, consoleRun.exitCode)
			assert.Contains(t, consoleRun.output, "Google Drive support ended after Databasus 3.67.0. "+
				"Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, "+
				"or continue using 3.67.0.")
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
			assert.Equal(t, 1, storageCount)
		})
	}
}
