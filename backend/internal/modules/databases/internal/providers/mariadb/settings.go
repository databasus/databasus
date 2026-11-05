// Package mysql backs up one MariaDB database with mariadb-dump.
package mariadb

import (
	mariadbengine "databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/modules/databases/internal/providers"
)

type Settings struct {
	// Version is what the server reported at the last save.
	Version  mariadbengine.Version
	Host     string
	Port     int
	Username string
	// Password is a secret.
	Password string
	// Database is the database the dump reads, empty until the user picks one.
	Database string
	// IsHTTPS encrypts the connection without verifying the server certificate.
	IsHTTPS       bool
	SSHTunnel     providers.SSHTunnelSettings
	ExcludeTables []string
	// Privileges are the backup privileges the account held at the last save.
	Privileges      []string
	IsExcludeEvents bool
	// IsSkipGaleraDisable keeps the dump from turning off Galera replication for its session.
	IsSkipGaleraDisable bool
}

func (Settings) GetType() providers.Type {
	return providers.TypeMariadb
}
