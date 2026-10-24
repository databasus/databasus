package backups_config_logical

import "errors"

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
)
