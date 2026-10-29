package verification_config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type VerificationConfigService struct {
	verificationConfigRepository *VerificationConfigRepository
	databaseAccess               *databases_bridge.Access
	workspaceService             *workspaces_bridge.Access
	auditLogService              *audit_logs.AuditLogService
	logger                       *slog.Logger
}

func (s *VerificationConfigService) GetByDatabaseID(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (*BackupVerificationConfig, error) {
	database, err := s.databaseAccess.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("insufficient permissions to view verification config")
	}

	config, err := s.verificationConfigRepository.GetByDatabaseID(database.ID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if err := s.initializeDefaultConfig(ctx, database.ID); err != nil {
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
	database, err := s.databaseAccess.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("insufficient permissions to modify verification config")
	}

	config, err := s.verificationConfigRepository.GetByDatabaseID(database.ID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if err := s.initializeDefaultConfig(ctx, database.ID); err != nil {
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
	config.SendNotificationsOn = make([]string, len(req.SendNotificationsOn))
	for index, notification := range req.SendNotificationsOn {
		config.SendNotificationsOn[index] = string(notification)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.VerificationInterval.Type == "" {
		config.VerificationInterval.Type = intervals.IntervalWeekly
	}

	if err := s.verificationConfigRepository.Save(ctx, config); err != nil {
		return nil, err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Backup verification config updated for database '%s'", database.Name),
		UserID:      &actor.UserID,
		WorkspaceID: &database.WorkspaceID,
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

// OnDatabaseCopied gives the copy the original's config in the copy's transaction, or the default one
// when the original has none.
func (s *VerificationConfigService) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	originalConfig, err := s.verificationConfigRepository.GetByDatabaseID(event.OriginalDatabaseID)
	if err != nil {
		return err
	}

	if originalConfig == nil {
		return s.initializeDefaultConfig(ctx, event.CopiedDatabaseID)
	}

	return s.verificationConfigRepository.Save(ctx, originalConfig.Copy(event.CopiedDatabaseID))
}

func (s *VerificationConfigService) initializeDefaultConfig(ctx context.Context, databaseID uuid.UUID) error {
	return s.verificationConfigRepository.Save(ctx, &BackupVerificationConfig{
		DatabaseID:                     databaseID,
		IsScheduledVerificationEnabled: false,
		ScheduleType:                   VerificationScheduleAfterBackup,
		VerificationInterval: intervals.Interval{
			Type:      intervals.IntervalWeekly,
			TimeOfDay: new("04:00"),
			Weekday:   new(0),
		},
		SendNotificationsOn: []string{
			string(NotificationVerificationFailed),
		},
	})
}
