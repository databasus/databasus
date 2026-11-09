package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

const storageRetirementGuidance = "Google Drive support ended after Databasus 3.67.0. " +
	"Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, " +
	"or continue using 3.67.0."

func TestStorageUpgradeInitialization(t *testing.T) {
	for _, providerTable := range []string{"google_drive_storages", "storages_google_drive", ""} {
		t.Run(
			"refuses shared server and console initialization with provider table "+providerTable,
			func(t *testing.T) {
				databaseDsn := testdb.OpenEmpty(t)
				db, err := database.Open(t.Context(), databaseDsn)
				require.NoError(t, err)
				t.Cleanup(func() { closeDatabase(db) })
				require.NoError(t, db.WithContext(t.Context()).Exec(
					"CREATE TABLE storages (id INT, type TEXT, workspace_id INT)",
				).Error)
				require.NoError(t, db.WithContext(t.Context()).Exec(
					"INSERT INTO storages VALUES (1, 'LOCAL', 1), (2, 'GOOGLE_DRIVE', 2)",
				).Error)
				require.NoError(t, db.WithContext(t.Context()).Exec(
					"CREATE TABLE goose_db_version (version_id BIGINT, is_applied BOOLEAN)",
				).Error)
				require.NoError(t, db.WithContext(t.Context()).Exec(
					"INSERT INTO goose_db_version VALUES (20250605090323, TRUE)",
				).Error)

				if providerTable != "" {
					require.NoError(t, db.WithContext(t.Context()).Exec(
						"CREATE TABLE "+providerTable+" (storage_id INT, client_secret TEXT, token_json TEXT)",
					).Error)
					require.NoError(t, db.WithContext(t.Context()).Exec(
						"INSERT INTO "+providerTable+" VALUES (2, 'enc:synthetic-secret', 'enc:synthetic-token')",
					).Error)
				}

				environment := newTestEnvironment(t, databaseDsn)

				createdApp, err := New(t.Context(), environment)

				require.EqualError(t, err, storageRetirementGuidance)
				assert.Nil(t, createdApp)
				assert.False(t, db.Migrator().HasTable("users"))
				assert.NoFileExists(t, environment.Config.SecretKeyPath)

				var hasUnchangedHistory bool
				require.NoError(t, db.WithContext(t.Context()).Raw(
					"SELECT COUNT(*) = 1 AND bool_and(version_id = 20250605090323 AND is_applied) FROM goose_db_version",
				).
					Scan(&hasUnchangedHistory).Error)
				assert.True(t, hasUnchangedHistory)

				var storageCount int
				require.NoError(
					t,
					db.WithContext(t.Context()).Raw("SELECT COUNT(*) FROM storages").Scan(&storageCount).Error,
				)
				assert.Equal(t, 2, storageCount)

				if providerTable != "" {
					var hasUnchangedSecrets bool
					require.NoError(t, db.WithContext(t.Context()).Raw(
						"SELECT client_secret = 'enc:synthetic-secret' AND token_json = 'enc:synthetic-token' FROM "+providerTable,
					).
						Scan(&hasUnchangedSecrets).Error)
					assert.True(t, hasUnchangedSecrets)
				}
			},
		)
	}

	t.Run("refuses initialization when storage inspection fails", func(t *testing.T) {
		databaseDsn := testdb.OpenEmpty(t)
		db, err := database.Open(t.Context(), databaseDsn)
		require.NoError(t, err)
		t.Cleanup(func() { closeDatabase(db) })
		require.NoError(t, db.WithContext(t.Context()).Exec("CREATE TABLE storages (id INT)").Error)

		createdApp, err := New(t.Context(), newTestEnvironment(t, databaseDsn))

		require.ErrorContains(t, err, "inspect storages before upgrading")
		assert.Nil(t, createdApp)
		assert.False(t, db.Migrator().HasTable("goose_db_version"))
	})

	t.Run("starts and restarts after removal on a fresh database", func(t *testing.T) {
		environment := newTestEnvironment(t, testdb.OpenEmpty(t))

		createdApp, err := New(t.Context(), environment)
		require.NoError(t, err)
		createdApp.Close()

		restartedApp, err := New(t.Context(), environment)
		require.NoError(t, err)
		restartedApp.Close()
	})
}
