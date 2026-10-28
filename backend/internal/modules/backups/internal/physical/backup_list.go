package physical

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type PhysicalBackupListItem struct {
	ID         uuid.UUID          `json:"id"`
	Type       catalog.BackupType `json:"type"`
	Status     string             `json:"status"`
	TimelineID int                `json:"timelineId"`

	StartLSN string `json:"startLsn"`
	StopLSN  string `json:"stopLsn"`

	RootFullBackupID          *uuid.UUID `json:"rootFullBackupId,omitempty"`
	ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId,omitempty"`

	FromWalFilename  *string `json:"fromWalFilename,omitempty"`
	ToWalFilename    *string `json:"toWalFilename,omitempty"`
	WalSegmentsCount *int    `json:"walSegmentsCount,omitempty"`
	HasGapBefore     bool    `json:"hasGapBefore,omitzero"`

	SizeMb float64 `json:"sizeMb"`

	FailMessage *string `json:"failMessage,omitempty"`

	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitzero"`
}

type GetPhysicalBackupsRequest struct {
	Limit      int                  `form:"limit"`
	Cursor     string               `form:"cursor"`
	Types      []catalog.BackupType `form:"type"`
	Statuses   []string             `form:"status"`
	BeforeDate *time.Time           `form:"beforeDate"`
}

type GetPhysicalBackupsResponse struct {
	Backups      []PhysicalBackupListItem `json:"backups"`
	TotalUsageMb float64                  `json:"totalUsageMb"`

	NextCursor string `json:"nextCursor,omitzero"`
}

type TriggerBackupType string

const (
	TriggerBackupTypeAuto        TriggerBackupType = "auto"
	TriggerBackupTypeFull        TriggerBackupType = "full"
	TriggerBackupTypeIncremental TriggerBackupType = "incremental"
)

type TriggerBackupRequest struct {
	Type TriggerBackupType `json:"type" binding:"required,oneof=auto full incremental"`
}

type GenerateRestoreTokenRequest struct {
	TargetTime *time.Time `json:"targetTime"`
}

type GenerateRestoreTokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}
