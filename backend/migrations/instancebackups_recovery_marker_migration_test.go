package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/migrations"
)

func TestInstanceBackupsRecoveryMarkerMigration(t *testing.T) {
	t.Parallel()

	t.Run("keeps the recovery that produced the database after the rename", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261118120000_instancebackups_recovery_marker.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		recoveryID := uuid.New()
		execute(t, db, `UPDATE instancebackups_configs
			SET recovery_paused_operation_id = ?, recovery_paused_at = '2026-11-18T10:00:00Z' WHERE id`, recoveryID)

		applyAllMigrations(t, db, migrationName)

		recoveredByIDs := listIDs(t, db,
			"SELECT recovered_by_operation_id FROM instancebackups_configs WHERE recovered_at IS NOT NULL")
		require.Len(t, recoveredByIDs, 1)
		assert.Equal(t, recoveryID, recoveredByIDs[0])
	})

	t.Run("restores the pause columns and constraint with the marker when rolled back", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261118120000_instancebackups_recovery_marker.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		applyAllMigrations(t, db, migrationName)
		recoveryID := uuid.New()
		execute(t, db, `UPDATE instancebackups_configs
			SET recovered_by_operation_id = ?, recovered_at = '2026-11-18T10:00:00Z'`, recoveryID)

		rollBackMigration(t, db, 20261118120000)

		pausedIDs := listIDs(t, db,
			"SELECT recovery_paused_operation_id FROM instancebackups_configs WHERE recovery_paused_at IS NOT NULL")
		require.Len(t, pausedIDs, 1)
		assert.Equal(t, recoveryID, pausedIDs[0])
		assert.Equal(t, []bool{true}, listBools(t, db, `SELECT EXISTS (SELECT 1 FROM pg_constraint
			WHERE conname = 'chk_instancebackups_configs_recovery_pause')`))
	})
}

func rollBackMigration(t *testing.T, db *gorm.DB, version int64) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.Files,
		goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)

	_, err = provider.ApplyVersion(t.Context(), version, false)
	require.NoError(t, err)
}
