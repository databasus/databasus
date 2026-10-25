package notifiers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/encryption"
)

type NotifierService struct {
	notifierRepository      *NotifierRepository
	logger                  *slog.Logger
	workspaceService        *workspaces_services.WorkspaceService
	auditLogService         *audit_logs.AuditLogService
	fieldEncryptor          encryption.FieldEncryptor
	notifierDatabaseCounter NotifierDatabaseCounter
}

func (s *NotifierService) SetNotifierDatabaseCounter(
	notifierDatabaseCounter NotifierDatabaseCounter,
) {
	s.notifierDatabaseCounter = notifierDatabaseCounter
}

func (s *NotifierService) SaveNotifier(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	notifier *Notifier,
) error {
	canManage, err := s.workspaceService.CanUserManageDBs(ctx, workspaceID, actor)
	if err != nil {
		return err
	}
	if !canManage {
		return ErrInsufficientPermissionsToManageNotifier
	}

	isUpdate := notifier.ID != uuid.Nil

	if isUpdate {
		existingNotifier, err := s.notifierRepository.FindByID(ctx, notifier.ID)
		if err != nil {
			return err
		}

		if existingNotifier.WorkspaceID != workspaceID {
			return ErrNotifierDoesNotBelongToWorkspace
		}

		existingNotifier.Update(notifier)

		if err := existingNotifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		oldName := existingNotifier.Name

		if err := existingNotifier.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.notifierRepository.Save(existingNotifier)
		if err != nil {
			return err
		}

		if oldName != existingNotifier.Name {
			s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
				Message: fmt.Sprintf(
					"Notifier updated and renamed from '%s' to '%s'",
					oldName,
					existingNotifier.Name,
				),
				UserID:      &actor.UserID,
				WorkspaceID: &workspaceID,
			})
		} else {
			s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
				Message:     fmt.Sprintf("Notifier updated: %s", existingNotifier.Name),
				UserID:      &actor.UserID,
				WorkspaceID: &workspaceID,
			})
		}
	} else {
		notifier.WorkspaceID = workspaceID
		notifier.FillDefaults()

		if err := notifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := notifier.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.notifierRepository.Save(notifier)
		if err != nil {
			return err
		}

		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Notifier created: %s", notifier.Name),
			UserID:      &actor.UserID,
			WorkspaceID: &workspaceID,
		})
	}

	return nil
}

func (s *NotifierService) DeleteNotifier(
	ctx context.Context,
	actor users.Actor,
	notifierID uuid.UUID,
) error {
	notifier, err := s.notifierRepository.FindByID(ctx, notifierID)
	if err != nil {
		return err
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, notifier.WorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canManage {
		return ErrInsufficientPermissionsToManageNotifier
	}

	attachedDatabasesIDs, err := s.notifierDatabaseCounter.GetNotifierAttachedDatabasesIDs(
		notifier.ID,
	)
	if err != nil {
		return err
	}
	if len(attachedDatabasesIDs) > 0 {
		return ErrNotifierHasAttachedDatabases
	}

	err = s.notifierRepository.Delete(notifier)
	if err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Notifier deleted: %s", notifier.Name),
		UserID:      &actor.UserID,
		WorkspaceID: &notifier.WorkspaceID,
	})

	return nil
}

func (s *NotifierService) GetNotifier(
	ctx context.Context,
	actor users.Actor,
	id uuid.UUID,
) (*Notifier, error) {
	notifier, err := s.notifierRepository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, notifier.WorkspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, ErrInsufficientPermissionsToViewNotifier
	}

	notifier.HideSensitiveData()
	return notifier, nil
}

func (s *NotifierService) GetNotifierByID(ctx context.Context, id uuid.UUID) (*Notifier, error) {
	return s.notifierRepository.FindByID(ctx, id)
}

func (s *NotifierService) GetAllNotifiers() ([]*Notifier, error) {
	return s.notifierRepository.GetAllNotifiers()
}

func (s *NotifierService) GetNotifiers(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
) ([]*Notifier, error) {
	canView, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, workspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, ErrInsufficientPermissionsToViewNotifiers
	}

	notifiers, err := s.notifierRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, notifier := range notifiers {
		notifier.HideSensitiveData()
	}

	return notifiers, nil
}

func (s *NotifierService) SendTestNotification(
	ctx context.Context,
	actor users.Actor,
	notifierID uuid.UUID,
) error {
	notifier, err := s.notifierRepository.FindByID(ctx, notifierID)
	if err != nil {
		return err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, notifier.WorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canView {
		return ErrInsufficientPermissionsToTestNotifier
	}

	err = notifier.Send(s.fieldEncryptor, s.logger, newTestNotification())
	if err != nil {
		return err
	}

	_, err = s.notifierRepository.Save(notifier)
	if err != nil {
		return err
	}

	return nil
}

func (s *NotifierService) SendTestNotificationToNotifier(
	ctx context.Context,
	notifier *Notifier,
) error {
	var usingNotifier *Notifier

	if notifier.ID != uuid.Nil {
		existingNotifier, err := s.notifierRepository.FindByID(ctx, notifier.ID)
		if err != nil {
			return err
		}

		if existingNotifier.WorkspaceID != notifier.WorkspaceID {
			return ErrNotifierDoesNotBelongToWorkspace
		}

		existingNotifier.Update(notifier)

		if err := existingNotifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := existingNotifier.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		usingNotifier = existingNotifier
	} else {
		notifier.FillDefaults()

		if err := notifier.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		usingNotifier = notifier
	}

	return usingNotifier.Send(s.fieldEncryptor, s.logger, newTestNotification())
}

func newTestNotification() notifier_models.Notification {
	return notifier_models.Notification{
		Type:    notifier_models.NotificationTypeAll,
		Heading: "Test message",
		Message: "This is a test message",
	}
}

func (s *NotifierService) SendNotification(
	ctx context.Context,
	notifier *Notifier,
	notification notifier_models.Notification,
) {
	messageRunes := []rune(notification.Message)
	if len(messageRunes) > 2000 {
		notification.Message = string(messageRunes[:2000])
	}

	notifiedFromDb, err := s.notifierRepository.FindByID(ctx, notifier.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load notifier before sending", "notifier_id", notifier.ID, "error", err)

		return
	}

	err = notifiedFromDb.Send(s.fieldEncryptor, s.logger, notification)
	if err != nil {
		// The notifier keeps working from the user's point of view, so without this a silently
		// broken destination is only visible by opening the notifier in the UI. Notifier backends
		// strip the credential from their transport errors before returning them.
		s.logger.ErrorContext(ctx, "failed to send notification",
			"notifier_id", notifiedFromDb.ID, "notification_type", notification.Type, "error", err)

		errMsg := err.Error()
		notifiedFromDb.LastSendError = &errMsg

		_, err = s.notifierRepository.Save(notifiedFromDb)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to save notifier", "error", err)
		}

		return
	}

	notifiedFromDb.LastSendError = nil
	_, err = s.notifierRepository.Save(notifiedFromDb)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to save notifier", "error", err)
	}
}

func (s *NotifierService) TransferNotifierToWorkspace(
	ctx context.Context,
	actor users.Actor,
	notifierID uuid.UUID,
	targetWorkspaceID uuid.UUID,
	transferingWithDbID *uuid.UUID,
) error {
	existingNotifier, err := s.notifierRepository.FindByID(ctx, notifierID)
	if err != nil {
		return err
	}

	canManageSource, err := s.workspaceService.CanUserManageDBs(ctx, existingNotifier.WorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canManageSource {
		return ErrInsufficientPermissionsInSourceWorkspace
	}

	canManageTarget, err := s.workspaceService.CanUserManageDBs(ctx, targetWorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canManageTarget {
		return ErrInsufficientPermissionsInTargetWorkspace
	}

	attachedDatabasesIDs, err := s.notifierDatabaseCounter.GetNotifierAttachedDatabasesIDs(
		existingNotifier.ID,
	)
	if err != nil {
		return err
	}

	if transferingWithDbID != nil {
		for _, dbID := range attachedDatabasesIDs {
			if dbID != *transferingWithDbID {
				return ErrNotifierHasOtherAttachedDatabasesCannotTransfer
			}
		}
	} else if len(attachedDatabasesIDs) > 0 {
		return ErrNotifierHasAttachedDatabasesCannotTransfer
	}

	sourceWorkspaceID := existingNotifier.WorkspaceID
	existingNotifier.WorkspaceID = targetWorkspaceID

	_, err = s.notifierRepository.Save(existingNotifier)
	if err != nil {
		return err
	}

	sourceWorkspace, err := s.workspaceService.GetWorkspaceByID(sourceWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get source workspace: %w", err)
	}

	targetWorkspace, err := s.workspaceService.GetWorkspaceByID(targetWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get target workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message: fmt.Sprintf(
			"Notifier transferred: %s from workspace '%s' to workspace '%s'",
			existingNotifier.Name,
			sourceWorkspace.Name,
			targetWorkspace.Name,
		),
		UserID:      &actor.UserID,
		WorkspaceID: &targetWorkspaceID,
	})

	return nil
}

func (s *NotifierService) OnBeforeWorkspaceDeletion(workspaceID uuid.UUID) error {
	notifiers, err := s.notifierRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get notifiers for workspace deletion: %w", err)
	}

	for _, notifier := range notifiers {
		if err := s.notifierRepository.Delete(notifier); err != nil {
			return fmt.Errorf("failed to delete notifier %s: %w", notifier.ID, err)
		}
	}

	return nil
}
