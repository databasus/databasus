package database

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrSchemaNewerThanBinary = errors.New("the internal database was migrated by a newer version")
	ErrNoTransaction         = errors.New("an after-commit hook needs a transaction in the context")
	ErrInvalidCursor         = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "after", Detail: "is not a cursor of this list"},
	})
)
