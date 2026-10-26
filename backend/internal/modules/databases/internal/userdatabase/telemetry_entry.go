package userdatabase

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases/internal/providers"
)

// TelemetryEntry is what the anonymous usage statistics report about one database. The ID and
// the last backup time stay in the process: they decide whether the database counts as active.
type TelemetryEntry struct {
	DatabaseID     uuid.UUID
	Type           Type
	HealthStatus   *HealthStatus
	LastBackupTime *time.Time
	providers.TelemetryFacts
}
