package notifier

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrNotifierNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the notifier does not exist",
		Errors: nil,
	}
	ErrNotifierInUse = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a database uses the notifier",
		Errors: nil,
	}
	ErrInvalidName = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "name", Detail: "must be between 1 and 255 characters long"},
	})
	ErrTypeChanged = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "must be the type the notifier was created with"},
	})
	ErrActorNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the signed-in user does not exist",
		Errors: nil,
	}
	errSettingsMissing = errors.New("the notifier has no row in the table of its provider")
)
