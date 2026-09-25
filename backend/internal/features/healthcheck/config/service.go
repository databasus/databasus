package healthcheck_config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	"databasus-backend/internal/features/databases"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

type HealthcheckConfigService struct {
	databaseService             *databases.DatabaseService
	healthcheckConfigRepository *HealthcheckConfigRepository
	workspaceService            *workspaces_services.WorkspaceService
	auditLogService             *audit_logs.AuditLogService
	logger                      *slog.Logger
}

func (s *HealthcheckConfigService) OnDatabaseCreated(
	ctx context.Context,
	databaseID uuid.UUID,
) {
	err := s.initializeDefaultConfig(databaseID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to initialize default healthcheck config", "error", err)
	}
}

func (s *HealthcheckConfigService) Save(
	ctx context.Context,
	user users_models.User,
	configDTO HealthcheckConfigDTO,
) error {
	database, err := s.databaseService.GetDatabaseByID(configDTO.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot modify healthcheck config for databases without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, &user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to modify healthcheck config")
	}

	healthcheckConfig := configDTO.ToDTO()

	healthcheckConfig.DatabaseID = database.ID

	err = s.healthcheckConfigRepository.Save(healthcheckConfig)
	if err != nil {
		return err
	}

	// for DBs with disabled healthcheck, we keep
	// health status as available
	if !healthcheckConfig.IsHealthcheckEnabled &&
		database.HealthStatus != nil {
		err = s.databaseService.SetHealthStatus(
			database.ID,
			nil,
		)
		if err != nil {
			return err
		}
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Healthcheck config updated for database '%s'", database.Name),
		UserID:      &user.ID,
		WorkspaceID: database.WorkspaceID,
	})

	return nil
}

func (s *HealthcheckConfigService) GetByDatabaseID(
	ctx context.Context,
	user users_models.User,
	databaseID uuid.UUID,
) (*HealthcheckConfig, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot access healthcheck config for databases without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, &user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to view healthcheck config")
	}

	config, err := s.healthcheckConfigRepository.GetByDatabaseID(database.ID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		err = s.initializeDefaultConfig(database.ID)
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

func (s *HealthcheckConfigService) GetConfigsByDatabaseIDs(
	databaseIDs []uuid.UUID,
) ([]HealthcheckConfig, error) {
	return s.healthcheckConfigRepository.FindByDatabaseIDs(databaseIDs)
}

func (s *HealthcheckConfigService) initializeDefaultConfig(
	databaseID uuid.UUID,
) error {
	return s.healthcheckConfigRepository.Save(&HealthcheckConfig{
		DatabaseID:                        databaseID,
		IsHealthcheckEnabled:              true,
		IsSentNotificationWhenUnavailable: true,
		IntervalMinutes:                   1,
		AttemptsBeforeConcideredAsDown:    3,
		StoreAttemptsDays:                 7,
	})
}
