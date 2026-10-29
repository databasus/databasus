package verifications

import (
	"time"

	"github.com/google/uuid"
)

type Verification struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	DatabaseID uuid.UUID  `gorm:"column:database_id;type:uuid;not null"`
	BackupID   uuid.UUID  `gorm:"column:backup_id;type:uuid;not null"`
	AgentID    *uuid.UUID `gorm:"column:agent_id;type:uuid"`

	Trigger VerificationTrigger `gorm:"column:trigger;type:text;not null"`
	Status  VerificationStatus  `gorm:"column:status;type:text;not null"`

	AttemptCount int `gorm:"column:attempt_count;type:integer;not null;default:1"`

	CreatedAt  time.Time  `gorm:"column:created_at"`
	StartedAt  *time.Time `gorm:"column:started_at"`
	FinishedAt *time.Time `gorm:"column:finished_at"`

	RestoreDurationMs *int64 `gorm:"column:restore_duration_ms"`
	VerifyDurationMs  *int64 `gorm:"column:verify_duration_ms"`

	PgRestoreExitCode         *int    `gorm:"column:pg_restore_exit_code"`
	RestoredDatabaseSizeBytes *int64  `gorm:"column:db_size_bytes_after_restore"`
	TableCount                *int    `gorm:"column:table_count"`
	SchemaCount               *int    `gorm:"column:schema_count"`
	FailMessage               *string `gorm:"column:fail_message"`

	TableStats []TableStat `gorm:"-"`
}

func (Verification) TableName() string {
	return "verification_runs"
}

type VerificationStatus string

const (
	VerificationStatusPending   VerificationStatus = "PENDING"
	VerificationStatusRunning   VerificationStatus = "RUNNING"
	VerificationStatusCompleted VerificationStatus = "COMPLETED"
	VerificationStatusFailed    VerificationStatus = "FAILED"
	VerificationStatusCanceled  VerificationStatus = "CANCELED"
)

type VerificationTrigger string

const (
	VerificationTriggerManual      VerificationTrigger = "MANUAL"
	VerificationTriggerScheduled   VerificationTrigger = "SCHEDULED"
	VerificationTriggerAfterBackup VerificationTrigger = "AFTER_BACKUP"
)
