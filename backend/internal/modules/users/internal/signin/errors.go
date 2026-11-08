package signin

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

// ErrInvalidSession is every reason a session token does not sign anybody in. The access check
// answers it with 401 unauthenticated.
var ErrInvalidSession = errors.New("the session is not valid")

var (
	ErrRateLimited = &apierror.Problem{
		Code:   apierror.CodeRateLimited,
		Detail: "too many attempts, try again later",
		Errors: nil,
	}
	ErrEmailNotFound = &apierror.Problem{
		Code:   apierror.CodeEmailNotFound,
		Detail: "email not found",
		Errors: nil,
	}
	ErrPasswordIncorrect = &apierror.Problem{
		Code:   apierror.CodePasswordIncorrect,
		Detail: "the password is incorrect",
		Errors: nil,
	}
	ErrRegistrationIncomplete = &apierror.Problem{
		Code:   apierror.CodeRegistrationIncomplete,
		Detail: "incomplete registration",
		Errors: nil,
	}
	ErrPasswordNotSet = &apierror.Problem{
		Code:   apierror.CodePasswordNotSet,
		Detail: "password not set",
		Errors: nil,
	}
	ErrPasswordNotSetGoogle = &apierror.Problem{
		Code:   apierror.CodePasswordNotSetGoogle,
		Detail: "password not set; use google",
		Errors: nil,
	}
	ErrPasswordNotSetGitHub = &apierror.Problem{
		Code:   apierror.CodePasswordNotSetGitHub,
		Detail: "password not set; use github",
		Errors: nil,
	}
	ErrPasswordNotSetGoogleGitHub = &apierror.Problem{
		Code:   apierror.CodePasswordNotSetGoogleGitHub,
		Detail: "password not set; use google or github",
		Errors: nil,
	}
	ErrUserDeactivated = &apierror.Problem{
		Code:   apierror.CodeUserDeactivated,
		Detail: "the user is deactivated",
		Errors: nil,
	}
	ErrRegistrationClosed = &apierror.Problem{
		Code:   apierror.CodeRegistrationClosed,
		Detail: "the instance does not allow registration without an invitation",
		Errors: nil,
	}
	// ErrSignInCodeNotSent admits nobody through password sign-in while the second factor is on and
	// the instance cannot deliver the code.
	ErrSignInCodeNotSent = &apierror.Problem{
		Code:   apierror.CodeSignInCodeNotSent,
		Detail: "could not send the sign-in code, please try again later",
		Errors: nil,
	}
	ErrTooManySignInCodes = &apierror.Problem{
		Code:   apierror.CodeTooManySignInCodes,
		Detail: "too many codes requested for this user, please try again later",
		Errors: nil,
	}
)

var (
	ErrPendingSignInNotUsable = &apierror.Problem{
		Code:   apierror.CodePendingSignInNotUsable,
		Detail: "this sign-in can no longer be completed, please sign in again",
		Errors: nil,
	}
	ErrSignInCodeIncorrect = &apierror.Problem{
		Code:   apierror.CodeSignInCodeIncorrect,
		Detail: "the code is incorrect",
		Errors: nil,
	}
	ErrSignInCodeResentTooSoon = &apierror.Problem{
		Code:   apierror.CodeSignInCodeResentTooSoon,
		Detail: "a code was sent less than a minute ago, please wait before requesting another",
		Errors: nil,
	}
)

var (
	ErrOAuthNotConfigured = &apierror.Problem{
		Code:   apierror.CodeOAuthNotConfigured,
		Detail: "the instance does not offer sign-in with this provider",
		Errors: nil,
	}
	ErrOAuthFailed = &apierror.Problem{
		Code:   apierror.CodeOAuthFailed,
		Detail: "the provider refused the sign-in or gave no verified email address",
		Errors: nil,
	}
)
