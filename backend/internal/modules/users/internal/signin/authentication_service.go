package signin

import (
	"context"
	"fmt"

	"databasus-backend/internal/modules/users/internal/identity"
)

func (s *Service) Authenticate(ctx context.Context, token string) (identity.Actor, error) {
	claims, err := s.sessions.read(token)
	if err != nil {
		return identity.Actor{}, err
	}

	user, isFound, err := s.accounts.GetRecord(ctx, claims.userID)
	if err != nil {
		return identity.Actor{}, err
	}

	if !isFound {
		return identity.Actor{}, fmt.Errorf("%w: the user does not exist", ErrInvalidSession)
	}

	if !user.IsActive() {
		return identity.Actor{}, fmt.Errorf("%w: the user is not active", ErrInvalidSession)
	}

	if user.PasswordVersion != claims.passwordVersion {
		return identity.Actor{}, fmt.Errorf("%w: the password changed after the token was issued", ErrInvalidSession)
	}

	return user.ToActor(), nil
}
