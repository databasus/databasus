package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const backupsLogicalLargestIndexSizesMigrationName = "20261124120000_backups_logical_largest_index_sizes.sql"

func TestBackupsLogicalLargestIndexSizesMigration(t *testing.T) {
	t.Parallel()

	t.Run("leaves the index sizes of an existing backup unrecorded", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupsLogicalLargestIndexSizesMigrationName)
		backupID := insertVerificationFixture(t, db).backupID

		applyAllMigrations(t, db, backupsLogicalLargestIndexSizesMigrationName)

		var unrecordedBackupIDs []uuid.UUID
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT id FROM backups_logical WHERE backup_largest_index_sizes_mb IS NULL",
		).Scan(&unrecordedBackupIDs).Error)
		assert.Equal(t, []uuid.UUID{backupID}, unrecordedBackupIDs)
	})

	t.Run("accepts up to eight nonnegative sizes and an empty list", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupsLogicalLargestIndexSizesMigrationName)
		applyAllMigrations(t, db, backupsLogicalLargestIndexSizesMigrationName)
		backupID := insertVerificationFixture(t, db).backupID

		for _, indexSizes := range []string{"{}", "{8,7,6,5,4,3,2,0}"} {
			require.NoError(t, setLargestIndexSizes(t, db, backupID, indexSizes), indexSizes)
		}
	})

	t.Run("refuses a ninth size, a negative size and a missing size", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, backupsLogicalLargestIndexSizesMigrationName)
		applyAllMigrations(t, db, backupsLogicalLargestIndexSizesMigrationName)
		backupID := insertVerificationFixture(t, db).backupID

		for _, indexSizes := range []string{"{9,8,7,6,5,4,3,2,1}", "{3,-1}", "{3,NULL}"} {
			require.Error(t, setLargestIndexSizes(t, db, backupID, indexSizes), indexSizes)
		}
	})
}

func setLargestIndexSizes(t *testing.T, db *gorm.DB, backupID uuid.UUID, indexSizes string) error {
	t.Helper()

	return db.WithContext(t.Context()).Exec(
		"UPDATE backups_logical SET backup_largest_index_sizes_mb = ?::double precision[] WHERE id = ?",
		indexSizes,
		backupID,
	).Error
}
