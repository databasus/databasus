package turnstile

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrVerificationFailed = &apierror.Problem{
		Code:   apierror.CodeHumanVerificationFailed,
		Detail: "the human-verification challenge was not answered or the answer was rejected",
		Errors: nil,
	}
	ErrUnexpectedAnswer = errors.New("cloudflare turnstile answered unexpectedly")
)
