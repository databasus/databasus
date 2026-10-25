package signin

import (
	"context"
	"fmt"
	"time"

	"databasus-backend/internal/modules/users/internal/identity"
)

// Authenticate returns the actor of a session token. A token whose signature, expiry, user, user
// status or password time does not hold answers ErrInvalidSession.
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

	if !user.PasswordCreationTime.Truncate(time.Second).Equal(claims.passwordCreationTime.Truncate(time.Second)) {
		return identity.Actor{}, fmt.Errorf("%w: the password changed after the token was issued", ErrInvalidSession)
	}

	return user.ToActor(), nil
}
