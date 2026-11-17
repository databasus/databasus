// Package health reports how well the instance is protected by Databasus backups and keeps the
// incidents that warn root admin, in-app and through the system mail server.
package health

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/instancebackups/internal/recovery"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
)

// Health is how well the instance is protected.
type Health string

const (
	HealthNotConfigured Health = "NOT_CONFIGURED"
	HealthInitializing  Health = "INITIALIZING"
	HealthHealthy       Health = "HEALTHY"
	HealthPartial       Health = "PARTIAL"
	HealthFailed        Health = "FAILED"
	HealthOverdue       Health = "OVERDUE"
	HealthStopped       Health = "STOPPED"
	HealthResuming      Health = "RESUMING"
)

// DestinationHealth is the protection of one selected destination.
type DestinationHealth struct {
	StorageID                uuid.UUID
	StorageName              string
	StorageType              storages.Type
	WorkspaceID              uuid.UUID
	WorkspaceName            string
	Health                   Health
	LastSuccessfulSnapshotAt *time.Time
	OverdueAt                *time.Time
	LastError                *string
}

// Status is everything root admin's protection view shows.
type Status struct {
	Health       Health
	Destinations []DestinationHealth
	// Incident is the open incident, nil when protection has no problem.
	Incident                      *IncidentView
	IsEmailConfigured             bool
	IsRootEmailUsable             bool
	IsMasterKeyAcknowledged       bool
	IsMasterKeyExportCodeRequired bool
	IdentityProofMethods          []users.IdentityProofMethod
	MasterKeyPath                 string
	Reminder                      configs.Reminder
	IsRecoveryPaused              bool
	// Recovery is the latest recovery of the local journal, nil when there was none.
	Recovery *recovery.Recovery
}
