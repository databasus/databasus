package httpapi

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var errNoActor = errors.New("the request reached a signed-in operation without an actor")

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
