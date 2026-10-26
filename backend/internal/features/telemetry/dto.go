package telemetry

import (
	"context"
	"time"

	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
)

type DatabaseEntry struct {
	Type               string                     `json:"type"`
	Version            string                     `json:"version"`
	BackupType         string                     `json:"backupType,omitzero"`
	IsSshTunnelEnabled bool                       `json:"isSshTunnelEnabled"`
	RawSizeMb          int64                      `json:"rawSizeMb,omitzero"`
	BackupSizeMb       int64                      `json:"backupSizeMb,omitzero"`
	Verification       *DatabaseVerificationEntry `json:"verification,omitempty"`
}

type DatabaseVerificationEntry struct {
	IsEnabled    bool   `json:"isEnabled"`
	ScheduleType string `json:"scheduleType"`
	IntervalType string `json:"intervalType,omitempty"`
}

type VerificationAgentEntry struct {
	MaxCPU            int `json:"maxCpu"`
	MaxRAMGb          int `json:"maxRamGb"`
	MaxDiskGb         int `json:"maxDiskGb"`
	MaxConcurrentJobs int `json:"maxConcurrentJobs"`
}

type CollectRequest struct {
	InstanceID         string                   `json:"instanceID"`
	AppVersion         string                   `json:"appVersion"`
	OS                 string                   `json:"os"`
	Arch               string                   `json:"arch"`
	InstalledAt        string                   `json:"installedAt,omitempty"`
	UserCount          int                      `json:"userCount,omitzero"`
	Databases          []DatabaseEntry          `json:"databases"`
	Storages           []string                 `json:"storages"`
	Notifiers          []string                 `json:"notifiers"`
	VerificationAgents []VerificationAgentEntry `json:"verificationAgents"`
}

type databaseLister interface {
	GetTelemetry(ctx context.Context) ([]databases.TelemetryEntry, error)
}

type storageTypeLister interface {
	GetStorageTypes(ctx context.Context) ([]storages.Type, error)
}

type notifierLister interface {
	GetNotifierTypes(ctx context.Context) ([]notifiers.Type, error)
}

type backupChecker interface {
	HasSuccessfulBackupSince(databaseID uuid.UUID, since time.Time) (bool, error)
	GetLatestCompletedBackup(databaseID uuid.UUID) (*backups_core_logical.LogicalBackup, error)
}

type latestPhysicalBackupReader interface {
	GetLatestCompletedFullBackup(databaseID uuid.UUID) (*physical_models.PhysicalFullBackup, error)
}

type userCounter interface {
	GetUserCount(ctx context.Context) (int64, error)
}

type verificationAgentLister interface {
	ListAgents(ctx context.Context) ([]*verification_agents.Agent, error)
}

type verificationConfigLister interface {
	ListEnabled() ([]*verification_config.BackupVerificationConfig, error)
}
