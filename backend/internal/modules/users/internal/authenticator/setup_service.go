package authenticator

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/totp"
	"databasus-backend/internal/modules/users/internal/identity"
)

// StartSetup replaces the user's unconfirmed setup, and the pending sign-in that waited for it,
// with a new one under a new secret. It may run inside the caller's transaction, so a sign-in binds
// its pending sign-in to the setup atomically.
func (s *Service) StartSetup(ctx context.Context, user identity.User) (StartedAuthenticatorAppSetup, error) {
	secret, err := totp.GenerateSecret()
	if err != nil {
		return StartedAuthenticatorAppSetup{}, err
	}

	encryptedSecret, err := s.encryptor.Encrypt(secret)
	if err != nil {
		return StartedAuthenticatorAppSetup{}, err
	}

	now := s.clock.Now()
	setup := AuthenticatorAppSetup{
		ID:                 uuid.New(),
		UserID:             user.ID,
		EncryptedSecret:    encryptedSecret,
		FailedAttemptCount: 0,
		ExpiresAt:          now.Add(setupLifetime),
		CreatedAt:          now,
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.apps.lockUser(ctx, user.ID)
		if err != nil {
			return err
		}

		err = s.setups.deleteByUserID(ctx, user.ID)
		if err != nil {
			return err
		}

		return s.setups.insert(ctx, setup)
	})
	if err != nil {
		return StartedAuthenticatorAppSetup{}, err
	}

	return StartedAuthenticatorAppSetup{
		SetupID:   setup.ID,
		Secret:    secret,
		Address:   totp.BuildAddress(addressIssuer, user.Email, secret),
		ExpiresAt: setup.ExpiresAt,
	}, nil
}

// ConfirmSetup connects the app of the setup when the code came from its secret, replacing the
// app connected before together with its recovery codes and the pending sign-ins that waited for
// it, and deleting the setup with the pending sign-in that waited for the setup. A wrong code
// counts only toward the setup's own five, never toward the per-account limit: whoever confirms a
// setup already sees its secret.
//
// A wrong code is reported without an error, so it is recorded when this runs inside the caller's
// transaction. A setup that cannot be confirmed answers ErrSetupNotUsable.
func (s *Service) ConfirmSetup(
	ctx context.Context,
	user identity.User,
	setupID uuid.UUID,
	code string,
) (AuthenticatorAppSetupConfirmation, error) {
	var confirmation AuthenticatorAppSetupConfirmation

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.apps.lockUser(ctx, user.ID)
		if err != nil {
			return err
		}

		setup, isFound, err := s.setups.getByID(ctx, setupID)
		if err != nil {
			return err
		}

		if !isFound || setup.UserID != user.ID || !setup.IsConfirmable(s.clock.Now()) {
			return ErrSetupNotUsable
		}

		secret, err := s.encryptor.Decrypt(setup.EncryptedSecret)
		if err != nil {
			return err
		}

		matchingStep, isMatched, err := totp.FindMatchingStep(secret, code, s.clock.Now())
		if err != nil {
			return err
		}

		if !isMatched {
			return s.setups.incrementFailedAttemptCount(ctx, setup.ID)
		}

		confirmation, err = s.connectApp(ctx, user, setup, matchingStep)

		return err
	})

	return confirmation, err
}

// connectApp starts the new app at the confirming code's period, so that code cannot be replayed
// right after the setup.
func (s *Service) connectApp(
	ctx context.Context,
	user identity.User,
	setup AuthenticatorAppSetup,
	confirmedStep int64,
) (AuthenticatorAppSetupConfirmation, error) {
	isReplaced, err := s.apps.deleteByUserID(ctx, user.ID)
	if err != nil {
		return AuthenticatorAppSetupConfirmation{}, err
	}

	app := AuthenticatorApp{
		ID:              uuid.New(),
		UserID:          user.ID,
		EncryptedSecret: setup.EncryptedSecret,
		LastUsedStep:    confirmedStep,
		ConnectedAt:     s.clock.Now(),
	}

	err = s.apps.insert(ctx, app)
	if err != nil {
		return AuthenticatorAppSetupConfirmation{}, err
	}

	recoveryCodes, err := s.issueRecoveryCodes(ctx, app.ID)
	if err != nil {
		return AuthenticatorAppSetupConfirmation{}, err
	}

	err = s.setups.deleteByID(ctx, setup.ID)
	if err != nil {
		return AuthenticatorAppSetupConfirmation{}, err
	}

	auditMessage := "Authenticator app connected: " + user.Email
	if isReplaced {
		auditMessage = "Authenticator app replaced: " + user.Email
	}

	err = s.recordAuditEvent(ctx, user, auditMessage)
	if err != nil {
		return AuthenticatorAppSetupConfirmation{}, err
	}

	return AuthenticatorAppSetupConfirmation{IsConnected: true, RecoveryCodes: recoveryCodes}, nil
}

func (s *Service) issueRecoveryCodes(ctx context.Context, appID uuid.UUID) ([]string, error) {
	shownCodes, storedCodes, err := generateRecoveryCodes(appID)
	if err != nil {
		return nil, err
	}

	err = s.recoveryCodes.insertAll(ctx, storedCodes)
	if err != nil {
		return nil, err
	}

	return shownCodes, nil
}
