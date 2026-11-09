package migrations_test

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/migrations"
)

const (
	googleDriveRemovalMigration = "20261109083400_remove_google_drive_storage.sql"
	googleDriveRecoveryGuidance = "Google Drive support ended after Databasus 3.67.0. " +
		"Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, " +
		"or continue using 3.67.0."
)

func TestGoogleDriveRemovalMigration(t *testing.T) {
	t.Parallel()

	for _, hasProviderSettings := range []bool{true, false} {
		name := "refuses unchanged records and history with missing settings"
		if hasProviderSettings {
			name = "refuses unchanged encrypted credentials and history with saved settings"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, googleDriveRemovalMigration)
			workspaceID := insertWorkspace(t, db)
			storageID := insertOldStorage(t, db, "GOOGLE_DRIVE", &workspaceID)

			if hasProviderSettings {
				execute(t, db, `INSERT INTO storages_google_drive (storage_id, client_id, client_secret, token_json)
					VALUES (?, 'synthetic-client', 'enc:synthetic-secret', 'enc:synthetic-token')`, storageID)
			}

			storageSnapshot := getRetirementSnapshot(t, db)

			_, err := database.Migrate(t.Context(), db, migrations.Files)

			require.ErrorContains(t, err, googleDriveRecoveryGuidance)
			assert.NotContains(t, err.Error(), "synthetic")
			hasUnchangedSnapshot := storageSnapshot == getRetirementSnapshot(t, db)
			assert.True(t, hasUnchangedSnapshot, "refusal must preserve records and history")
			assert.True(t, db.Migrator().HasTable("storages_google_drive"))

			execute(t, db, "DELETE FROM storages WHERE id = ?", storageID)
			applyAllMigrations(t, db, googleDriveRemovalMigration)
			assert.False(t, db.Migrator().HasTable("storages_google_drive"))
		})
	}

	t.Run("removes obsolete settings after an older dump preserves supported secrets", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, storagesMigrationName)
		workspaceID := insertWorkspace(t, db)
		secretColumns := insertOldStorageOfEveryTypeWithSecrets(t, db, workspaceID)

		applyAllMigrations(t, db, googleDriveRemovalMigration)

		assert.False(t, db.Migrator().HasTable("storages_google_drive"))
		assert.False(t, db.Migrator().HasTable("google_drive_storages"))

		for _, secretColumn := range secretColumns {
			var storedSecret string
			require.NoError(t, db.WithContext(t.Context()).Raw(
				"SELECT "+secretColumn.column+" FROM "+secretColumn.table+" WHERE storage_id = ?", secretColumn.storageID,
			).
				Scan(&storedSecret).Error)
			hasUnchangedSecret := storedSecret == secretColumn.value
			assert.True(t, hasUnchangedSecret, "%s.%s", secretColumn.table, secretColumn.column)
		}
	})

	t.Run(
		"restores only an empty constrained settings table when rolled down after fresh installation",
		func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, googleDriveRemovalMigration)
			applyAllMigrations(t, db, googleDriveRemovalMigration)
			require.False(t, db.Migrator().HasTable("storages_google_drive"))
			sqlDB, err := db.DB()
			require.NoError(t, err)
			provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.Files,
				goose.WithDisableGlobalRegistry(true))
			require.NoError(t, err)

			_, err = provider.DownTo(t.Context(), 20261109083399)
			require.NoError(t, err)
			assert.True(t, db.Migrator().HasTable("storages_google_drive"))
			assert.Empty(t, listIDs(t, db, "SELECT storage_id FROM storages_google_drive"))

			_, err = database.Migrate(t.Context(), db, migrations.Files)
			require.NoError(t, err)
			assert.False(t, db.Migrator().HasTable("storages_google_drive"))
		},
	)
}

func getRetirementSnapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()

	var snapshot strings.Builder
	for _, tableName := range []string{"storages", "storages_google_drive", "goose_db_version"} {
		var tableSnapshot string
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT COALESCE(jsonb_agg(to_jsonb(records) ORDER BY to_jsonb(records)::text), '[]')::text FROM "+tableName+" records",
		).
			Scan(&tableSnapshot).Error)
		snapshot.WriteString(tableSnapshot)
	}

	return snapshot.String()
}
