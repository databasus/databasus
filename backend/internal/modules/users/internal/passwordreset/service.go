package passwordreset

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/turnstile"
)

const (
	resetCodeLifetime = time.Hour

	requestRateLimitScope  = "reset-password"
	requestRateLimit       = 3
	requestRateLimitWindow = time.Hour

	// maxResetCodesPerHour is the allowance kept in the database, which a restart that empties the
	// in-memory limiter does not reset.
	maxResetCodesPerHour int64 = 3
	resetCodeRetention         = time.Hour
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	Accounts     *accounts.Service
	Mailer       mailer.Sender
	RateLimiter  ratelimit.Counter
	Turnstile    *turnstile.Client
	Logger       *slog.Logger
}

type Service struct {
	resetCodes  *resetCodeRepository
	transactor  *database.Transactor
	clock       clock.Source
	accounts    *accounts.Service
	mailer      mailer.Sender
	rateLimiter ratelimit.Counter
	turnstile   *turnstile.Client
	logger      *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		resetCodes:  deps.Repositories.resetCodes,
		transactor:  deps.Transactor,
		clock:       deps.Clock,
		accounts:    deps.Accounts,
		mailer:      deps.Mailer,
		rateLimiter: deps.RateLimiter,
		turnstile:   deps.Turnstile,
		logger:      deps.Logger,
	}
}

type ResetRequest struct {
	Email           string
	TurnstileAnswer *string
}

type ResetCompletion struct {
	Email       string
	Code        string
	NewPassword string
}

// RequestReset answers alike for an active user, a user who is not active and an address without a
// user, and only an active user with a deliverable address gets a code. The rate limit counts every
// address, so its answer tells nothing either. A failure to deliver is logged, not answered, for the
// same reason.
func (s *Service) RequestReset(ctx context.Context, request ResetRequest) error {
	err := s.turnstile.Verify(ctx, request.TurnstileAnswer)
	if err != nil {
		return err
	}

	err = s.checkRateLimit(ctx, request.Email)
	if err != nil {
		return err
	}

	user, isFound, err := s.accounts.GetRecordByEmail(ctx, request.Email)
	if err != nil {
		return err
	}

	switch {
	case !isFound:
		s.logger.InfoContext(ctx, "reset code not sent: no user with this email")

		return nil
	case !user.IsActive():
		s.logger.InfoContext(
			ctx,
			"reset code not sent: the user is not active",
			slog.String("user_id", user.ID.String()),
		)

		return nil
	case !s.mailer.IsConfigured():
		s.logger.ErrorContext(ctx, "reset code not sent: the instance has no mail server",
			slog.String("user_id", user.ID.String()))

		return nil
	case !identity.IsEmailAddress(user.Email):
		s.logger.WarnContext(ctx, "reset code not sent: the user has no deliverable address",
			slog.String("user_id", user.ID.String()))

		return nil
	}

	code, isIssued, err := s.issueResetCode(ctx, user)
	if err != nil || !isIssued {
		return err
	}

	err = s.mailer.SendEmail(ctx, user.Email, resetCodeSubject, resetCodeBody(code))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to send the reset code",
			slog.String("user_id", user.ID.String()), slog.Any("error", err))
	}

	return nil
}

// CompleteReset replaces the password of the user who returns with the newest usable code. A wrong
// code leaves it usable, as before.
func (s *Service) CompleteReset(ctx context.Context, completion ResetCompletion) error {
	if !identity.IsPasswordLongEnough(completion.NewPassword) {
		return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "newPassword",
			Detail: fmt.Sprintf("must be at least %d characters long", identity.MinPasswordLength),
		}})
	}

	hashedPassword, err := identity.HashPassword(completion.NewPassword)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		user, isFound, err := s.accounts.GetRecordByEmail(ctx, completion.Email)
		if err != nil {
			return err
		}

		if !isFound || !user.IsActive() {
			return ErrInvalidResetCode
		}

		resetCode, isFound, err := s.resetCodes.getNewestUsable(ctx, user.ID, s.clock.Now())
		if err != nil {
			return err
		}

		if !isFound || bcrypt.CompareHashAndPassword([]byte(resetCode.HashedCode), []byte(completion.Code)) != nil {
			return ErrInvalidResetCode
		}

		err = s.resetCodes.markUsed(ctx, resetCode.ID)
		if err != nil {
			return err
		}

		return s.accounts.ReplacePassword(ctx, user, hashedPassword, "Password reset via email code")
	})
}

// DeleteResetCodesPastRetention deletes the codes the hourly allowance no longer counts; every one
// of them has expired.
func (s *Service) DeleteResetCodesPastRetention(ctx context.Context, now time.Time) (int64, error) {
	return s.resetCodes.deleteCreatedBefore(ctx, now.Add(-resetCodeRetention))
}

// issueResetCode stores a new code unless the user used up the hourly allowance, which answers
// false.
func (s *Service) issueResetCode(ctx context.Context, user accounts.UserRecord) (string, bool, error) {
	now := s.clock.Now()

	recentCodeCount, err := s.resetCodes.countCreatedSince(ctx, user.ID, now.Add(-time.Hour))
	if err != nil {
		return "", false, err
	}

	if recentCodeCount >= maxResetCodesPerHour {
		s.logger.WarnContext(ctx, "reset code not sent: the hourly allowance is used up",
			slog.String("user_id", user.ID.String()))

		return "", false, nil
	}

	code, err := identity.GenerateSixDigitCode()
	if err != nil {
		return "", false, err
	}

	hashedCode, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", false, fmt.Errorf("hash the reset code: %w", err)
	}

	err = s.resetCodes.insert(ctx, ResetCode{
		ID:         uuid.New(),
		UserID:     user.ID,
		HashedCode: string(hashedCode),
		ExpiresAt:  now.Add(resetCodeLifetime),
		IsUsed:     false,
		CreatedAt:  now,
	})
	if err != nil {
		return "", false, err
	}

	return code, true, nil
}

// checkRateLimit refuses the request when the limiter refuses it or cannot evaluate it, and logs a
// limiter failure without the address.
func (s *Service) checkRateLimit(ctx context.Context, email string) error {
	isAllowed, err := s.rateLimiter.RecordAttemptAndCheckIsAllowed(ctx, ratelimit.Attempt{
		Scope:      requestRateLimitScope,
		Identifier: email,
		Limit:      requestRateLimit,
		Window:     requestRateLimitWindow,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to evaluate a rate limit",
			slog.String("scope", requestRateLimitScope), slog.Any("error", err))

		return ErrRateLimited
	}

	if !isAllowed {
		return ErrRateLimited
	}

	return nil
}
