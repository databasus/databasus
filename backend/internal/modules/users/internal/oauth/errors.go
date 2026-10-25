package oauth

import "errors"

var (
	ErrUnexpectedAnswer = errors.New("the identity provider answered unexpectedly")
	ErrNoVerifiedEmail  = errors.New("the provider's account has no verified email address")
)
