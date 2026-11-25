package signin

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
)

const (
	signInCodeVerificationScope  = "signin-code-verification"
	signInCodeVerificationLimit  = 10
	signInCodeVerificationWindow = time.Minute

	signInCodeResendScope  = "signin-code-resend"
	signInCodeResendWindow = time.Minute
)

// VerifySignInCode completes a pending sign-in with its emailed code, or with an app code or a
// recovery code when it waits for the app. Each guess first claims one of the five attempts, so
// simultaneous guesses never check more than five codes, and the correct code spends the pending
// sign-in, so two requests carrying it cannot both sign in.
func (s *Service) VerifySignInCode(
	ctx context.Context,
	pendingSignInID uuid.UUID,
	code string,
	turnstileAnswer *string,
) (Session, error) {
	err := s.turnstile.Verify(ctx, turnstileAnswer)
	if err != nil {
		return Session{}, err
	}

	err = s.checkRateLimit(ctx, ratelimit.Attempt{
		Scope:      signInCodeVerificationScope,
		Identifier: pendingSignInID.String(),
		Limit:      signInCodeVerificationLimit,
		Window:     signInCodeVerificationWindow,
	})
	if err != nil {
		return Session{}, err
	}

	pendingSignIn, user, err := s.loadUsablePendingSignIn(ctx, pendingSignInID)
	if err != nil {
		return Session{}, err
	}

	switch pendingSignIn.Kind {
	case PendingSignInKindEmailCode:
		return s.verifyEmailedCode(ctx, pendingSignIn, user, code)
	case PendingSignInKindAuthenticatorCode:
		return s.verifyAuthenticatorCode(ctx, pendingSignIn, user, code)
	case PendingSignInKindAuthenticatorSetup:
		return Session{}, ErrPendingSignInNotUsable
	}

	return Session{}, ErrPendingSignInNotUsable
}

func (s *Service) verifyEmailedCode(
	ctx context.Context,
	pendingSignIn PendingSignIn,
	user accounts.UserRecord,
	code string,
) (Session, error) {
	attemptCount, isAttemptClaimed, err := s.pendingSignIns.claimAttempt(ctx, pendingSignIn.ID, s.clock.Now())
	if err != nil {
		return Session{}, err
	}

	if !isAttemptClaimed {
		return Session{}, ErrPendingSignInNotUsable
	}

	if pendingSignIn.HashedCode == nil ||
		bcrypt.CompareHashAndPassword([]byte(*pendingSignIn.HashedCode), []byte(code)) != nil {
		return Session{}, s.refuseWrongSignInCode(ctx, attemptCount, user)
	}

	return s.spendAndCompleteSignIn(ctx, pendingSignIn, user)
}

func (s *Service) spendAndCompleteSignIn(
	ctx context.Context,
	pendingSignIn PendingSignIn,
	user accounts.UserRecord,
) (Session, error) {
	var session Session

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		isSpent, err := s.pendingSignIns.spend(ctx, pendingSignIn.ID, s.clock.Now())
		if err != nil {
			return err
		}

		if !isSpent {
			return ErrPendingSignInNotUsable
		}

		session, err = s.completeSignIn(ctx, user)

		return err
	})

	return session, err
}

// ResendSignInCode sends a new code and answers the pending sign-in it names. The previous code
// stops working only once the new one is on its way, so a refused resend leaves the user with the
// code they already have. A pending sign-in that waits for an app sends no mail and has nothing to
// resend.
func (s *Service) ResendSignInCode(
	ctx context.Context,
	pendingSignInID uuid.UUID,
	turnstileAnswer *string,
) (PendingSignInView, error) {
	err := s.turnstile.Verify(ctx, turnstileAnswer)
	if err != nil {
		return PendingSignInView{}, err
	}

	pendingSignIn, user, err := s.loadUsablePendingSignIn(ctx, pendingSignInID)
	if err != nil {
		return PendingSignInView{}, err
	}

	if pendingSignIn.Kind != PendingSignInKindEmailCode {
		return PendingSignInView{}, ErrPendingSignInNotResendable
	}

	isAllowed, err := s.rateLimiter.RecordAttemptAndCheckIsAllowed(ctx, ratelimit.Attempt{
		Scope:      signInCodeResendScope,
		Identifier: user.ID.String(),
		Limit:      1,
		Window:     signInCodeResendWindow,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to evaluate a rate limit",
			slog.String("scope", signInCodeResendScope), slog.Any("error", err))

		return PendingSignInView{}, ErrSignInCodeResentTooSoon
	}

	if !isAllowed {
		return PendingSignInView{}, ErrSignInCodeResentTooSoon
	}

	newPendingSignIn, err := s.issueSignInCode(ctx, user, false)
	if err != nil {
		return PendingSignInView{}, err
	}

	err = s.pendingSignIns.markUsed(ctx, pendingSignIn.ID)
	if err != nil {
		return PendingSignInView{}, err
	}

	return newPendingSignIn, nil
}

// DeletePendingSignInsPastRetention deletes the pending sign-ins the hourly cap no longer counts
// and returns how many it deleted.
func (s *Service) DeletePendingSignInsPastRetention(ctx context.Context, now time.Time) (int64, error) {
	return s.pendingSignIns.deleteCreatedBefore(ctx, now.Add(-pendingSignInRetention))
}

// loadUsablePendingSignIn re-checks what the password step checked, because a pending sign-in
// proves that a password was correct minutes ago, not that the user may be let in now: the user
// still exists, is active and has not changed the password. The second-factor setting is not
// among the checks, so switching it off or changing its method does not strand somebody holding a
// code or a setup.
func (s *Service) loadUsablePendingSignIn(
	ctx context.Context,
	pendingSignInID uuid.UUID,
) (PendingSignIn, accounts.UserRecord, error) {
	pendingSignIn, isFound, err := s.pendingSignIns.getByID(ctx, pendingSignInID)
	if err != nil {
		return PendingSignIn{}, accounts.UserRecord{}, err
	}

	if !isFound || !pendingSignIn.IsUsable(s.clock.Now()) {
		return PendingSignIn{}, accounts.UserRecord{}, ErrPendingSignInNotUsable
	}

	user, isFound, err := s.accounts.GetRecord(ctx, pendingSignIn.UserID)
	if err != nil {
		return PendingSignIn{}, accounts.UserRecord{}, err
	}

	isPasswordUnchanged := user.PasswordVersion == pendingSignIn.PasswordVersion

	if !isFound || !user.IsActive() || !isPasswordUnchanged {
		return PendingSignIn{}, accounts.UserRecord{}, ErrPendingSignInNotUsable
	}

	return pendingSignIn, user, nil
}

func (s *Service) refuseWrongSignInCode(ctx context.Context, attemptCount int, user accounts.UserRecord) error {
	if attemptCount < maxCodeAttempts {
		return ErrSignInCodeIncorrect
	}

	err := s.recordAuditEvent(ctx, user, "Two-factor sign-in abandoned after too many incorrect codes: "+user.Email)
	if err != nil {
		return err
	}

	return ErrPendingSignInNotUsable
}
