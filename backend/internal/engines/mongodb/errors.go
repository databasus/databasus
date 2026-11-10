package mongodb

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrUnreadableVersion       = errors.New("could not parse MongoDB version")
	ErrUnsupportedVersion      = errors.New("unsupported MongoDB version")
	ErrInsufficientPermissions = errors.New("insufficient permissions for backup")
	ErrUserNameCollision       = errors.New("failed to generate unique username")
	ErrTunnelPortRequired      = errors.New("SSH tunnel requires an explicit MongoDB port")

	ErrDumpDatabaseNameRequired    = errors.New("database name is required for mongodump backups")
	ErrRestoreDatabaseNameRequired = errors.New("target database name is required for mongorestore")

	errUserNotListed   = errors.New("user not listed")
	errUserUnparseable = errors.New("could not parse user info")
)

type serverErrorKind int

const (
	serverErrorOther serverErrorKind = iota
	serverErrorNameTaken
	serverErrorUnauthorized
)

// classifyServerError reads what the server reported from its error code: 51003 UserAlreadyExists
// when a user name is taken, 13 Unauthorized when the connecting user may not run the command.
func classifyServerError(err error) serverErrorKind {
	commandErr, isCommandErr := errors.AsType[mongo.CommandError](err)
	if !isCommandErr {
		return serverErrorOther
	}

	switch commandErr.Code {
	case 51003:
		return serverErrorNameTaken
	case 13:
		return serverErrorUnauthorized
	default:
		return serverErrorOther
	}
}
