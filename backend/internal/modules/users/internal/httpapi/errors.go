package httpapi

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var errSetupMissing = errors.New("a pending sign-in that waits for a setup came without its secret")

var (
	errUnauthenticated = &apierror.Problem{
		Code:   apierror.CodeUnauthenticated,
		Detail: "the request has no valid session",
		Errors: nil,
	}
	errNotAdmin = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "the operation is for instance admins",
		Errors: nil,
	}
)
