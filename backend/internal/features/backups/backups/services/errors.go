package backups_services

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	// ErrDatabaseBackupInProgress refuses a database deletion through the databases module, which
	// answers it as it is.
	ErrDatabaseBackupInProgress = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a backup of the database is in progress, so the database cannot be deleted",
		Errors: nil,
	}
	errBackupInProgress = errors.New("backup is in progress, storage cannot be removed")
)
