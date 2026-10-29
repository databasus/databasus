package verification_runs

import (
	"time"

	"github.com/google/uuid"
)

type RestoreVerification struct {
	ID         uuid.UUID  `json:"id"                gorm:"column:id;type:uuid;primaryKey"`
	DatabaseID uuid.UUID  `json:"databaseId"        gorm:"column:database_id;type:uuid;not null"`
	BackupID   uuid.UUID  `json:"backupId"          gorm:"column:backup_id;type:uuid;not null"`
	AgentID    *uuid.UUID `json:"agentId,omitempty" gorm:"column:agent_id;type:uuid"`

	Trigger VerificationTrigger `json:"trigger" gorm:"column:trigger;type:text;not null"`
	Status  VerificationStatus  `json:"status"  gorm:"column:status;type:text;not null"`

	AttemptCount int `json:"attemptCount" gorm:"column:attempt_count;type:integer;not null;default:1"`

	CreatedAt  time.Time  `json:"createdAt"           gorm:"column:created_at"`
	StartedAt  *time.Time `json:"startedAt,omitzero"  gorm:"column:started_at"`
	FinishedAt *time.Time `json:"finishedAt,omitzero" gorm:"column:finished_at"`

	RestoreDurationMs *int64 `json:"restoreDurationMs,omitempty" gorm:"column:restore_duration_ms"`
	VerifyDurationMs  *int64 `json:"verifyDurationMs,omitempty"  gorm:"column:verify_duration_ms"`

	PgRestoreExitCode       *int    `json:"pgRestoreExitCode,omitempty"       gorm:"column:pg_restore_exit_code"`
	DBSizeBytesAfterRestore *int64  `json:"dbSizeBytesAfterRestore,omitempty" gorm:"column:db_size_bytes_after_restore"`
	TableCount              *int    `json:"tableCount,omitempty"              gorm:"column:table_count"`
	SchemaCount             *int    `json:"schemaCount,omitempty"             gorm:"column:schema_count"`
	FailMessage             *string `json:"failMessage,omitempty"             gorm:"column:fail_message"`

	TableStats []RestoreVerificationTableStat `json:"tableStats,omitempty" gorm:"foreignKey:RestoreVerificationID;references:ID"`
}

func (RestoreVerification) TableName() string {
	return "verification_runs"
}

type RestoreVerificationTableStat struct {
	ID                    uuid.UUID `json:"id"         gorm:"column:id;type:uuid;primaryKey"`
	RestoreVerificationID uuid.UUID `json:"-"          gorm:"column:run_id;type:uuid;not null"`
	SchemaName            string    `json:"schemaName" gorm:"column:schema_name;type:text;not null"`
	Name                  string    `json:"name"       gorm:"column:name;type:text;not null"`
	RowCount              int64     `json:"rowCount"   gorm:"column:row_count;type:bigint;not null"`
}

func (RestoreVerificationTableStat) TableName() string {
	return "verification_run_table_stats"
}
