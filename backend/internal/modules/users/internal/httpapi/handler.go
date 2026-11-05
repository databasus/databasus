package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/passwordreset"
	"databasus-backend/internal/modules/users/internal/settings"
	"databasus-backend/internal/modules/users/internal/signin"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	SignIn        *signin.Service
	Accounts      *accounts.Service
	Settings      *settings.Service
	PasswordReset *passwordreset.Service
}

type Handler struct {
	signIn        *signin.Service
	accounts      *accounts.Service
	settings      *settings.Service
	passwordReset *passwordreset.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		signIn:        deps.SignIn,
		accounts:      deps.Accounts,
		settings:      deps.Settings,
		passwordReset: deps.PasswordReset,
	}
}

func getActor(ctx context.Context) (identity.Actor, error) {
	actor, isFound := identity.GetActor(ctx)
	if !isFound {
		return identity.Actor{}, errNoActor
	}

	return actor, nil
}

func toUserBody(user identity.User) User {
	return User{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      UserRole(user.Role),
		Status:    UserStatus(user.Status),
		CreatedAt: user.CreatedAt,
	}
}

// getSecret reads a writeOnly field, which the generated model makes optional.
func getSecret(secret *string) string {
	if secret == nil {
		return ""
	}

	return *secret
}
