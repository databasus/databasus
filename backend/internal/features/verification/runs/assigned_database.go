package verification_runs

import (
	"context"
	"fmt"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
)

// AssignedDatabase is what the verification agent learns about the database it restores: the type
// and the version it picks a client binary by. No connection setting leaves the server.
type AssignedDatabase struct {
	Type              databases.Type      `json:"type"`
	PostgresqlLogical *AssignedPostgresql `json:"postgresqlLogical"`
}

type AssignedPostgresql struct {
	Version pgshared.Version `json:"version"`
}

// readAssignedDatabase returns errDatabaseNotVerifiable for a database the agent cannot restore
// yet: an engine other than logical PostgreSQL, or one without a version to pick a client by. The
// scheduler skips enqueueing such a database, and the claim path skips handing it to an agent.
func readAssignedDatabase(
	ctx context.Context,
	databasesModule databases.Module,
	database databases.Database,
) (AssignedDatabase, error) {
	if database.Type != databases.TypePostgresqlLogical {
		return AssignedDatabase{}, fmt.Errorf(
			"%w: database type %q not supported for verification yet", errDatabaseNotVerifiable, database.Type,
		)
	}

	settings, err := databasesModule.GetPostgresqlLogicalConnectionSettings(ctx, database.ID)
	if err != nil {
		return AssignedDatabase{}, err
	}

	if settings.Version == "" {
		return AssignedDatabase{}, fmt.Errorf("%w: postgresql version empty", errDatabaseNotVerifiable)
	}

	return AssignedDatabase{
		Type:              database.Type,
		PostgresqlLogical: &AssignedPostgresql{Version: settings.Version},
	}, nil
}
