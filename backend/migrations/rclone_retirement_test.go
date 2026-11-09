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
	rcloneRemovalMigration = "20261109134345_remove_rclone_storage.sql"
	rcloneRecoveryGuidance = "Databasus 3.67.1 supports rclone storages. " +
		"Roll back to Databasus 3.67.1 and remove all rclone storages before upgrading, " +
		"or continue using 3.67.1."
)

func TestRcloneRemovalMigration(t *testing.T) {
	t.Parallel()

	for _, hasProviderSettings := range []bool{true, false} {
		name := "refuses unchanged records and history with missing settings"
		if hasProviderSettings {
			name = "refuses unchanged encrypted credentials and history with saved settings"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, rcloneRemovalMigration)
			workspaceID := insertWorkspace(t, db)
			storageID := insertOldStorage(t, db, "RCLONE", &workspaceID)

			if hasProviderSettings {
				execute(t, db, `INSERT INTO storages_rclone (storage_id, config_content, remote_path)
 VALUES (?, 'enc:synthetic-config', '/backups')`, storageID)
			}

			storageSnapshot := getRcloneRetirementSnapshot(t, db)

			_, err := database.Migrate(t.Context(), db, migrations.Files)

			require.ErrorContains(t, err, rcloneRecoveryGuidance)
			assert.NotContains(t, err.Error(), "synthetic")
			hasUnchangedSnapshot := storageSnapshot == getRcloneRetirementSnapshot(t, db)
			assert.True(t, hasUnchangedSnapshot, "refusal must preserve records and history")
			assert.True(t, db.Migrator().HasTable("storages_rclone"))

			execute(t, db, "DELETE FROM storages WHERE id = ?", storageID)
			applyAllMigrations(t, db, rcloneRemovalMigration)
			assert.False(t, db.Migrator().HasTable("storages_rclone"))
		})
	}

	t.Run("removes obsolete settings after an older dump preserves supported secrets", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, storagesMigrationName)
		workspaceID := insertWorkspace(t, db)
		secretColumns := insertOldStorageOfEveryTypeWithSecrets(t, db, workspaceID)

		applyAllMigrations(t, db, rcloneRemovalMigration)

		assert.False(t, db.Migrator().HasTable("storages_rclone"))
		assert.False(t, db.Migrator().HasTable("rclone_storages"))

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
			db := openDatabaseMigratedBefore(t, rcloneRemovalMigration)
			applyAllMigrations(t, db, rcloneRemovalMigration)
			require.False(t, db.Migrator().HasTable("storages_rclone"))
			sqlDB, err := db.DB()
			require.NoError(t, err)
			provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.Files,
				goose.WithDisableGlobalRegistry(true))
			require.NoError(t, err)

			_, err = provider.DownTo(t.Context(), 20261109134344)
			require.NoError(t, err)
			assert.True(t, db.Migrator().HasTable("storages_rclone"))
			assert.Empty(t, listIDs(t, db, "SELECT storage_id FROM storages_rclone"))

			_, err = database.Migrate(t.Context(), db, migrations.Files)
			require.NoError(t, err)
			assert.False(t, db.Migrator().HasTable("storages_rclone"))
		},
	)
}

func getRcloneRetirementSnapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()

	var snapshot strings.Builder
	for _, tableName := range []string{"storages", "storages_rclone", "goose_db_version"} {
		var tableSnapshot string
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT COALESCE(jsonb_agg(to_jsonb(records) ORDER BY to_jsonb(records)::text), '[]')::text FROM "+tableName+" records",
		).
			Scan(&tableSnapshot).Error)
		snapshot.WriteString(tableSnapshot)
	}

	return snapshot.String()
}
