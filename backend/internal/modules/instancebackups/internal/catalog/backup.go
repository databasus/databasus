package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Status is where a Databasus backup is in its life, across the dump and every copy.
type Status string

const (
	StatusInProgress  Status = "IN_PROGRESS"
	StatusCompleted   Status = "COMPLETED"
	StatusPartial     Status = "PARTIAL"
	StatusFailed      Status = "FAILED"
	StatusInterrupted Status = "INTERRUPTED"
)

// DumpStatus is where the one dump of a backup is; at most one row is RUNNING at a time.
type DumpStatus string

const (
	DumpStatusRunning     DumpStatus = "RUNNING"
	DumpStatusSucceeded   DumpStatus = "SUCCEEDED"
	DumpStatusFailed      DumpStatus = "FAILED"
	DumpStatusInterrupted DumpStatus = "INTERRUPTED"
)

// Trigger is what started a backup.
type Trigger string

const (
	TriggerSchedule         Trigger = "SCHEDULE"
	TriggerManual           Trigger = "MANUAL"
	TriggerDestinationAdded Trigger = "DESTINATION_ADDED"
)

// Backup is one encrypted snapshot of the internal database. SpoolPath names its file in the local
// spool while copies still need it.
type Backup struct {
	ID             uuid.UUID  `gorm:"column:id"`
	Status         Status     `gorm:"column:status"`
	DumpStatus     DumpStatus `gorm:"column:dump_status"`
	Trigger        Trigger    `gorm:"column:trigger"`
	SourceRelease  string     `gorm:"column:source_release"`
	StartedAt      time.Time  `gorm:"column:started_at"`
	SnapshotAt     *time.Time `gorm:"column:snapshot_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
	SizeBytes      *int64     `gorm:"column:size_bytes"`
	FailureReason  *string    `gorm:"column:failure_reason"`
	SpoolPath      *string    `gorm:"column:spool_path"`
	SpoolExpiresAt *time.Time `gorm:"column:spool_expires_at"`
}

func (Backup) TableName() string {
	return "instancebackups_backups"
}

// dumpEnd is the final state of a backup that ends with its dump.
type dumpEnd struct {
	DumpStatus    DumpStatus
	Status        Status
	FailureReason string
	CompletedAt   time.Time
}

// BackupWithCopies is a backup as history shows it.
type BackupWithCopies struct {
	Backup Backup
	Copies []Copy
}

// BackupPage is one page of history, the newest snapshot first.
type BackupPage struct {
	Items      []BackupWithCopies
	NextCursor *string
}

// getOutcome is the final status of a backup whose copies all ended. A backup no copy of which was
// stored because Databasus stopped is interrupted rather than failed.
func getOutcome(copies []Copy) Status {
	storedCount := 0
	isInterrupted := false

	for _, backupCopy := range copies {
		switch backupCopy.Status {
		case CopyStatusStored, CopyStatusDeleted:
			storedCount++
		case CopyStatusInterrupted:
			isInterrupted = true
		case CopyStatusPending, CopyStatusUploading, CopyStatusRetrying, CopyStatusFailed:
		}
	}

	switch {
	case storedCount == len(copies):
		return StatusCompleted
	case storedCount > 0:
		return StatusPartial
	case isInterrupted:
		return StatusInterrupted
	default:
		return StatusFailed
	}
}
