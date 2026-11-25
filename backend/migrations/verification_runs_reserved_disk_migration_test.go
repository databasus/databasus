package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const verificationRunsReservedDiskMigrationName = "20261124120100_verification_runs_reserved_disk.sql"

func TestVerificationRunsReservedDiskMigration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name               string
		status             string
		backupSizeMb       float64
		rawDbSizeMb        float64
		expectedReservedMb int64
	}{
		{
			name:               "reserves archive, database and half the database for a running verification",
			status:             "RUNNING",
			backupSizeMb:       10000,
			rawDbSizeMb:        6000.5,
			expectedReservedMb: 19001,
		},
		{
			name:               "reserves archive and the 5120 MB gap for a running verification of an unknown size",
			status:             "RUNNING",
			backupSizeMb:       10000,
			rawDbSizeMb:        0,
			expectedReservedMb: 15120,
		},
		{
			name:               "reserves nothing for a pending verification",
			status:             "PENDING",
			backupSizeMb:       10000,
			rawDbSizeMb:        20000,
			expectedReservedMb: 0,
		},
	}

	for _, migrationCase := range cases {
		t.Run(migrationCase.name, func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, verificationRunsReservedDiskMigrationName)
			fixture := insertVerificationFixture(t, db)
			execute(t, db, "UPDATE backups_logical SET backup_size_mb = ?, backup_raw_db_size_mb = ? WHERE id = ?",
				migrationCase.backupSizeMb, migrationCase.rawDbSizeMb, fixture.backupID)
			verificationID := insertVerificationRun(t, db, fixture, migrationCase.status)

			applyAllMigrations(t, db, verificationRunsReservedDiskMigrationName)

			var reservedDiskMb int64
			require.NoError(t, db.WithContext(t.Context()).Raw(
				"SELECT reserved_disk_mb FROM verification_runs WHERE id = ?", verificationID,
			).Scan(&reservedDiskMb).Error)
			assert.Equal(t, migrationCase.expectedReservedMb, reservedDiskMb)
		})
	}
}

func insertVerificationRun(t *testing.T, db *gorm.DB, fixture verificationFixture, status string) uuid.UUID {
	t.Helper()

	verificationID := uuid.New()
	execute(t, db, `INSERT INTO verification_runs (id, database_id, backup_id, status, trigger)
		VALUES (?, ?, ?, ?, 'MANUAL')`, verificationID, fixture.databaseID, fixture.backupID, status)

	return verificationID
}
