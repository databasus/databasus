package config

import "errors"

var (
	ErrRequiredVariableNotSet = errors.New("required environment variable is not set")
	ErrInvalidVariable        = errors.New("environment variable has an invalid value")
)
