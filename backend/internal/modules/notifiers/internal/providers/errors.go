package providers

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrUnknownNotifierType   = errors.New("no provider is registered for the notifier type")
	ErrSettingsOfAnotherType = errors.New("the settings belong to a provider of another type")
)

// NewInvalidSettingError is the problem of a setting the provider refuses, named by its field in
// the contract.
func NewInvalidSettingError(field, detail string) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: "settings." + field, Detail: detail}})
}
