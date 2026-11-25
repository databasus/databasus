package signin

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/authenticator"
)

// authenticatorCodeLifetime matches the emailed code, so both second-factor screens last as long.
const authenticatorCodeLifetime = signInCodeLifetime

// SetupSignIn is the session of a sign-in that connected the required app, with the app's recovery
// codes, which only this answer carries.
type SetupSignIn struct {
	Session       Session
	RecoveryCodes []string
}

// startAuthenticatorCodeSignIn hands back the live pending sign-in that waits for a code of the same
// app and was started with the same password, so reaching the code screen again changes nothing.
// It sends no mail and so stays outside the hourly cap on emailed codes.
func (s *Service) startAuthenticatorCodeSignIn(
	ctx context.Context,
	user accounts.UserRecord,
	appID uuid.UUID,
) (PendingSignInView, error) {
	now := s.clock.Now()
	newPendingSignIn := PendingSignIn{
		ID:                   uuid.New(),
		UserID:               user.ID,
		Kind:                 PendingSignInKindAuthenticatorCode,
		HashedCode:           nil,
		AuthenticatorAppID:   &appID,
		AuthenticatorSetupID: nil,
		PasswordVersion:      user.PasswordVersion,
		ExpiresAt:            now.Add(authenticatorCodeLifetime),
		IsUsed:               false,
		FailedAttemptCount:   0,
		CreatedAt:            now,
	}

	pendingSignInID := newPendingSignIn.ID

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.pendingSignIns.lockUser(ctx, user.ID)
		if err != nil {
			return err
		}

		livePendingSignIn, isLive, err := s.pendingSignIns.getLive(ctx, livePendingSignInFilter{
			userID:             user.ID,
			kind:               PendingSignInKindAuthenticatorCode,
			authenticatorAppID: &appID,
			passwordVersion:    user.PasswordVersion,
			now:                now,
		})
		if err != nil {
			return err
		}

		if isLive {
			pendingSignInID = livePendingSignIn.ID

			return nil
		}

		return s.pendingSignIns.insert(ctx, newPendingSignIn)
	})
	if err != nil {
		return PendingSignInView{}, err
	}

	return PendingSignInView{
		PendingSignInID: pendingSignInID,
		Email:           user.Email,
		Kind:            PendingSignInKindAuthenticatorCode,
		Setup:           nil,
	}, nil
}

// startAuthenticatorSetupSignIn starts a new setup on every password step instead of handing back a
// live one, so each secret appears in exactly one answer. The new setup replaces the earlier one
// and the pending sign-in that waited for it.
func (s *Service) startAuthenticatorSetupSignIn(
	ctx context.Context,
	user accounts.UserRecord,
) (PendingSignInView, error) {
	var view PendingSignInView

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.pendingSignIns.lockUser(ctx, user.ID)
		if err != nil {
			return err
		}

		startedSetup, err := s.authenticator.StartSetup(ctx, user.ToUser())
		if err != nil {
			return err
		}

		pendingSignIn := PendingSignIn{
			ID:                   uuid.New(),
			UserID:               user.ID,
			Kind:                 PendingSignInKindAuthenticatorSetup,
			HashedCode:           nil,
			AuthenticatorAppID:   nil,
			AuthenticatorSetupID: &startedSetup.SetupID,
			PasswordVersion:      user.PasswordVersion,
			ExpiresAt:            startedSetup.ExpiresAt,
			IsUsed:               false,
			FailedAttemptCount:   0,
			CreatedAt:            s.clock.Now(),
		}

		view = PendingSignInView{
			PendingSignInID: pendingSignIn.ID,
			Email:           user.Email,
			Kind:            PendingSignInKindAuthenticatorSetup,
			Setup:           &startedSetup,
		}

		return s.pendingSignIns.insert(ctx, pendingSignIn)
	})

	return view, err
}

// verifyAuthenticatorCode completes a pending sign-in with an app code or a recovery code. The
// attempt is claimed first, as for an emailed code, and the code check then counts a wrong code
// toward the per-account limit as well.
func (s *Service) verifyAuthenticatorCode(
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

	isAccepted, err := s.authenticator.CheckCode(ctx, user.ToUser(), code)
	if err != nil {
		return Session{}, err
	}

	if !isAccepted {
		return Session{}, s.refuseWrongSignInCode(ctx, attemptCount, user)
	}

	return s.spendAndCompleteSignIn(ctx, pendingSignIn, user)
}

// ConfirmSignInAuthenticatorAppSetup connects the app the instance requires and issues the session
// in the same transaction. Deleting the confirmed setup deletes the pending sign-in that waited for
// it, which is what keeps two confirmations with one code from both signing in.
func (s *Service) ConfirmSignInAuthenticatorAppSetup(
	ctx context.Context,
	pendingSignInID uuid.UUID,
	code string,
	turnstileAnswer *string,
) (SetupSignIn, error) {
	err := s.turnstile.Verify(ctx, turnstileAnswer)
	if err != nil {
		return SetupSignIn{}, err
	}

	err = s.checkRateLimit(ctx, ratelimit.Attempt{
		Scope:      signInCodeVerificationScope,
		Identifier: pendingSignInID.String(),
		Limit:      signInCodeVerificationLimit,
		Window:     signInCodeVerificationWindow,
	})
	if err != nil {
		return SetupSignIn{}, err
	}

	pendingSignIn, user, err := s.loadUsablePendingSignIn(ctx, pendingSignInID)
	if err != nil {
		return SetupSignIn{}, err
	}

	if pendingSignIn.Kind != PendingSignInKindAuthenticatorSetup || pendingSignIn.AuthenticatorSetupID == nil {
		return SetupSignIn{}, ErrPendingSignInNotUsable
	}

	attemptCount, isAttemptClaimed, err := s.pendingSignIns.claimAttempt(ctx, pendingSignIn.ID, s.clock.Now())
	if err != nil {
		return SetupSignIn{}, err
	}

	if !isAttemptClaimed {
		return SetupSignIn{}, ErrPendingSignInNotUsable
	}

	var setupSignIn SetupSignIn

	isConnected := false

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		confirmation, err := s.authenticator.ConfirmSetup(ctx, user.ToUser(), *pendingSignIn.AuthenticatorSetupID, code)
		if errors.Is(err, authenticator.ErrSetupNotUsable) {
			return ErrPendingSignInNotUsable
		}

		if err != nil || !confirmation.IsConnected {
			return err
		}

		isConnected = true

		session, err := s.completeSignIn(ctx, user)
		setupSignIn = SetupSignIn{Session: session, RecoveryCodes: confirmation.RecoveryCodes}

		return err
	})
	if err != nil {
		return SetupSignIn{}, err
	}

	if !isConnected {
		return SetupSignIn{}, s.refuseWrongSignInCode(ctx, attemptCount, user)
	}

	return setupSignIn, nil
}
