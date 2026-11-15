package httpapi

import "databasus-backend/internal/lib/apierror"

var errNoActor = &apierror.Problem{Code: apierror.CodeUnauthenticated, Detail: "no actor in request", Errors: nil}
