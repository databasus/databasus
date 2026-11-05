// Package postgresqllogical backs up one PostgreSQL database with pg_dump.
package postgresqllogical

import (
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases/internal/providers"
)

type Settings struct {
	// Version is what the server reported at the last save.
	Version  pgshared.Version
	Host     string
	Port     int
	Username string
	// Password is a secret.
	Password string
	// Database is the database the dump reads, empty until the user picks one.
	Database  string
	TLS       providers.PostgresqlTLSSettings
	SSHTunnel providers.SSHTunnelSettings
	// IncludeSchemas is empty for every schema.
	IncludeSchemas     []string
	ExcludeTables      []string
	ExcludeTableData   []string
	CPUCount           int
	IsSkipUserMappings bool
}

func (Settings) GetType() providers.Type {
	return providers.TypePostgresqlLogical
}
