package access

import "databasus-backend/internal/lib/apierror"

var (
	// ErrWorkspaceNotFound is also the answer for a workspace the actor is not a member of, so a user
	// outside a team cannot learn its workspace IDs.
	ErrWorkspaceNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the workspace does not exist",
		Errors: nil,
	}
	ErrActionNotAllowed = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "the workspace role does not allow the action",
		Errors: nil,
	}
)
