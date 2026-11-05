package workspace

import (
	"databasus-backend/internal/lib/apierror"
)

var (
	ErrWorkspaceNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the workspace does not exist",
		Errors: nil,
	}
	ErrCreationNotAllowed = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "the instance does not allow this user to create workspaces",
		Errors: nil,
	}
	ErrWorkspaceNotEmpty = &apierror.Problem{
		Code:   apierror.CodeWorkspaceNotEmpty,
		Detail: "the workspace still has content that must be deleted first",
		Errors: nil,
	}
	ErrInvalidName = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "name", Detail: "must be between 1 and 255 characters long"},
	})
	ErrActorNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the signed-in user does not exist",
		Errors: nil,
	}
)
