package httpapi

import (
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/authenticator"
	"databasus-backend/internal/modules/users/internal/passwordreset"
	"databasus-backend/internal/modules/users/internal/settings"
	"databasus-backend/internal/modules/users/internal/signin"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	SignIn        *signin.Service
	Accounts      *accounts.Service
	Authenticator *authenticator.Service
	Settings      *settings.Service
	PasswordReset *passwordreset.Service
}

type Handler struct {
	signIn        *signin.Service
	accounts      *accounts.Service
	authenticator *authenticator.Service
	settings      *settings.Service
	passwordReset *passwordreset.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		signIn:        deps.SignIn,
		accounts:      deps.Accounts,
		authenticator: deps.Authenticator,
		settings:      deps.Settings,
		passwordReset: deps.PasswordReset,
	}
}

func toUserBody(userWithApp accounts.UserWithAuthenticatorApp) User {
	user := userWithApp.User
	body := User{
		Id:                          user.ID,
		Email:                       user.Email,
		Name:                        user.Name,
		Role:                        UserRole(user.Role),
		Status:                      UserStatus(user.Status),
		IsRootAdmin:                 user.IsRootAdmin,
		CreatedAt:                   user.CreatedAt,
		AuthenticatorAppConnectedAt: nil,
		UnusedRecoveryCodeCount:     nil,
	}

	if userWithApp.AuthenticatorApp != nil {
		body.AuthenticatorAppConnectedAt = &userWithApp.AuthenticatorApp.ConnectedAt
		body.UnusedRecoveryCodeCount = &userWithApp.AuthenticatorApp.UnusedRecoveryCodeCount
	}

	return body
}

// getSecret reads a writeOnly field, which the generated model makes optional.
func getSecret(secret *string) string {
	if secret == nil {
		return ""
	}

	return *secret
}
