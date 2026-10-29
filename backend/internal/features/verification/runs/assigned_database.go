package verification_runs

import (
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
