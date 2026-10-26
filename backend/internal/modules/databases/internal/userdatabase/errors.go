package userdatabase

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
	ErrInvalidName = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "name", Detail: "must be between 1 and 255 characters long"},
	})
	ErrTypeChanged = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "must be the type the database was created with"},
	})
	// ErrWorkspaceMissing refuses a request body without its required workspaceId, which the server
	// decodes as the nil UUID because it does not validate request bodies against the contract.
	ErrWorkspaceMissing = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "workspaceId", Detail: "is required"},
	})
	ErrNotifierNotInWorkspace = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "notifierIds", Detail: "must name notifiers of the database's workspace"},
	})
	// ErrWorkspaceHasDatabases refuses a workspace deletion through the workspaces module, which
	// answers it as it is.
	ErrWorkspaceHasDatabases = &apierror.Problem{
		Code:   apierror.CodeWorkspaceNotEmpty,
		Detail: "the workspace still has databases, which must be deleted first",
		Errors: nil,
	}
	// ErrNotifierUsedOutsideTargetWorkspace refuses a notifier transfer through the notifiers module,
	// which answers it as it is.
	ErrNotifierUsedOutsideTargetWorkspace = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a database outside the target workspace uses the notifier",
		Errors: nil,
	}
	ErrActorNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the signed-in user does not exist",
		Errors: nil,
	}
	errSettingsMissing = errors.New("the database has no row in the table of its provider")
)

// NewConnectionFailedError carries the reason the user's server gave. It comes from the user's own
// server, and the engines never send our secrets in a query, so it reveals nothing of ours.
func NewConnectionFailedError(connectionErr error) error {
	return &apierror.Problem{
		Code:   apierror.CodeDatabaseConnectionFailed,
		Detail: connectionErr.Error(),
		Errors: nil,
	}
}
