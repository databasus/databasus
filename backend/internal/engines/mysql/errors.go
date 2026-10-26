package mysql

import "errors"

var (
	ErrNoClientForArchitecture = errors.New("no client is shipped for this architecture")
	ErrUnreadableClientVersion = errors.New("could not read a MySQL version")
	ErrDatabaseNameRequired    = errors.New("database name is required for MySQL backup")
)
