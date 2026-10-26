package databases

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var ErrInsufficientPermissionsToTestDatabaseConnection = errors.New(
	"insufficient permissions to test this database connection",
)

var ErrDatabaseConnectionTargetLookup = errors.New("failed to load database connection target")

var ErrDatabaseConnectionAuthorizationLookup = errors.New(
	"failed to verify database connection permissions",
)

// ErrWorkspaceHasDatabases refuses a workspace deletion through the workspaces module, which
// answers it as it is.
var ErrWorkspaceHasDatabases = &apierror.Problem{
	Code:   apierror.CodeWorkspaceNotEmpty,
	Detail: "the workspace still has databases, which must be deleted first",
	Errors: nil,
}

// ErrNotifierUsedOutsideTargetWorkspace refuses a notifier transfer through the notifiers module,
// which answers it as it is.
var ErrNotifierUsedOutsideTargetWorkspace = &apierror.Problem{
	Code:   apierror.CodeInUse,
	Detail: "a database outside the target workspace uses the notifier",
	Errors: nil,
}

var (
	ErrNotifierNotFound                      = errors.New("notifier not found")
	ErrNotifierNotInWorkspace                = errors.New("notifier does not belong to this workspace")
	ErrInsufficientPermissionsToViewNotifier = errors.New(
		"insufficient permissions to view notifier in this workspace",
	)
)
