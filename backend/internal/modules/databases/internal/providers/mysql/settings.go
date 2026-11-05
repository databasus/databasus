// Package mysql backs up one MySQL database with mysqldump.
package mysql

import (
	mysqlengine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/modules/databases/internal/providers"
)

type Settings struct {
	// Version is what the server reported at the last save.
	Version  mysqlengine.Version
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
	Privileges []string
}

func (Settings) GetType() providers.Type {
	return providers.TypeMysql
}
