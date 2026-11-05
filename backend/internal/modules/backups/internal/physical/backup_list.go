package physical

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type PhysicalBackupListItem struct {
	ID         uuid.UUID
	Type       catalog.BackupType
	Status     string
	TimelineID int

	StartLSN string
	StopLSN  string

	RootFullBackupID          *uuid.UUID
	ParentIncrementalBackupID *uuid.UUID

	FromWalFilename  *string
	ToWalFilename    *string
	WalSegmentsCount *int
	HasGapBefore     bool

	SizeMb float64

	FailMessage *string

	CreatedAt   time.Time
	CompletedAt *time.Time
}

type GetPhysicalBackupsRequest struct {
	Limit      *int
	Cursor     string
	Types      []catalog.BackupType
	Statuses   []string
	BeforeDate *time.Time
}

type GetPhysicalBackupsResponse struct {
	Backups      []PhysicalBackupListItem
	TotalUsageMb float64

	NextCursor string
}
