package signin

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
)

const (
	signInRateLimitScope  = "signin"
	signInRateLimit       = 10
	signInRateLimitWindow = time.Minute
)

// unmatchableHash is the bcrypt hash of a random secret nobody knows. A sign-in for an address
// without a usable password is compared against it, so it takes as long as a wrong password.
// This dummy bcrypt hash equalizes missing-user timing and authenticates no account.
// nosemgrep: generic.secrets.security.detected-bcrypt-hash.detected-bcrypt-hash
const unmatchableHash = "$2a$10$J1I5VBn.teMq6ueky2hqy.ECHJ6JOS4AcT6kjk62wWxRONkvkj7n6"

type SignInRequest struct {
	Email           string
	Password        string
	TurnstileAnswer *string
}

// SignIn refuses alike until the password is verified: an unknown address, a user who has not
// finished sign-up, a user without a password and a wrong password all get invalid_credentials
// after the same bcrypt comparison. Only then may the answer name a deactivated user. The log keeps
// each reason, because failed sign-ins are the signal that someone guesses credentials.
func (s *Service) SignIn(ctx context.Context, request SignInRequest) (SignInOutcome, error) {
	err := s.turnstile.Verify(ctx, request.TurnstileAnswer)
	if err != nil {
		return SignInOutcome{}, err
	}

	err = s.checkRateLimit(ctx, ratelimit.Attempt{
		Scope:      signInRateLimitScope,
		Identifier: request.Email,
		Limit:      signInRateLimit,
		Window:     signInRateLimitWindow,
	})
	if err != nil {
		return SignInOutcome{}, err
	}

	user, err := s.verifyPassword(ctx, request)
	if err != nil {
		return SignInOutcome{}, err
	}

	if !user.IsActive() {
		s.logger.WarnContext(ctx, "sign-in refused: the user is deactivated", slog.String("user_id", user.ID.String()))

		return SignInOutcome{}, ErrUserDeactivated
	}

	instanceSettings, err := s.settings.GetSettings(ctx)
	if err != nil {
		return SignInOutcome{}, err
	}

	if instanceSettings.IsTwoFactorAuthRequired {
		pendingSignIn, err := s.startTwoFactorSignIn(ctx, user)
		if err != nil {
			return SignInOutcome{}, err
		}

		return SignInOutcome{Session: nil, PendingSignIn: &pendingSignIn}, nil
	}

	session, err := s.completeSignIn(ctx, user)
	if err != nil {
		return SignInOutcome{}, err
	}

	return SignInOutcome{Session: &session, PendingSignIn: nil}, nil
}

func (s *Service) verifyPassword(ctx context.Context, request SignInRequest) (accounts.UserRecord, error) {
	user, isFound, err := s.accounts.GetRecordByEmail(ctx, request.Email)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	refusal := ""

	switch {
	case !isFound:
		refusal = "no user with this email"
	case user.Status == identity.StatusInvited:
		refusal = "the user has not finished sign-up"
	case !user.HasPassword():
		refusal = "the user has no password"
	}

	if refusal != "" {
		_ = bcrypt.CompareHashAndPassword([]byte(unmatchableHash), []byte(request.Password))

		s.logger.WarnContext(
			ctx,
			"sign-in refused",
			slog.String("reason", refusal),
			slog.String("email", request.Email),
		)

		return accounts.UserRecord{}, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.HashedPassword), []byte(request.Password))
	if err != nil {
		s.logger.WarnContext(ctx, "sign-in refused: incorrect password", slog.String("user_id", user.ID.String()))

		return accounts.UserRecord{}, ErrInvalidCredentials
	}

	return user, nil
}
