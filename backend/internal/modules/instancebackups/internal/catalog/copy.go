package catalog

import (
	"time"

	"github.com/google/uuid"
)

// CopyStatus is where the delivery of one backup to one storage is.
type CopyStatus string

const (
	CopyStatusPending     CopyStatus = "PENDING"
	CopyStatusUploading   CopyStatus = "UPLOADING"
	CopyStatusRetrying    CopyStatus = "RETRYING"
	CopyStatusStored      CopyStatus = "STORED"
	CopyStatusFailed      CopyStatus = "FAILED"
	CopyStatusInterrupted CopyStatus = "INTERRUPTED"
	CopyStatusDeleted     CopyStatus = "DELETED"
)

// Copy is the delivery of one backup to the destination selected when the backup ran. StorageID is
// nil once the storage was deleted; the names stay as history.
type Copy struct {
	ID                  uuid.UUID  `gorm:"column:id"`
	BackupID            uuid.UUID  `gorm:"column:backup_id"`
	StorageID           *uuid.UUID `gorm:"column:storage_id"`
	SelectionGeneration int64      `gorm:"column:selection_generation"`
	StorageName         string     `gorm:"column:storage_name"`
	WorkspaceName       string     `gorm:"column:workspace_name"`
	Status              CopyStatus `gorm:"column:status"`
	AttemptCount        int        `gorm:"column:attempt_count"`
	NextAttemptAt       *time.Time `gorm:"column:next_attempt_at"`
	FileName            *string    `gorm:"column:file_name"`
	StoredAt            *time.Time `gorm:"column:stored_at"`
	LastError           *string    `gorm:"column:last_error"`
}

func (Copy) TableName() string {
	return "instancebackups_copies"
}

// IsDownloadable reports whether the copy's file can still be read from its storage.
func (c Copy) IsDownloadable() bool {
	return c.Status == CopyStatusStored && c.StorageID != nil && c.FileName != nil
}

func (c Copy) isEnded() bool {
	switch c.Status {
	case CopyStatusStored, CopyStatusFailed, CopyStatusInterrupted, CopyStatusDeleted:
		return true
	case CopyStatusPending, CopyStatusUploading, CopyStatusRetrying:
		return false
	}

	return false
}
