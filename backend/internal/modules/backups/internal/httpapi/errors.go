package httpapi

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	errNoActor              = errors.New("the request reached a signed-in operation without an actor")
	errUnknownConfigType    = errors.New("the backup config has no conversion to the contract")
	errUnknownStoredValue   = errors.New("a stored value of the backup config has no value in the contract")
	errUnknownSettingsValue = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "is not a backup method"},
	})
	errUnknownStorageOption = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "storage.type", Detail: "is not a storage option"},
	})
)
