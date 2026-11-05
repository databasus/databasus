package verifications

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
)

type AssignedDatabase struct {
	Type              databases.Type
	PostgresqlLogical *AssignedPostgresql
}

type AssignedPostgresql struct {
	Version pgshared.Version
}

func readAssignedDatabase(
	ctx context.Context,
	databasesModule databases.Module,
	database databases.Database,
) (AssignedDatabase, error) {
	versions, err := databasesModule.GetPostgresqlLogicalVersions(ctx, []uuid.UUID{database.ID})
	if err != nil {
		return AssignedDatabase{}, err
	}

	version := versions[database.ID]
	if version == "" {
		return AssignedDatabase{}, fmt.Errorf("%w: postgresql logical version unavailable", errDatabaseNotVerifiable)
	}

	return AssignedDatabase{
		Type:              database.Type,
		PostgresqlLogical: &AssignedPostgresql{Version: pgshared.Version(version)},
	}, nil
}
