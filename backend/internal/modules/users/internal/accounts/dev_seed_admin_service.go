package accounts

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users/internal/identity"
)

type DevSeedAccount struct {
	UserID uuid.UUID
	Email  string
}

type DevSeedAdminDeps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	AuditLog     *audit.Log
}

type DevSeedAdmin struct {
	repository *userRepository
	transactor *database.Transactor
	auditLog   *audit.Log
}

func NewDevSeedAdmin(deps DevSeedAdminDeps) *DevSeedAdmin {
	return &DevSeedAdmin{
		repository: deps.Repositories.users,
		transactor: deps.Transactor,
		auditLog:   deps.AuditLog,
	}
}

func (s *DevSeedAdmin) EnsureBootstrapAdmin(ctx context.Context, account DevSeedAccount) (bool, error) {
	var isChanged bool

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repository.lockAdministration(ctx); err != nil {
			return err
		}

		user, isFound, err := s.repository.getByID(ctx, account.UserID)
		if err != nil {
			return err
		}

		if !isFound || !user.IsActive() || user.Email != account.Email {
			return ErrDevSeedAccountMismatch
		}

		if user.IsBootstrapAdmin && user.Role == identity.RoleAdmin {
			return nil
		}

		previousAdmin, hasPreviousAdmin, err := s.repository.getBootstrapAdmin(ctx)
		if err != nil {
			return err
		}

		if hasPreviousAdmin && previousAdmin.ID != user.ID {
			if err := s.repository.updateBootstrapAdmin(ctx, previousAdmin.ID, false); err != nil {
				return err
			}
		}

		if user.Role != identity.RoleAdmin {
			if err := s.repository.updateRole(ctx, user.ID, identity.RoleAdmin); err != nil {
				return err
			}
		}

		if !user.IsBootstrapAdmin {
			if err := s.repository.updateBootstrapAdmin(ctx, user.ID, true); err != nil {
				return err
			}
		}

		message := fmt.Sprintf("Development seed assigned bootstrap administrator: %s (%s)", user.Email, user.ID)
		if hasPreviousAdmin {
			message += fmt.Sprintf("; previous holder: %s (%s)", previousAdmin.Email, previousAdmin.ID)
		}

		if err := s.auditLog.Record(ctx, audit.Event{Author: nil, WorkspaceID: nil, Message: message}); err != nil {
			return err
		}

		isChanged = true

		return nil
	})

	return isChanged, err
}
