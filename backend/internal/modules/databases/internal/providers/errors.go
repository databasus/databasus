package providers

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrUnknownDatabaseType   = errors.New("no provider is registered for the database type")
	ErrSettingsOfAnotherType = errors.New("the settings belong to a provider of another type")
	// ErrSecretUnreadable marks a stored secret that does not decrypt, which is an internal failure
	// rather than a refusal of the user's server.
	ErrSecretUnreadable = errors.New("a stored secret does not decrypt")
)

// NewInvalidSettingError is the problem of a setting the provider refuses, named by its field in the
// contract.
func NewInvalidSettingError(field, detail string) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: "settings." + field, Detail: detail}})
}
