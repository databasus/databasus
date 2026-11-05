package mysql

import "errors"

var (
	ErrNoClientForArchitecture     = errors.New("no client is shipped for this architecture")
	ErrUnreadableClientVersion     = errors.New("could not read a MySQL version")
	ErrDatabaseNameRequired        = errors.New("database name is required for MySQL backup")
	ErrDumpDatabaseNameRequired    = errors.New("database name is required for mysqldump backups")
	ErrRestoreDatabaseNameRequired = errors.New("target database name is required for mysql restore")
)
