package identityproof

import "databasus-backend/internal/lib/apierror"

var (
	ErrProofInvalid = &apierror.Problem{
		Code:   apierror.CodeIdentityProofInvalid,
		Detail: "the password or the confirmation code is wrong, expired, used or meant for another operation",
		Errors: nil,
	}
	ErrProofUnavailable = &apierror.Problem{
		Code:   apierror.CodeIdentityProofUnavailable,
		Detail: "no mail server or usable address can receive a confirmation code",
		Errors: nil,
	}
	ErrRateLimited = &apierror.Problem{
		Code:   apierror.CodeRateLimited,
		Detail: "too many identity confirmations; wait a minute",
		Errors: nil,
	}
	errNoSession = &apierror.Problem{
		Code:   apierror.CodeUnauthenticated,
		Detail: "the request has no session",
		Errors: nil,
	}
)
