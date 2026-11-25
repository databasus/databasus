package accounts

import (
	"databasus-backend/internal/modules/users/internal/authenticator"
	"databasus-backend/internal/modules/users/internal/identity"
)

type UserWithAuthenticatorApp struct {
	User             identity.User
	AuthenticatorApp *authenticator.AuthenticatorAppState
}
