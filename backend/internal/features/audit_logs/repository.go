package audit_logs

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/storage"
)

// AuditLogRepository reads lib_audit_events directly, the one exception to lib/audit owning its
// table, until the workspaces module takes over the audit log views.
type AuditLogRepository struct{}

const selectAuditLogs = `
	SELECT
		al.id,
		al.author_user_id AS user_id,
		al.workspace_id,
		al.message,
		al.created_at,
		al.author_email AS user_email,
		al.author_name AS user_name,
		w.name AS workspace_name
	FROM lib_audit_events al
	LEFT JOIN workspaces w ON al.workspace_id = w.id`

func (r *AuditLogRepository) GetGlobal(
	ctx context.Context,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	auditLogs := make([]*AuditLogDTO, 0)

	sql := selectAuditLogs
	args := []any{}

	if beforeDate != nil {
		sql += " WHERE al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().WithContext(ctx).Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) GetByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	auditLogs := make([]*AuditLogDTO, 0)

	sql := selectAuditLogs + " WHERE al.author_user_id = ?"
	args := []any{userID}

	if beforeDate != nil {
		sql += " AND al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().WithContext(ctx).Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) GetByWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	limit, offset int,
	beforeDate *time.Time,
) ([]*AuditLogDTO, error) {
	auditLogs := make([]*AuditLogDTO, 0)

	sql := selectAuditLogs + " WHERE al.workspace_id = ?"
	args := []any{workspaceID}

	if beforeDate != nil {
		sql += " AND al.created_at < ?"
		args = append(args, *beforeDate)
	}

	sql += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := storage.GetDb().WithContext(ctx).Raw(sql, args...).Scan(&auditLogs).Error

	return auditLogs, err
}

func (r *AuditLogRepository) CountGlobal(ctx context.Context, beforeDate *time.Time) (int64, error) {
	sql := "SELECT COUNT(*) FROM lib_audit_events"
	args := []any{}

	if beforeDate != nil {
		sql += " WHERE created_at < ?"
		args = append(args, *beforeDate)
	}

	var count int64
	err := storage.GetDb().WithContext(ctx).Raw(sql, args...).Scan(&count).Error

	return count, err
}
