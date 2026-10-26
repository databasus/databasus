package databases_bridge

import "errors"

var (
	ErrDatabaseNotFound        = errors.New("database not found")
	ErrInsufficientPermissions = errors.New("insufficient permissions to access this database")
)
