package files

import (
	"time"

	"github.com/google/uuid"
)

// PendingDeletion is the obligation to remove one file. Row presence is the whole state: there is no
// status column, because a row exists exactly while nobody has committed to keeping the file it
// names.
type PendingDeletion struct {
	CleanupDatabaseID  *uuid.UUID `gorm:"column:cleanup_database_id;type:uuid"`
	CleanupWorkspaceID *uuid.UUID `gorm:"column:cleanup_workspace_id;type:uuid"`
	ID                 uuid.UUID  `gorm:"column:id;primaryKey;type:uuid"`
	StorageID          uuid.UUID  `gorm:"column:storage_id;not null;type:uuid"`
	FileName           string     `gorm:"column:file_name;not null;type:text"`
	NotBefore          time.Time  `gorm:"column:not_before;not null"`
	// Generation is what a write receipt matches. Every event that takes the file away from its
	// writer raises it, which invalidates the receipt the writer is holding.
	Generation int `gorm:"column:generation;not null"`
	// AttemptCount counts deletion attempts and nothing else, so the backoff grows with failures
	// rather than with how often the file was requested.
	AttemptCount int       `gorm:"column:attempt_count;not null"`
	LastError    *string   `gorm:"column:last_error;type:text"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (PendingDeletion) TableName() string {
	return "storages_pending_file_deletions"
}

func (p *PendingDeletion) GetReference() StoredFileReference {
	return StoredFileReference{StorageID: p.StorageID, FileName: p.FileName}
}
