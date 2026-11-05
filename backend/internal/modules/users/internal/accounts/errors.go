package accounts

import (
	"errors"
	"fmt"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/users/internal/identity"
)

var (
	ErrUserNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the user does not exist",
		Errors: nil,
	}
	ErrEmailTaken = &apierror.Problem{
		Code:   apierror.CodeEmailTaken,
		Detail: "a user with this email address already exists",
		Errors: nil,
	}
	ErrInvitationNotAllowed = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "the instance does not allow this user to invite users",
		Errors: nil,
	}
	ErrNotOwnUser = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "a user can change only their own profile and password",
		Errors: nil,
	}
	ErrNotAdmin = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "the operation is for instance admins",
		Errors: nil,
	}
	ErrCannotDeactivateYourself = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "an administrator cannot deactivate themselves",
		Errors: nil,
	}
	ErrCannotChangeOwnRole = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "an administrator cannot change their own role",
		Errors: nil,
	}
	ErrBootstrapAdminProtected = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "the bootstrap administrator cannot be deactivated or lose the admin role",
		Errors: nil,
	}
	ErrOnlyBootstrapAdminManagesAdmins = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "only the bootstrap administrator grants, revokes, activates and deactivates instance admins",
		Errors: nil,
	}
	ErrInvalidCursor = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "after", Detail: "is not a cursor of this list"},
	})
	ErrAnotherUser = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "only an instance admin can act on another user",
		Errors: nil,
	}
)

var ErrUnknownOAuthProvider = errors.New("the identity provider is unknown")

var (
	ErrConsoleEmailInvalid     = errors.New("the email is not a valid email address")
	ErrConsoleRoleInvalid      = errors.New("the role must be admin or member")
	ErrConsolePasswordTooShort = fmt.Errorf("the password must be at least %d characters long",
		identity.MinPasswordLength)
	ErrConsoleUserNotFound = errors.New("no user has this email address")
	// ErrConsoleFirstUserMustBeAdmin keeps an instance from getting a member as its first user,
	// which would leave nobody able to grant the admin role.
	ErrConsoleFirstUserMustBeAdmin = errors.New(
		"this instance has no account yet, and its first account administers it: create it with the admin role",
	)
	ErrConsoleUserExistsInAnotherState = errors.New(
		"an account with this email already exists with another role, is deactivated, " +
			"has not completed sign-up or has no password; nothing changed",
	)
)
