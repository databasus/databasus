package logical

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrDatabaseNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the database does not exist",
		Errors: nil,
	}
	ErrBackupNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the backup does not exist",
		Errors: nil,
	}
	ErrNoStorage = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "databaseId", Detail: "must name a database whose backup config names a storage"},
	})
	ErrPhysicalDatabase = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "databaseId", Detail: "must name a database with logical backups"},
	})
	ErrBackupAlreadyRunning = &apierror.Problem{
		Code:   apierror.CodeAlreadyRunning,
		Detail: "a backup of the database is already running",
		Errors: nil,
	}
	ErrBackupNotRunning = &apierror.Problem{
		Code:   apierror.CodeNotRunning,
		Detail: "the backup is no longer running",
		Errors: nil,
	}
	// ErrRunningBackupDeletion refuses to delete a backup that is still writing its file.
	ErrRunningBackupDeletion = &apierror.Problem{
		Code:   apierror.CodeAlreadyRunning,
		Detail: "the backup is running, so it cannot be deleted",
		Errors: nil,
	}
	ErrBackupReadByRestore = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a restore reads the backup, so it cannot be deleted",
		Errors: nil,
	}
	// ErrDatabaseBackupRunning refuses a database deletion through the databases module, which answers
	// it as it is.
	ErrDatabaseBackupRunning = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a backup of the database is in progress, so the database cannot be deleted",
		Errors: nil,
	}
	ErrDatabaseRestoreRunning = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a restore of a backup of the database is in progress, so the database cannot be deleted",
		Errors: nil,
	}
	ErrStorageChangeBackupRunning = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a backup of the database is in progress, so its backups cannot move to another storage",
		Errors: nil,
	}
	ErrStorageChangeRestoreRunning = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a restore of a backup of the database is in progress, so its backups cannot move to another storage",
		Errors: nil,
	}

	errEncryptionMetadataMissing = errors.New("the backup is marked as encrypted but has no encryption metadata")
	// errBackupNoLongerRunning fails a publication whose backup another writer ended meanwhile.
	errBackupNoLongerRunning = errors.New("the backup is no longer running")
	errWorkspaceNotFound     = errors.New("the workspace of the database does not exist")
)
