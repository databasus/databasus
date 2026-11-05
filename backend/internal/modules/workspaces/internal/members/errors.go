package members

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrMemberNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the member does not exist",
		Errors: nil,
	}
	ErrWorkspaceNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the workspace does not exist",
		Errors: nil,
	}
	ErrAlreadyMember = &apierror.Problem{
		Code:   apierror.CodeAlreadyMember,
		Detail: "the user is already a member of the workspace",
		Errors: nil,
	}
	ErrCannotChangeOwnRole = &apierror.Problem{
		Code:   apierror.CodeCannotChangeOwnRole,
		Detail: "a member cannot change their own role",
		Errors: nil,
	}
	ErrOwnerRequiresTransfer = &apierror.Problem{
		Code:   apierror.CodeOwnerRequiresTransfer,
		Detail: "the owner changes only through an ownership transfer",
		Errors: nil,
	}
	ErrRoleNotAssignable = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "role", Detail: "must be WORKSPACE_ADMIN, WORKSPACE_MEMBER or WORKSPACE_VIEWER"},
	})
	ErrInvalidCursor = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "after", Detail: "is not a cursor of this list"},
	})
	ErrActorNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the signed-in user does not exist",
		Errors: nil,
	}
)

var errMemberUserMissing = errors.New("a member names a user that does not exist")
