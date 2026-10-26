package mariadb

import "errors"

var (
	ErrNoClientForArchitecture = errors.New("no client is shipped for this architecture")
	ErrUnreadableClientVersion = errors.New("could not read a MariaDB version")
	ErrDatabaseNameRequired    = errors.New("database name is required for MariaDB backup")
)
