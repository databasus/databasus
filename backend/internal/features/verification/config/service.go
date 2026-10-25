package verification_config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
)

type VerificationConfigService struct {
	verificationConfigRepository *VerificationConfigRepository
	databaseService              *databases.DatabaseService
	workspaceService             *workspaces_services.WorkspaceService
	auditLogService              *audit_logs.AuditLogService
	logger                       *slog.Logger
}

func (s *VerificationConfigService) GetByDatabaseID(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (*BackupVerificationConfig, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot access verification config for databases without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to view verification config")
	}

	config, err := s.verificationConfigRepository.GetByDatabaseID(database.ID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if err := s.initializeDefaultConfig(database.ID); err != nil {
			return nil, err
		}

		config, err = s.verificationConfigRepository.GetByDatabaseID(database.ID)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (s *VerificationConfigService) Save(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	req *SaveBackupVerificationConfigDTO,
) (*BackupVerificationConfig, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot modify verification config for databases without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to modify verification config")
	}

	config, err := s.verificationConfigRepository.GetByDatabaseID(database.ID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if err := s.initializeDefaultConfig(database.ID); err != nil {
			return nil, err
		}

		config, err = s.verificationConfigRepository.GetByDatabaseID(database.ID)
		if err != nil {
			return nil, err
		}
	}

	config.IsScheduledVerificationEnabled = req.IsScheduledVerificationEnabled
	config.ScheduleType = req.ScheduleType
	config.VerificationInterval = req.VerificationInterval
	config.SendNotificationsOn = req.SendNotificationsOn

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if err := s.verificationConfigRepository.Save(config); err != nil {
		return nil, err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Backup verification config updated for database '%s'", database.Name),
		UserID:      &actor.UserID,
		WorkspaceID: database.WorkspaceID,
	})

	return config, nil
}

func (s *VerificationConfigService) GetByDatabaseIDNoAuth(
	databaseID uuid.UUID,
) (*BackupVerificationConfig, error) {
	return s.verificationConfigRepository.GetByDatabaseID(databaseID)
}

func (s *VerificationConfigService) ListEnabled() ([]*BackupVerificationConfig, error) {
	return s.verificationConfigRepository.FindAllEnabled()
}

func (s *VerificationConfigService) OnDatabaseCopied(ctx context.Context, originalDatabaseID, newDatabaseID uuid.UUID) {
	originalConfig, err := s.verificationConfigRepository.GetByDatabaseID(originalDatabaseID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to load source verification config on database copy",
			"error", err,
			"database_id", originalDatabaseID,
		)
		return
	}

	if originalConfig == nil {
		if err := s.initializeDefaultConfig(newDatabaseID); err != nil {
			s.logger.ErrorContext(
				ctx,
				"failed to initialize default verification config on database copy",
				"error", err,
				"database_id", newDatabaseID,
			)
		}
		return
	}

	newConfig := originalConfig.Copy(newDatabaseID)

	if err := s.verificationConfigRepository.Save(newConfig); err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to save verification config on database copy",
			"error", err,
			"database_id", newDatabaseID,
		)
	}
}

func (s *VerificationConfigService) initializeDefaultConfig(databaseID uuid.UUID) error {
	return s.verificationConfigRepository.Save(&BackupVerificationConfig{
		DatabaseID:                     databaseID,
		IsScheduledVerificationEnabled: false,
		ScheduleType:                   VerificationScheduleAfterBackup,
		VerificationInterval: intervals.Interval{
			Type:      intervals.IntervalWeekly,
			TimeOfDay: new("04:00"),
			Weekday:   new(0),
		},
		SendNotificationsOn: []VerificationNotificationType{
			NotificationVerificationFailed,
		},
	})
}
