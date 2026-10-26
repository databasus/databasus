package healthcheck_attempt

import (
	"context"

	"github.com/google/uuid"

	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/databases"
)

type moduleHealthcheckedDatabases struct {
	databaseAccess *databases_bridge.Access
	// getDatabases is read per call: each test application installs its own module.
	getDatabases func() databases.Module
}

func (d moduleHealthcheckedDatabases) GetDatabaseByID(ctx context.Context, id uuid.UUID) (databases.Database, error) {
	return d.databaseAccess.GetDatabaseByID(ctx, id)
}

func (d moduleHealthcheckedDatabases) ProbeLiveness(ctx context.Context, databaseID uuid.UUID) error {
	return d.getDatabases().ProbeLiveness(ctx, databaseID)
}

func (d moduleHealthcheckedDatabases) SetHealthStatus(
	ctx context.Context,
	databaseID uuid.UUID,
	healthStatus *databases.HealthStatus,
) error {
	return d.getDatabases().SetHealthStatus(ctx, databaseID, healthStatus)
}
