package database

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrSchemaNewerThanBinary = errors.New("the internal database was migrated by a newer version")
	ErrNoTransaction         = errors.New("an after-commit hook needs a transaction in the context")
	ErrSchemaIncompatible    = errors.New("the snapshot's schema does not match this release")
	ErrMigrationNameInvalid  = errors.New("a migration file name does not start with its version")
	ErrInvalidCursor         = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "after", Detail: "is not a cursor of this list"},
	})
)
