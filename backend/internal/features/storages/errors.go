package storages

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrInsufficientPermissionsToManageStorage = errors.New(
		"insufficient permissions to manage storage in this workspace",
	)
	ErrInsufficientPermissionsToViewStorage = errors.New(
		"insufficient permissions to view storage in this workspace",
	)
	ErrInsufficientPermissionsToViewStorages = errors.New(
		"insufficient permissions to view storages in this workspace",
	)
	ErrInsufficientPermissionsToTestStorage = errors.New(
		"insufficient permissions to test storage in this workspace",
	)
	ErrInsufficientPermissionsInSourceWorkspace = errors.New(
		"insufficient permissions to manage storage in source workspace",
	)
	ErrInsufficientPermissionsInTargetWorkspace = errors.New(
		"insufficient permissions to manage storage in target workspace",
	)
	ErrStorageDoesNotBelongToWorkspace = errors.New(
		"storage does not belong to this workspace",
	)
	ErrStorageInUse = errors.New(
		"storage is used by a backup config or still holds backups; remove them first, " +
			"they are the only record of the files it holds",
	)
	ErrStorageHasAttachedDatabasesCannotTransfer = errors.New(
		"storage has attached databases and cannot be transferred",
	)
	ErrStorageHasOtherAttachedDatabasesCannotTransfer = errors.New(
		"storage has other attached databases and cannot be transferred",
	)
	// Rclone accepts a freeform config blob whose `type =` may select backends
	// that read arbitrary local files (`local`, `alias`, `combine`, `union`,
	// `crypt`, `chunker`, `cache`) or initiate outbound connections to arbitrary
	// hosts (`http`, `webdav`, `sftp`, `ftp`) — LFI and SSRF surface that we
	// cannot expose to untrusted callers. Restrict to admins, who are the
	// instance operators and won't harm their own deployment.
	ErrRcloneStorageRequiresAdmin = errors.New(
		"rclone storage can only be managed by administrators",
	)
)

// ErrWorkspaceStorageInUse names no backup and no other workspace, so a refused workspace deletion
// reveals nothing its owner may not see.
var ErrWorkspaceStorageInUse = &apierror.Problem{
	Code:   apierror.CodeInUse,
	Detail: "a storage of the workspace is still in use",
	Errors: nil,
}
