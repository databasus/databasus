package configs

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrDatabaseNotFound = &apierror.Problem{Code: apierror.CodeNotFound, Detail: "database not found", Errors: nil}
	ErrConfigMissing    = errors.New("healthcheck config missing")
)
