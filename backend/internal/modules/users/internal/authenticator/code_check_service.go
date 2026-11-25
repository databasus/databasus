package authenticator

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/totp"
	"databasus-backend/internal/modules/users/internal/identity"
)

// CheckCode accepts a current app code or an unused recovery code of the user once, and reports
// false for anything else. It is the only check of app and recovery codes, at sign-in, in the
// profile and in confirmations, so the per-account limit counts all of them. It answers
// too_many_wrong_codes without checking once ten wrong codes fall within the last hour.
//
// It commits on its own, so it must not run inside the caller's transaction: a wrong code is
// recorded even though the caller's operation then fails, and an accepted code stays spent.
func (s *Service) CheckCode(ctx context.Context, user identity.User, code string) (bool, error) {
	isAccepted := false

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.apps.lockUser(ctx, user.ID)
		if err != nil {
			return err
		}

		now := s.clock.Now()

		failureCount, err := s.failures.countSince(ctx, user.ID, now.Add(-wrongCodeWindow))
		if err != nil {
			return err
		}

		if failureCount >= maxWrongCodesPerHour {
			s.logger.WarnContext(ctx, "second factor refused: too many wrong codes",
				slog.String("user_id", user.ID.String()))

			return ErrTooManyWrongCodes
		}

		app, hasApp, err := s.apps.getByUserID(ctx, user.ID)
		if err != nil || !hasApp {
			return err
		}

		isAccepted, err = s.spendCode(ctx, user, app, code)
		if err != nil || isAccepted {
			return err
		}

		s.logger.WarnContext(ctx, "second factor refused: wrong code", slog.String("user_id", user.ID.String()))

		return s.failures.insert(ctx, SecondFactorFailure{ID: uuid.New(), UserID: user.ID, FailedAt: now})
	})

	return isAccepted, err
}

// spendCode tells an app code from a recovery code by its form: six digits are an app code.
func (s *Service) spendCode(ctx context.Context, user identity.User, app AuthenticatorApp, code string) (bool, error) {
	if totp.IsCodeFormat(code) {
		return s.spendAppCode(ctx, app, code)
	}

	isSpent, err := s.recoveryCodes.spend(ctx, app.ID, hashRecoveryCode(normalizeRecoveryCode(code)), s.clock.Now())
	if err != nil || !isSpent {
		return false, err
	}

	return true, s.recordAuditEvent(ctx, user, "Recovery code used: "+user.Email)
}

func (s *Service) spendAppCode(ctx context.Context, app AuthenticatorApp, code string) (bool, error) {
	secret, err := s.encryptor.Decrypt(app.EncryptedSecret)
	if err != nil {
		return false, err
	}

	matchingStep, isMatched, err := totp.FindMatchingStep(secret, code, s.clock.Now())
	if err != nil || !isMatched {
		return false, err
	}

	return s.apps.advanceLastUsedStep(ctx, app.ID, matchingStep)
}
