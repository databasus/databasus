package settings

import (
	"fmt"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrNotAdmin = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "only an instance admin can change the instance settings or send a test email",
		Errors: nil,
	}
	ErrEmailNotConfigured = &apierror.Problem{
		Code:   apierror.CodeEmailNotConfigured,
		Detail: "the instance has no mail server configured",
		Errors: nil,
	}
	ErrActorEmailMissing = &apierror.Problem{
		Code:   apierror.CodeAdminEmailMissing,
		Detail: "your user has no valid email address to send a test email to",
		Errors: nil,
	}
	ErrTestEmailRateLimited = &apierror.Problem{
		Code:   apierror.CodeRateLimited,
		Detail: "too many test emails, try again later",
		Errors: nil,
	}
	ErrTwoFactorWithoutMailServer = &apierror.Problem{
		Code:   apierror.CodeEmailNotConfigured,
		Detail: "cannot require two-factor authentication: the instance has no mail server configured",
		Errors: nil,
	}
)

func newTwoFactorAdminEmailMissingProblem(adminEmail string) *apierror.Problem {
	return &apierror.Problem{
		Code: apierror.CodeAdminEmailMissing,
		Detail: fmt.Sprintf(
			"cannot require two-factor authentication: administrator %q has no valid email address", adminEmail,
		),
		Errors: nil,
	}
}
