package backups_core_logical

import (
	"errors"
	"time"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	storage_files "databasus-backend/internal/features/storages/files"
)

type BackupFilters struct {
	Statuses   []BackupStatus
	BeforeDate *time.Time
}

type BackupMetadata struct {
	BackupID       uuid.UUID                           `json:"backupId"`
	EncryptionSalt *string                             `json:"encryptionSalt"`
	EncryptionIV   *string                             `json:"encryptionIV"`
	Encryption     backups_core_enums.BackupEncryption `json:"encryption"`
}

func (m *BackupMetadata) Validate() error {
	if m.BackupID == uuid.Nil {
		return errors.New("backup ID is required")
	}

	if m.Encryption == "" {
		return errors.New("encryption is required")
	}

	if m.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		if m.EncryptionSalt == nil {
			return errors.New("encryption salt is required when encryption is enabled")
		}

		if m.EncryptionIV == nil {
			return errors.New("encryption IV is required when encryption is enabled")
		}
	}

	return nil
}

// The receipts are what the publishing transaction spends to keep the files this
// attempt uploaded.
type BackupArtifacts struct {
	Metadata *BackupMetadata
	Receipts []storage_files.WriteReceipt
}

type DatabaseBackupTotals struct {
	DatabaseID            uuid.UUID `gorm:"column:database_id"`
	BackupsCount          int64     `gorm:"column:backups_count"`
	CompletedBackupsCount int64     `gorm:"column:completed_backups_count"`
	FailedBackupsCount    int64     `gorm:"column:failed_backups_count"`
	CompletedBackupSizeMb float64   `gorm:"column:completed_backup_size_mb"`
}

type BackupTotals struct {
	BackupsCount          int64   `gorm:"column:backups_count"`
	CompletedBackupSizeMb float64 `gorm:"column:completed_backup_size_mb"`
}
