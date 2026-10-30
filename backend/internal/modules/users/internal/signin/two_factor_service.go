package signin

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
)

const (
	signInCodeLifetime = 10 * time.Minute
	// pendingSignInRetention is the hour the hourly cap counts over, and a pending sign-in is kept
	// exactly as long.
	pendingSignInRetention       = time.Hour
	maxSignInCodesPerHour  int64 = 5
)

// startTwoFactorSignIn runs only after the password was accepted. A live pending sign-in started
// with the same password is handed back unchanged, so reaching the code screen again neither sends
// a second message nor spends the hourly allowance.
func (s *Service) startTwoFactorSignIn(ctx context.Context, user accounts.UserRecord) (PendingSignInView, error) {
	return s.issueSignInCode(ctx, user, true)
}

// issueSignInCode creates nothing before the instance can deliver, and a failed send leaves the
// counted row behind unusable, so the caller never sees a code screen for a message that will not
// arrive.
func (s *Service) issueSignInCode(
	ctx context.Context,
	user accounts.UserRecord,
	shouldReuseLiveCode bool,
) (PendingSignInView, error) {
	if !s.mailer.IsConfigured() {
		s.logger.ErrorContext(ctx, "sign-in code not sent: the instance has no mail server",
			slog.String("user_id", user.ID.String()))

		return PendingSignInView{}, ErrSignInCodeNotSent
	}

	code, err := identity.GenerateSixDigitCode()
	if err != nil {
		return PendingSignInView{}, err
	}

	hashedCode, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return PendingSignInView{}, err
	}

	now := s.clock.Now()
	newPendingSignIn := PendingSignIn{
		ID:                 uuid.New(),
		UserID:             user.ID,
		HashedCode:         string(hashedCode),
		PasswordVersion:    user.PasswordVersion,
		ExpiresAt:          now.Add(signInCodeLifetime),
		IsUsed:             false,
		FailedAttemptCount: 0,
		CreatedAt:          now,
	}

	livePendingSignIn, isLive, err := s.createPendingSignIn(ctx, newPendingSignIn, shouldReuseLiveCode)
	if err != nil {
		return PendingSignInView{}, err
	}

	if isLive {
		return PendingSignInView{PendingSignInID: livePendingSignIn.ID, Email: user.Email}, nil
	}

	err = s.mailer.SendEmail(ctx, user.Email, signInCodeSubject, signInCodeBody(code))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to send the sign-in code",
			slog.String("user_id", user.ID.String()), slog.Any("error", err))

		markError := s.pendingSignIns.markUsed(ctx, newPendingSignIn.ID)
		if markError != nil {
			s.logger.ErrorContext(ctx, "failed to discard an unsent pending sign-in",
				slog.String("user_id", user.ID.String()), slog.Any("error", markError))
		}

		return PendingSignInView{}, ErrSignInCodeNotSent
	}

	return PendingSignInView{PendingSignInID: newPendingSignIn.ID, Email: user.Email}, nil
}

// createPendingSignIn inserts the pending sign-in within the hourly cap, or hands back the live
// one that the same password started when shouldReuseLiveCode is set. It reports true with the
// live one.
func (s *Service) createPendingSignIn(
	ctx context.Context,
	newPendingSignIn PendingSignIn,
	shouldReuseLiveCode bool,
) (PendingSignIn, bool, error) {
	var livePendingSignIn PendingSignIn

	isLive := false

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.pendingSignIns.lockUser(ctx, newPendingSignIn.UserID)
		if err != nil {
			return err
		}

		if shouldReuseLiveCode {
			livePendingSignIn, isLive, err = s.pendingSignIns.getLive(ctx, livePendingSignInFilter{
				userID:          newPendingSignIn.UserID,
				passwordVersion: newPendingSignIn.PasswordVersion,
				now:             newPendingSignIn.CreatedAt,
			})
			if err != nil || isLive {
				return err
			}
		}

		recentCount, err := s.pendingSignIns.countCreatedSince(
			ctx, newPendingSignIn.UserID, newPendingSignIn.CreatedAt.Add(-pendingSignInRetention),
		)
		if err != nil {
			return err
		}

		if recentCount >= maxSignInCodesPerHour {
			return ErrTooManySignInCodes
		}

		return s.pendingSignIns.insert(ctx, newPendingSignIn)
	})
	if err != nil {
		return PendingSignIn{}, false, err
	}

	return livePendingSignIn, isLive, nil
}
