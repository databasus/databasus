// Package userdatabase creates, lists, reads, changes, copies and deletes the user databases that
// Databasus backs up, keeps their parent rows and notifier attachments, and records their last backup
// and health.
package userdatabase

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases/internal/providers"
)

// Type is the kind of database. It is fixed when the database is created.
type Type = providers.Type

// HealthStatus is the outcome of the last scheduled health check, nil while health checks are off.
type HealthStatus string

const (
	HealthStatusAvailable   HealthStatus = "AVAILABLE"
	HealthStatusUnavailable HealthStatus = "UNAVAILABLE"
)

// Database is a database as other modules see it, without its settings.
type Database struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Type        Type
	NotifierIDs []uuid.UUID
	// LastBackupTime is the time of the newest successful backup of any kind, nil before the first.
	LastBackupTime         *time.Time
	LastBackupErrorMessage *string
	HealthStatus           *HealthStatus
}

// WithSettings is a database with the settings of its provider. Secrets in them stay encrypted.
type WithSettings struct {
	Database
	Settings providers.Settings
}

// Creation is a new database; the type of its settings becomes the database's type.
type Creation struct {
	WorkspaceID uuid.UUID
	Name        string
	Settings    providers.Settings
	NotifierIDs []uuid.UUID
}

// Change replaces the name, the settings and the notifiers of a database. A secret the settings
// leave out keeps its stored value while the resource address stays the stored one.
type Change struct {
	DatabaseID  uuid.UUID
	Name        string
	Settings    providers.Settings
	NotifierIDs []uuid.UUID
}
