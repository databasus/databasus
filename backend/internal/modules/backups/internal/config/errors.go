package config

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrStaleStoragePublication = errors.New("physical publication belongs to an inactive storage generation")
	ErrDatabaseNotFound        = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the database does not exist",
		Errors: nil,
	}
	ErrStorageNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the storage does not exist",
		Errors: nil,
	}
	ErrMethodMismatch = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "must be the backup method of the database"},
	})
	ErrStorageNotInDatabaseWorkspace = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "storageId", Detail: "must name a storage of the database's workspace"},
	})
	ErrListFilterRequired = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "workspaceId", Detail: "exactly one of workspaceId and storageId is required"},
	})
	// ErrStorageUsedOutsideTargetWorkspace refuses a storage transfer that would leave a database
	// backing up into a storage of another workspace.
	ErrStorageUsedOutsideTargetWorkspace = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a database outside the target workspace backs up into the storage",
		Errors: nil,
	}
	ErrNotLogicalDatabase  = errors.New("the database is not backed up logically")
	ErrNotPhysicalDatabase = errors.New("the database is not backed up physically")
	errUnknownBackupType   = errors.New("the database has an unknown physical backup type")
	errMethodRowMissing    = errors.New("a backup config has no row of its method")
)

func newInvalidFieldError(field, detail string) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: field, Detail: detail}})
}

func newInvalidSettingError(field, detail string) error {
	return newInvalidFieldError("settings."+field, detail)
}
