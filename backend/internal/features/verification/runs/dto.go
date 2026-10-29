package verification_runs

import (
	"github.com/google/uuid"
)

type AgentCapacity struct {
	MaxCPU            int `json:"maxCpu"            binding:"min=0"`
	MaxRAMMb          int `json:"maxRamMb"          binding:"min=0"`
	MaxDiskGb         int `json:"maxDiskGb"         binding:"min=0"`
	MaxConcurrentJobs int `json:"maxConcurrentJobs" binding:"min=0"`
}

type ClaimRequest struct {
	Capacity AgentCapacity `json:"capacity"`
}

type JobAssignment struct {
	VerificationID     uuid.UUID        `json:"verificationId"`
	BackupID           uuid.UUID        `json:"backupId"`
	BackupSizeMb       float64          `json:"backupSizeMb"`
	MaxContainerDiskMb float64          `json:"maxContainerDiskMb"`
	Database           AssignedDatabase `json:"database"`
	TimescaledbVersion string           `json:"timescaledbVersion"`
}

type ReportTableStat struct {
	SchemaName string `json:"schemaName" binding:"required"`
	Name       string `json:"name"       binding:"required"`
	RowCount   int64  `json:"rowCount"`
}

type ReportRequest struct {
	Status                  string            `json:"status"                  binding:"required,oneof=COMPLETED FAILED"`
	PgRestoreExitCode       *int              `json:"pgRestoreExitCode"`
	FailureKind             *string           `json:"failureKind"`
	RestoreDurationMs       *int64            `json:"restoreDurationMs"`
	VerifyDurationMs        *int64            `json:"verifyDurationMs"`
	DBSizeBytesAfterRestore *int64            `json:"dbSizeBytesAfterRestore"`
	TableCount              *int              `json:"tableCount"`
	SchemaCount             *int              `json:"schemaCount"`
	TableStats              []ReportTableStat `json:"tableStats"`
	FailMessage             *string           `json:"failMessage"`
}
