package healthcheck_attempt

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
)

type HealthcheckAttemptSender interface {
	Notify(ctx context.Context, notifierIDs []uuid.UUID, notification notifiers.Notification)
}

type HealthcheckedDatabases interface {
	GetDatabaseByID(ctx context.Context, id uuid.UUID) (databases.Database, error)

	ProbeLiveness(ctx context.Context, databaseID uuid.UUID) error

	SetHealthStatus(ctx context.Context, databaseID uuid.UUID, healthStatus *databases.HealthStatus) error
}
