package healthcheck_config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	databases_bridge "databasus-backend/internal/features/databases"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type HealthcheckConfigService struct {
	databaseAccess *databases_bridge.Access
	// getDatabases is read per call: each test application installs its own module.
	getDatabases                func() databases.Module
	healthcheckConfigRepository *HealthcheckConfigRepository
	workspaceService            *workspaces_bridge.Access
	auditLogService             *audit_logs.AuditLogService
	logger                      *slog.Logger
}

// OnDatabaseCreated writes the default config in the creation's transaction.
func (s *HealthcheckConfigService) OnDatabaseCreated(
	ctx context.Context,
	event databases.DatabaseCreated,
) error {
	return s.initializeDefaultConfig(ctx, event.DatabaseID)
}

func (s *HealthcheckConfigService) Save(
	ctx context.Context,
	actor users.Actor,
	configDTO HealthcheckConfigDTO,
) error {
	database, err := s.databaseAccess.GetDatabaseByID(ctx, configDTO.DatabaseID)
	if err != nil {
		return err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return err
	}
	if !isAllowed {
		return errors.New("insufficient permissions to modify healthcheck config")
	}

	healthcheckConfig := configDTO.ToDTO()

	healthcheckConfig.DatabaseID = database.ID

	err = s.healthcheckConfigRepository.Save(ctx, healthcheckConfig)
	if err != nil {
		return err
	}

	// for DBs with disabled healthcheck, we keep
	// health status as available
	if !healthcheckConfig.IsHealthcheckEnabled &&
		database.HealthStatus != nil {
		err = s.getDatabases().SetHealthStatus(ctx, database.ID, nil)
		if err != nil {
			return err
		}
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Healthcheck config updated for database '%s'", database.Name),
		UserID:      &actor.UserID,
		WorkspaceID: &database.WorkspaceID,
	})

	return nil
}

func (s *HealthcheckConfigService) GetByDatabaseID(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (*HealthcheckConfig, error) {
	database, err := s.databaseAccess.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("insufficient permissions to view healthcheck config")
	}

	config, err := s.healthcheckConfigRepository.GetByDatabaseID(database.ID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		err = s.initializeDefaultConfig(ctx, database.ID)
		if err != nil {
			return nil, err
		}

		config, err = s.healthcheckConfigRepository.GetByDatabaseID(database.ID)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (s *HealthcheckConfigService) GetDatabasesWithEnabledHealthcheck() (
	[]HealthcheckConfig, error,
) {
	return s.healthcheckConfigRepository.GetDatabasesWithEnabledHealthcheck()
}

func (s *HealthcheckConfigService) initializeDefaultConfig(
	ctx context.Context,
	databaseID uuid.UUID,
) error {
	return s.healthcheckConfigRepository.Save(ctx, &HealthcheckConfig{
		DatabaseID:                        databaseID,
		IsHealthcheckEnabled:              true,
		IsSentNotificationWhenUnavailable: true,
		IntervalMinutes:                   1,
		AttemptsBeforeConcideredAsDown:    3,
		StoreAttemptsDays:                 7,
	})
}
