package providers

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrUnknownStorageType    = errors.New("no provider is registered for the storage type")
	ErrSettingsOfAnotherType = errors.New("the settings belong to a provider of another type")
)

// NewInvalidSettingError is the problem of a setting the provider refuses, named by its field in the
// contract.
func NewInvalidSettingError(field, detail string) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: "settings." + field, Detail: detail}})
}

// NewFixedSettingError refuses a change of a setting that names where every stored file lives.
func NewFixedSettingError(field string) error {
	return NewInvalidSettingError(field, "is fixed when the storage is created")
}
