package configs

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Clock        clock.Source
	Repositories Repositories
	Transactor   *database.Transactor
	AuditLog     *audit.Log
	Users        users.Module
	Workspaces   workspaces.Module
	Databases    databases.Module
}
type Service struct {
	clock      clock.Source
	repository *configRepository
	transactor *database.Transactor
	auditLog   *audit.Log
	users      users.Module
	workspaces workspaces.Module
	databases  databases.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		clock:      deps.Clock,
		repository: deps.Repositories.configs,
		transactor: deps.Transactor,
		auditLog:   deps.AuditLog,
		users:      deps.Users,
		workspaces: deps.Workspaces,
		databases:  deps.Databases,
	}
}

func (s *Service) GetConfig(ctx context.Context, actor users.Actor, id uuid.UUID) (Config, error) {
	if _, err := s.authorize(ctx, actor, id, workspaces.ActionView); err != nil {
		return Config{}, err
	}
	return s.GetConfigWithoutAuth(ctx, id)
}

func (s *Service) GetConfigWithoutAuth(ctx context.Context, id uuid.UUID) (Config, error) {
	config, isFound, err := s.repository.get(ctx, id)
	if err != nil {
		return Config{}, err
	}
	if !isFound {
		return GetDefault(id), nil
	}
	return config, nil
}

func (s *Service) SaveConfig(ctx context.Context, actor users.Actor, config Config) (Config, error) {
	database, err := s.authorize(ctx, actor, config.DatabaseID, workspaces.ActionEditContent)
	if err != nil {
		return Config{}, err
	}

	if config.ScheduleType != VerificationScheduleInterval && config.ScheduleType != VerificationScheduleAfterBackup {
		return Config{}, invalidField(
			apierror.FieldError{Field: "scheduleType", Detail: "must be INTERVAL or AFTER_BACKUP"},
		)
	}
	if config.IsAutomaticVerificationEnabled && config.ScheduleType == VerificationScheduleInterval {
		if err := config.Schedule.Validate(); err != nil {
			return Config{}, invalidField(apierror.FieldError{Field: "schedule", Detail: err.Error()})
		}
	}
	for _, trigger := range config.NotificationTriggers {
		if trigger != NotificationVerificationSuccess && trigger != NotificationVerificationFailed {
			return Config{}, invalidField(
				apierror.FieldError{Field: "notificationTriggers", Detail: "unknown notification trigger"},
			)
		}
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repository.save(ctx, config, s.clock.Now()); err != nil {
			return err
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
				WorkspaceID: &database.WorkspaceID,
				Message:     "verification config updated for database: " + database.Name,
			},
		)
	})
	return config, err
}

func (s *Service) ListEnabledConfigs(ctx context.Context) ([]Config, error) {
	return s.repository.listEnabled(ctx)
}

func (s *Service) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	config, isFound, err := s.repository.get(ctx, event.OriginalDatabaseID)
	if err != nil || !isFound {
		return err
	}
	config.DatabaseID = event.CopiedDatabaseID
	return s.repository.save(ctx, config, s.clock.Now())
}

func (s *Service) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return s.repository.delete(ctx, event.DatabaseID)
}

func (s *Service) authorize(
	ctx context.Context,
	actor users.Actor,
	id uuid.UUID,
	action workspaces.Action,
) (databases.Database, error) {
	foundDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{id})
	if err != nil {
		return databases.Database{}, err
	}
	if len(foundDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}
	database := foundDatabases[0]
	if err := s.workspaces.Authorize(ctx, actor, database.WorkspaceID, action); err != nil {
		var problem *apierror.Problem
		if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
			return databases.Database{}, ErrDatabaseNotFound
		}
		return databases.Database{}, err
	}
	return database, nil
}

func invalidField(field apierror.FieldError) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{field})
}
