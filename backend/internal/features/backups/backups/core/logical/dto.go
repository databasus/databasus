package backups_core_logical

import (
	"errors"
	"time"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/modules/storages"
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
	Receipts []storages.WriteReceipt
}
