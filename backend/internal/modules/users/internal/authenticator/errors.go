package authenticator

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrTooManyWrongCodes = &apierror.Problem{
		Code:   apierror.CodeTooManyWrongCodes,
		Detail: "too many wrong codes for this user within an hour, please try again later",
		Errors: nil,
	}
	ErrSecondFactorInvalid = &apierror.Problem{
		Code:   apierror.CodeSecondFactorInvalid,
		Detail: "the app code or recovery code is wrong, already used or missing",
		Errors: nil,
	}
	// ErrSetupNotUsable is how the profile hears about a setup that expired, was replaced or was
	// destroyed by wrong codes; sign-in answers its own pending_sign_in_not_usable instead.
	ErrSetupNotUsable = &apierror.Problem{
		Code:   apierror.CodeAuthenticatorSetupNotUsable,
		Detail: "this authenticator setup can no longer be confirmed, please start a new one",
		Errors: nil,
	}
	ErrAppRequired = &apierror.Problem{
		Code:   apierror.CodeAuthenticatorAppRequired,
		Detail: "the instance requires an authenticator app, so it can be replaced but not disconnected",
		Errors: nil,
	}
	ErrNotOwnUser = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "a user can change only their own authenticator app",
		Errors: nil,
	}
)

var ErrConsoleUserNotFound = errors.New("no user has this email address")
