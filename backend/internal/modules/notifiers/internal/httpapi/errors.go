package httpapi

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	errUnknownSettingsType  = errors.New("the settings have no conversion to the contract")
	errUnknownSettingsValue = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "is not a notifier type"},
	})
)
