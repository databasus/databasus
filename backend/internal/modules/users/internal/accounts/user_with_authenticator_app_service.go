package accounts

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users/internal/identity"
)

func (s *Service) withAuthenticatorApp(ctx context.Context, user identity.User) (UserWithAuthenticatorApp, error) {
	users, err := s.withAuthenticatorApps(ctx, []identity.User{user})
	if err != nil {
		return UserWithAuthenticatorApp{}, err
	}

	return users[0], nil
}

func (s *Service) withAuthenticatorApps(
	ctx context.Context,
	users []identity.User,
) ([]UserWithAuthenticatorApp, error) {
	userIDs := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	appStates, err := s.authenticator.GetAppStates(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	usersWithApps := make([]UserWithAuthenticatorApp, 0, len(users))
	for _, user := range users {
		userWithApp := UserWithAuthenticatorApp{User: user, AuthenticatorApp: nil}

		if appState, hasApp := appStates[user.ID]; hasApp {
			userWithApp.AuthenticatorApp = &appState
		}

		usersWithApps = append(usersWithApps, userWithApp)
	}

	return usersWithApps, nil
}
