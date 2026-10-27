//go:build testhelpers

package backups_bridge

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/storage"
)

// SeedTestLogicalBackup inserts a completed logical backup row for the old tests, which need a backup
// without running a dump. Its file does not exist in the storage.
func SeedTestLogicalBackup(t *testing.T, databaseID, storageID uuid.UUID, sizeMb float64) backups.LogicalBackup {
	t.Helper()

	return seedLogicalBackup(t, databaseID, storageID, sizeMb, backups.LogicalBackupStatusCompleted)
}

// SeedInProgressTestLogicalBackup inserts a running logical backup row that no run drives.
func SeedInProgressTestLogicalBackup(t *testing.T, databaseID, storageID uuid.UUID) backups.LogicalBackup {
	t.Helper()

	return seedLogicalBackup(t, databaseID, storageID, 0, backups.LogicalBackupStatusInProgress)
}

func seedLogicalBackup(
	t *testing.T,
	databaseID, storageID uuid.UUID,
	sizeMb float64,
	status backups.LogicalBackupStatus,
) backups.LogicalBackup {
	t.Helper()

	backupID := uuid.New()
	seededBackup := backups.LogicalBackup{
		ID:                        backupID,
		DatabaseID:                databaseID,
		StorageID:                 storageID,
		FileName:                  "test-backup-" + backupID.String(),
		Status:                    status,
		FailMessage:               nil,
		BackupSizeMb:              sizeMb,
		BackupRawDbSizeMb:         sizeMb,
		BackupDurationMs:          0,
		Encryption:                backups.EncryptionNone,
		EncryptionSalt:            nil,
		EncryptionIV:              nil,
		TimescaledbVersion:        "",
		RestoreVerificationStatus: backups.LogicalBackupVerificationStatusNotVerified,
		CreatedAt:                 time.Now().UTC().Truncate(time.Microsecond),
	}

	err := storage.GetDb().WithContext(t.Context()).Exec(`INSERT INTO backups_logical
		(id, database_id, storage_id, file_name, status, backup_size_mb, backup_raw_db_size_mb,
		encryption, restore_verification_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		seededBackup.ID, seededBackup.DatabaseID, seededBackup.StorageID, seededBackup.FileName,
		seededBackup.Status, seededBackup.BackupSizeMb, seededBackup.BackupRawDbSizeMb, seededBackup.Encryption,
		seededBackup.RestoreVerificationStatus, seededBackup.CreatedAt,
	).Error
	if err != nil {
		t.Fatalf("seed a %s logical backup: %v", status, err)
	}

	return seededBackup
}
