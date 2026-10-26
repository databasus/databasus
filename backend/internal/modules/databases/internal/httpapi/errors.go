package httpapi

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	errNoActor              = errors.New("the request reached a signed-in operation without an actor")
	errUnknownSettingsType  = errors.New("the settings have no conversion to the contract")
	errUnknownSettingsValue = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "is not a database type"},
	})
)
