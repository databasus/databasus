package authenticator

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
)

// DeleteApp removes the user's app with its recovery codes and the pending sign-ins that wait for
// its code; sessions stay. It reports false when the user had no app. It runs inside the caller's
// transaction, which decides who may reset whom and records the reset.
func (s *Service) DeleteApp(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isDeleted bool

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.apps.lockUser(ctx, userID)
		if err != nil {
			return err
		}

		isDeleted, err = s.apps.deleteByUserID(ctx, userID)

		return err
	})

	return isDeleted, err
}

// ResetAppFromConsole is the way back for an owner who lost the phone and the recovery codes, so it
// takes no actor: whoever runs it already runs commands inside the instance. The audit event keeps
// the host from being a way to move a second factor unobserved. It reports false when the account
// had no app.
func (s *Service) ResetAppFromConsole(ctx context.Context, email string) (bool, error) {
	var isDeleted bool

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		user, isFound, err := s.users.GetUserByEmail(ctx, email)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrConsoleUserNotFound
		}

		isDeleted, err = s.DeleteApp(ctx, user.ID)
		if err != nil || !isDeleted {
			return err
		}

		return s.auditLog.Record(ctx, audit.Event{
			Author:      nil,
			WorkspaceID: nil,
			Message:     "Authenticator app reset from the host console: " + user.Email,
		})
	})

	return isDeleted, err
}
