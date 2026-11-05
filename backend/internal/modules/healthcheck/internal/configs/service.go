package configs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	AuditLog     *audit.Log
	Users        users.Module
	Workspaces   workspaces.Module
	Databases    databases.Module
}
type Service struct {
	repository *configRepository
	transactor *database.Transactor
	auditLog   *audit.Log
	users      users.Module
	workspaces workspaces.Module
	databases  databases.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		repository: deps.Repositories.configs,
		transactor: deps.Transactor,
		auditLog:   deps.AuditLog,
		users:      deps.Users,
		workspaces: deps.Workspaces,
		databases:  deps.Databases,
	}
}

func (s *Service) GetConfig(ctx context.Context, actor users.Actor, id uuid.UUID) (Config, error) {
	if _, err := s.Authorize(
		ctx,
		Authorization{Actor: actor, DatabaseID: id, Action: workspaces.ActionView},
	); err != nil {
		return Config{}, err
	}

	return s.GetCurrentConfig(ctx, id)
}

func (s *Service) SaveConfig(ctx context.Context, actor users.Actor, config Config) (Config, error) {
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		authorizedDatabase, err := s.Authorize(
			ctx,
			Authorization{Actor: actor, DatabaseID: config.DatabaseID, Action: workspaces.ActionEditContent},
		)
		if err != nil {
			return err
		}

		userDatabase, isFound, err := s.databases.GetDatabaseForUpdate(ctx, config.DatabaseID)
		if err != nil {
			return err
		}
		if !isFound {
			return ErrDatabaseNotFound
		}
		if userDatabase.WorkspaceID != authorizedDatabase.WorkspaceID {
			if _, err := s.Authorize(
				ctx,
				Authorization{Actor: actor, DatabaseID: config.DatabaseID, Action: workspaces.ActionEditContent},
			); err != nil {
				return err
			}
		}

		if _, err := s.GetCurrentConfig(ctx, config.DatabaseID); err != nil {
			return err
		}

		if err := s.repository.save(ctx, config); err != nil {
			return err
		}

		if !config.IsHealthcheckEnabled {
			if err := s.databases.SetHealthStatus(ctx, config.DatabaseID, nil); err != nil {
				return err
			}
		}

		author, isFound, err := s.users.GetUser(ctx, actor.UserID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrDatabaseNotFound
		}

		return s.auditLog.Record(
			ctx,
			audit.Event{
				Author:      author.GetAuditAuthor(),
				WorkspaceID: &userDatabase.WorkspaceID,
				Message:     fmt.Sprintf("Healthcheck config updated for database '%s'", userDatabase.Name),
			},
		)
	})
	return config, err
}

func (s *Service) ListDue(ctx context.Context, now time.Time) ([]Config, error) {
	return s.repository.listDue(ctx, now)
}

func (s *Service) OnDatabaseCreated(ctx context.Context, event databases.DatabaseCreated) error {
	return s.repository.save(ctx, GetDefaultConfig(event.DatabaseID))
}

func (s *Service) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	config, err := s.GetCurrentConfig(ctx, event.OriginalDatabaseID)
	if err != nil {
		return err
	}
	config.DatabaseID = event.CopiedDatabaseID
	return s.repository.save(ctx, config)
}

func (s *Service) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return s.repository.delete(ctx, event.DatabaseID)
}

func (s *Service) GetCurrentConfig(ctx context.Context, id uuid.UUID) (Config, error) {
	config, isFound, err := s.repository.getConfig(ctx, id)
	if err != nil {
		return Config{}, err
	}

	if !isFound {
		return Config{}, ErrConfigMissing
	}

	return config, nil
}
