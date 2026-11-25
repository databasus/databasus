package signin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/settings"
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

	pendingSignIn, isSecondFactorRequired, err := s.startSecondFactor(ctx, user)
	if err != nil {
		return SignInOutcome{}, err
	}

	if isSecondFactorRequired {
		return SignInOutcome{Session: nil, PendingSignIn: &pendingSignIn}, nil
	}

	session, err := s.completeSignIn(ctx, user)
	if err != nil {
		return SignInOutcome{}, err
	}

	return SignInOutcome{Session: &session, PendingSignIn: nil}, nil
}

// startSecondFactor chooses the second factor of the account: a connected app under every
// requirement, then the emailed code or a new app setup as the instance requires. It reports false
// when the instance requires none.
func (s *Service) startSecondFactor(
	ctx context.Context,
	user accounts.UserRecord,
) (PendingSignInView, bool, error) {
	app, hasApp, err := s.authenticator.GetConnectedApp(ctx, user.ID)
	if err != nil {
		return PendingSignInView{}, false, err
	}

	if hasApp {
		pendingSignIn, err := s.startAuthenticatorCodeSignIn(ctx, user, app.ID)

		return pendingSignIn, true, err
	}

	requirement, err := s.settings.GetTwoFactorRequirement(ctx)
	if err != nil {
		return PendingSignInView{}, false, err
	}

	switch requirement {
	case settings.TwoFactorRequirementEmail:
		pendingSignIn, err := s.startTwoFactorSignIn(ctx, user)

		return pendingSignIn, true, err
	case settings.TwoFactorRequirementAuthenticatorApp:
		pendingSignIn, err := s.startAuthenticatorSetupSignIn(ctx, user)

		return pendingSignIn, true, err
	case settings.TwoFactorRequirementOff:
		return PendingSignInView{}, false, nil
	}

	return PendingSignInView{}, false, fmt.Errorf("unknown two-factor requirement %q", requirement)
}

func (s *Service) verifyPassword(ctx context.Context, request SignInRequest) (accounts.UserRecord, error) {
	user, isFound, err := s.accounts.GetRecordByEmail(ctx, request.Email)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	if refusal := s.getPasswordSignInRefusal(user, isFound); refusal != nil {
		_ = bcrypt.CompareHashAndPassword([]byte(unmatchableHash), []byte(request.Password))

		s.logger.WarnContext(
			ctx,
			"sign-in refused",
			slog.String("reason", string(refusal.Code)),
			slog.String("email", request.Email),
		)

		return accounts.UserRecord{}, refusal
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.HashedPassword), []byte(request.Password))
	if err != nil {
		s.logger.WarnContext(ctx, "sign-in refused: incorrect password", slog.String("user_id", user.ID.String()))

		return accounts.UserRecord{}, ErrPasswordIncorrect
	}

	return user, nil
}

func (s *Service) getPasswordSignInRefusal(user accounts.UserRecord, isFound bool) *apierror.Problem {
	switch {
	case !isFound:
		return ErrEmailNotFound
	case user.Status == identity.StatusInvited:
		return ErrRegistrationIncomplete
	case user.HasPassword():
		return nil
	}

	isGoogleAvailable := user.GoogleOAuthID != nil && s.google.IsConfigured()
	isGitHubAvailable := user.GitHubOAuthID != nil && s.gitHub.IsConfigured()

	switch {
	case isGoogleAvailable && isGitHubAvailable:
		return ErrPasswordNotSetGoogleGitHub
	case isGoogleAvailable:
		return ErrPasswordNotSetGoogle
	case isGitHubAvailable:
		return ErrPasswordNotSetGitHub
	default:
		return ErrPasswordNotSet
	}
}
