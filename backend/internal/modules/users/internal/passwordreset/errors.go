package passwordreset

import "databasus-backend/internal/lib/apierror"

var (
	ErrRateLimited = &apierror.Problem{
		Code:   apierror.CodeRateLimited,
		Detail: "too many password reset requests for this address, try again later",
		Errors: nil,
	}
	// ErrInvalidResetCode answers alike a wrong, used, expired, exhausted or older code and an address
	// without a user, so the answer tells nothing about the accounts.
	ErrInvalidResetCode = &apierror.Problem{
		Code:   apierror.CodeInvalidResetCode,
		Detail: "the reset code is not valid",
		Errors: nil,
	}
)
