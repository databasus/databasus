package configs

import "databasus-backend/internal/lib/apierror"

var ErrDatabaseNotFound = &apierror.Problem{Code: apierror.CodeNotFound, Detail: "database not found", Errors: nil}
