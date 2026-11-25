package authenticator

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/settings"
)

// StartProfileSetup starts connecting an app to the actor's own account. A user who already has an
// app replaces it this way, which needs a current app code or recovery code: a session alone must
// not move the second factor to another device. The connected app keeps working until the new one
// is confirmed.
func (s *Service) StartProfileSetup(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	code string,
) (StartedAuthenticatorAppSetup, error) {
	user, err := s.getOwnUser(ctx, actor, userID)
	if err != nil {
		return StartedAuthenticatorAppSetup{}, err
	}

	_, hasApp, err := s.apps.getByUserID(ctx, user.ID)
	if err != nil {
		return StartedAuthenticatorAppSetup{}, err
	}

	if hasApp {
		err = s.requireCurrentCode(ctx, user, code)
		if err != nil {
			return StartedAuthenticatorAppSetup{}, err
		}
	}

	return s.StartSetup(ctx, user)
}

func (s *Service) ConfirmProfileSetup(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	setupID uuid.UUID,
	code string,
) ([]string, error) {
	user, err := s.getOwnUser(ctx, actor, userID)
	if err != nil {
		return nil, err
	}

	confirmation, err := s.ConfirmSetup(ctx, user, setupID, code)
	if err != nil {
		return nil, err
	}

	if !confirmation.IsConnected {
		return nil, ErrSecondFactorInvalid
	}

	return confirmation.RecoveryCodes, nil
}

// Disconnect removes the actor's own app and its recovery codes. It is refused while the instance
// requires the authenticator app method, because the next sign-in would only set up another one.
func (s *Service) Disconnect(ctx context.Context, actor identity.Actor, userID uuid.UUID, code string) error {
	user, err := s.getOwnUser(ctx, actor, userID)
	if err != nil {
		return err
	}

	requirement, err := s.settings.GetTwoFactorRequirement(ctx)
	if err != nil {
		return err
	}

	if requirement == settings.TwoFactorRequirementAuthenticatorApp {
		return ErrAppRequired
	}

	err = s.requireCurrentCode(ctx, user, code)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		isDeleted, err := s.apps.deleteByUserID(ctx, user.ID)
		if err != nil || !isDeleted {
			return err
		}

		return s.recordAuditEvent(ctx, user, "Authenticator app disconnected: "+user.Email)
	})
}

func (s *Service) RegenerateRecoveryCodes(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	code string,
) ([]string, error) {
	user, err := s.getOwnUser(ctx, actor, userID)
	if err != nil {
		return nil, err
	}

	err = s.requireCurrentCode(ctx, user, code)
	if err != nil {
		return nil, err
	}

	var recoveryCodes []string

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.apps.lockUser(ctx, user.ID)
		if err != nil {
			return err
		}

		app, hasApp, err := s.apps.getByUserID(ctx, user.ID)
		if err != nil {
			return err
		}

		if !hasApp {
			return ErrSecondFactorInvalid
		}

		err = s.recoveryCodes.deleteByAppID(ctx, app.ID)
		if err != nil {
			return err
		}

		recoveryCodes, err = s.issueRecoveryCodes(ctx, app.ID)
		if err != nil {
			return err
		}

		return s.recordAuditEvent(ctx, user, "Recovery codes regenerated: "+user.Email)
	})

	return recoveryCodes, err
}

func (s *Service) requireCurrentCode(ctx context.Context, user identity.User, code string) error {
	if code == "" {
		return ErrSecondFactorInvalid
	}

	isAccepted, err := s.CheckCode(ctx, user, code)
	if err != nil {
		return err
	}

	if !isAccepted {
		return ErrSecondFactorInvalid
	}

	return nil
}

func (s *Service) getOwnUser(ctx context.Context, actor identity.Actor, userID uuid.UUID) (identity.User, error) {
	if actor.UserID != userID {
		return identity.User{}, ErrNotOwnUser
	}

	user, isFound, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return identity.User{}, err
	}

	if !isFound {
		return identity.User{}, fmt.Errorf("the signed-in user %s does not exist", userID)
	}

	return user, nil
}
