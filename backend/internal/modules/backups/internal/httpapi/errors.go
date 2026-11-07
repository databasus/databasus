package httpapi

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	errUnknownRestoreTokenSubject = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{{Field: "type", Detail: "must be POINT_IN_TIME or BACKUP"}},
	)
	errUnknownPhysicalValue = errors.New("a stored physical backup value has no value in the contract")
	errUnknownConfigType    = errors.New("the backup config has no conversion to the contract")
	errUnknownStoredValue   = errors.New("a stored value of the backup config has no value in the contract")
	errUnknownSettingsValue = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "is not a backup method"},
	})
	errUnknownStorageOption = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "storage.type", Detail: "is not a storage option"},
	})
	errUnknownRestoreTarget = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "target.type", Detail: "is not a database type with logical backups"},
	})
)
