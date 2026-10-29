package verifications

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/verification/internal/configs"
)

func (s *Service) syncBackupVerificationStatus(
	ctx context.Context,
	verification *Verification,
	terminalStatus VerificationStatus,
) {
	var status backups.LogicalBackupVerificationStatus

	switch terminalStatus {
	case VerificationStatusCompleted:
		status = backups.LogicalBackupVerificationStatusVerifiedSuccessful
	case VerificationStatusFailed:
		status = backups.LogicalBackupVerificationStatusVerificationFailed
	case VerificationStatusPending, VerificationStatusRunning, VerificationStatusCanceled:
		return
	}

	if err := s.backups.SetLogicalBackupVerificationStatus(ctx, verification.BackupID, status); err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to sync backup restore-verification status",
			slog.Any("error", err),
			slog.Any("verification_id", verification.ID),
			slog.Any("backup_id", verification.BackupID),
		)
	}
}

func (s *Service) notifyTerminal(
	ctx context.Context,
	verification *Verification,
	terminalStatus VerificationStatus,
	failMessage *string,
) {
	if terminalStatus == VerificationStatusCanceled {
		return
	}

	config, err := s.configService.GetConfigWithoutAuth(ctx, verification.DatabaseID)
	if err != nil {
		s.logger.ErrorContext(ctx,
			"failed to load verification config for notification",
			slog.Any("error", err),
			slog.Any("verification_id", verification.ID),
		)

		return
	}

	wantedType := configs.NotificationVerificationSuccess
	if terminalStatus == VerificationStatusFailed {
		wantedType = configs.NotificationVerificationFailed
	}

	if !slices.Contains(config.NotificationTriggers, wantedType) {
		return
	}

	database, err := s.getDatabase(ctx, verification.DatabaseID)
	if err != nil {
		s.logger.ErrorContext(ctx,
			"failed to load database for notification",
			slog.Any("error", err),
			slog.Any("verification_id", verification.ID),
		)

		return
	}

	title, body := buildNotificationCopy(verification, database, terminalStatus, failMessage)

	sentNotificationType := notifiers.CategoryVerificationSuccess
	if wantedType == configs.NotificationVerificationFailed {
		sentNotificationType = notifiers.CategoryVerificationFailed
	}

	s.notifierService.Notify(ctx, database.NotifierIDs, notifiers.Notification{
		Category: sentNotificationType,
		Heading:  title,
		Message:  body,
	})
}

func buildNotificationCopy(
	verification *Verification,
	database databases.Database,
	terminalStatus VerificationStatus,
	failMessage *string,
) (title, body string) {
	workspaceLabel := fmt.Sprintf(" (workspace %q)", database.WorkspaceID.String())

	if terminalStatus == VerificationStatusCompleted {
		title = fmt.Sprintf("backup verification passed for %q%s", database.Name, workspaceLabel)
	} else {
		title = fmt.Sprintf(
			"backup verification failed after %d attempts for %q%s",
			verification.AttemptCount, database.Name, workspaceLabel,
		)
	}

	bodyParts := []string{
		fmt.Sprintf("verification id: %s", verification.ID),
		fmt.Sprintf("backup id: %s", verification.BackupID),
		fmt.Sprintf("attempt: %d", verification.AttemptCount),
	}

	if verification.RestoreDurationMs != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("restore duration: %d ms", *verification.RestoreDurationMs))
	}
	if verification.VerifyDurationMs != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("verify duration: %d ms", *verification.VerifyDurationMs))
	}
	if verification.RestoredDatabaseSizeBytes != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("db size: %d bytes", *verification.RestoredDatabaseSizeBytes))
	}
	if verification.TableCount != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("table count: %d", *verification.TableCount))
	}

	if failMessage != nil && *failMessage != "" {
		messageRunes := []rune(*failMessage)
		if len(messageRunes) > 2000 {
			messageRunes = messageRunes[:2000]
		}

		bodyParts = append(bodyParts, fmt.Sprintf("error: %s", string(messageRunes)))
	}

	body = ""
	for i, part := range bodyParts {
		if i > 0 {
			body += "\n"
		}

		body += part
	}

	return title, body
}
