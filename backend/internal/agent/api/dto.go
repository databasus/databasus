package api

import (
	"time"

	"github.com/google/uuid"
)

type HeartbeatRequest struct {
	MaxCPU                 int         `json:"maxCpu"`
	MaxRAMGb               int         `json:"maxRamGb"`
	MaxDiskGb              int         `json:"maxDiskGb"`
	MaxConcurrentJobs      int         `json:"maxConcurrentJobs"`
	CurrentVerificationIDs []uuid.UUID `json:"currentVerificationIds"`
}

type HeartbeatResponse struct {
	LastSeenAt           time.Time   `json:"lastSeenAt"`
	AbortVerificationIDs []uuid.UUID `json:"abortVerificationIds"`
}

type versionResponse struct {
	Version string `json:"version"`
}

type AgentCapacity struct {
	MaxCPU            int `json:"maxCpu"`
	MaxRAMMb          int `json:"maxRamMb"`
	MaxDiskGb         int `json:"maxDiskGb"`
	MaxConcurrentJobs int `json:"maxConcurrentJobs"`
}

type ClaimRequest struct {
	Capacity AgentCapacity `json:"capacity"`
}

type AssignedPostgresql struct {
	Version string `json:"version"`
}

type AssignedDatabase struct {
	Type              string              `json:"type"`
	PostgresqlLogical *AssignedPostgresql `json:"postgresqlLogical"`
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
	SchemaName string `json:"schemaName"`
	Name       string `json:"name"`
	RowCount   int64  `json:"rowCount"`
}

type ReportRequest struct {
	Status                  VerificationStatus `json:"status"`
	PgRestoreExitCode       *int               `json:"pgRestoreExitCode"`
	FailureKind             *FailureType       `json:"failureKind"`
	RestoreDurationMs       *int64             `json:"restoreDurationMs"`
	VerifyDurationMs        *int64             `json:"verifyDurationMs"`
	DBSizeBytesAfterRestore *int64             `json:"dbSizeBytesAfterRestore"`
	TableCount              *int               `json:"tableCount"`
	SchemaCount             *int               `json:"schemaCount"`
	TableStats              []ReportTableStat  `json:"tableStats"`
	FailMessage             *string            `json:"failMessage"`
}
