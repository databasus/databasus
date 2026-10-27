package backups_services

import (
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
	ErrStorageChangeBackupInProgress = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a backup of the database is in progress, so its backups cannot move to another storage",
		Errors: nil,
	}
)
