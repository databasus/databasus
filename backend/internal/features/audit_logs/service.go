package audit_logs

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users"
)

type AuditLogService struct {
	// getUsersModule is read per call: each test application installs its own module.
	getUsersModule func() users.Module
	logger         *slog.Logger
}

// WriteAuditLog records the entry through lib/audit with its author's email and name as they are
// now. The callers pass only the user ID, so the bridge looks the account up. The write ignores the
// caller's cancellation: the event of an action that already happened must be kept even if the
// client disconnected mid-request.
func (s *AuditLogService) WriteAuditLog(ctx context.Context, entry audit_logs_models.AuditEntry) {
	ctx = context.WithoutCancel(ctx)

	err := getAuditLog().Record(ctx, audit.Event{
		Author:      s.findAuthor(ctx, entry.UserID),
		WorkspaceID: entry.WorkspaceID,
		Message:     entry.Message,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to record audit event",
			"error", err, "user_id", entry.UserID, "workspace_id", entry.WorkspaceID)
	}
}

// findAuthor returns nil for an entry without a user and for an account that is gone, which only a
// test that deletes the account before the write can cause; the event is then kept without an
// author rather than lost.
func (s *AuditLogService) findAuthor(ctx context.Context, userID *uuid.UUID) *audit.Author {
	if userID == nil {
		return nil
	}

	user, isFound, err := s.getUsersModule().GetUser(ctx, *userID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to read the audit author, recording the event without an author",
			"error", err, "user_id", userID)

		return nil
	}

	if !isFound {
		s.logger.WarnContext(ctx, "audit author not found, recording the event without an author",
			"user_id", userID)

		return nil
	}

	return &audit.Author{UserID: user.ID, Email: user.Email, Name: user.Name}
}
