// Package restore restores logical backups into the target databases users name. A target's
// credentials live only in the memory of its running restore.
package restore

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusInProgress Status = "IN_PROGRESS"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
	StatusCanceled   Status = "CANCELED"
)

type Restore struct {
	ID       uuid.UUID
	BackupID uuid.UUID
	// DatabaseID is the database of the backup, which the schema keeps equal to the backup's.
	DatabaseID        uuid.UUID
	Status            Status
	FailMessage       *string
	RestoreDurationMs int64
	CreatedAt         time.Time
}
