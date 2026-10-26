package healthcheck_attempt

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/notifiers"
)

type HealthcheckAttemptSender interface {
	Notify(ctx context.Context, notifierIDs []uuid.UUID, notification notifiers.Notification)
}

type DatabaseService interface {
	GetDatabaseByID(id uuid.UUID) (*databases.Database, error)

	ProbeDatabaseLiveness(ctx context.Context, database *databases.Database) error

	SetHealthStatus(
		databaseID uuid.UUID,
		healthStatus *databases.HealthStatus,
	) error
}
