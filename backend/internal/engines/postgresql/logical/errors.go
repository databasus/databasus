package logical

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrDatabaseNameRequired = errors.New("database name is required for single database backup (pg_dump)")
	ErrCreateRoleMissing    = errors.New("current database user lacks CREATEROLE privilege")
	ErrUserNameCollision    = errors.New("failed to generate unique username")
	ErrUnsupportedVersion   = errors.New("unsupported PostgreSQL version")
	ErrUnreadableVersion    = errors.New("could not parse version")
	ErrRowLevelSecurity     = errors.New("row-level security is enabled")

	ErrDumpDatabaseNameRequired    = errors.New("database name is required for pg_dump backups")
	ErrRestoreDatabaseNameRequired = errors.New("target database name is required for pg_restore")
)

type ArchiveRestoreError struct {
	Err error
}

func (e *ArchiveRestoreError) Error() string {
	return e.Err.Error()
}

func (e *ArchiveRestoreError) Unwrap() error {
	return e.Err
}

type serverErrorKind int

const (
	serverErrorOther serverErrorKind = iota
	serverErrorNameTaken
	serverErrorPermissionDenied
)

// classifyServerError reads what the server reported from its SQLSTATE: 42710 duplicate_object
// when a role name is taken, 42501 insufficient_privilege when the connecting role may not do what
// it asked.
func classifyServerError(err error) serverErrorKind {
	pgErr, isPgErr := errors.AsType[*pgconn.PgError](err)
	if !isPgErr {
		return serverErrorOther
	}

	switch pgErr.Code {
	case "42710":
		return serverErrorNameTaken
	case "42501":
		return serverErrorPermissionDenied
	default:
		return serverErrorOther
	}
}
