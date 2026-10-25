package audit_logs

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users"
)

type AuditLogService struct {
	auditLogRepository *AuditLogRepository
	userRepository     *users_repositories.UserRepository
	logger             *slog.Logger
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

func (s *AuditLogService) GetGlobalAuditLogs(
	ctx context.Context,
	actor users.Actor,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	if actor.Role != users.RoleAdmin {
		return nil, ErrOnlyAdminsCanViewGlobalLogs
	}

	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetGlobal(ctx, limit, offset, request.BeforeDate)
	if err != nil {
		return nil, err
	}

	total, err := s.auditLogRepository.CountGlobal(ctx, request.BeforeDate)
	if err != nil {
		return nil, err
	}

	return &GetAuditLogsResponse{
		AuditLogs: auditLogs,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (s *AuditLogService) GetUserAuditLogs(
	ctx context.Context,
	targetUserID uuid.UUID,
	actor users.Actor,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	// Users can view their own logs, ADMIN can view any user's logs
	if actor.Role != users.RoleAdmin && actor.UserID != targetUserID {
		return nil, ErrInsufficientPermissionsToViewLogs
	}

	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetByUser(
		ctx,
		targetUserID,
		limit,
		offset,
		request.BeforeDate,
	)
	if err != nil {
		return nil, err
	}

	return &GetAuditLogsResponse{
		AuditLogs: auditLogs,
		Total:     int64(len(auditLogs)),
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (s *AuditLogService) GetWorkspaceAuditLogs(
	ctx context.Context,
	workspaceID uuid.UUID,
	request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset := max(request.Offset, 0)

	auditLogs, err := s.auditLogRepository.GetByWorkspace(
		ctx,
		workspaceID,
		limit,
		offset,
		request.BeforeDate,
	)
	if err != nil {
		return nil, err
	}

	return &GetAuditLogsResponse{
		AuditLogs: auditLogs,
		Total:     int64(len(auditLogs)),
		Limit:     limit,
		Offset:    offset,
	}, nil
}

// findAuthor returns nil for an entry without a user and for an account that is gone, which only a
// test that deletes the account before the write can cause; the event is then kept without an
// author rather than lost.
func (s *AuditLogService) findAuthor(ctx context.Context, userID *uuid.UUID) *audit.Author {
	if userID == nil {
		return nil
	}

	user, err := s.userRepository.GetUserByID(ctx, *userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.WarnContext(ctx, "audit author not found, recording the event without an author",
			"user_id", userID)

		return nil
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to read the audit author, recording the event without an author",
			"error", err, "user_id", userID)

		return nil
	}

	return &audit.Author{UserID: user.ID, Email: user.Email, Name: user.Name}
}
