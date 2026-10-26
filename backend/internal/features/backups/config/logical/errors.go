package backups_config_logical

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrInsufficientPermissionsInSourceWorkspace = errors.New(
		"insufficient permissions to manage database in source workspace",
	)
	ErrInsufficientPermissionsInTargetWorkspace = errors.New(
		"insufficient permissions to manage database in target workspace",
	)
	ErrTargetStorageNotInTargetWorkspace = errors.New(
		"target storage does not belong to target workspace",
	)
	ErrTargetNotifierNotInTargetWorkspace = errors.New(
		"target notifier does not belong to target workspace",
	)
	ErrStorageHasOtherAttachedDatabases = errors.New(
		"storage has other attached databases and cannot be transferred with this database",
	)
	ErrDatabaseHasNoStorage = errors.New(
		"database has no storage attached",
	)
	ErrDatabaseHasNoWorkspace = errors.New(
		"database has no workspace",
	)
	ErrTargetStorageNotSpecified = errors.New(
		"target storage is not specified",
	)
	ErrStorageWithoutID = errors.New(
		"storage must be referenced by the identifier of an existing storage",
	)
	ErrStorageNotFound = errors.New(
		"storage not found",
	)
	ErrStorageNotInDatabaseWorkspace = errors.New(
		"storage does not belong to the same workspace as the database",
	)
	ErrInsufficientPermissionsToViewStorage = errors.New(
		"insufficient permissions to view storage in this workspace",
	)
)

// ErrStorageUsedOutsideTargetWorkspace refuses a storage transfer that would leave a database using a
// storage of another workspace.
var ErrStorageUsedOutsideTargetWorkspace = &apierror.Problem{
	Code:   apierror.CodeInUse,
	Detail: "a database outside the target workspace uses the storage",
	Errors: nil,
}
