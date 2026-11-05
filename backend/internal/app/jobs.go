package app

import (
	"context"
	"log/slog"
	"time"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
)

const auditEventsCleanupInterval = time.Hour

func newAuditEventsCleanupJob(auditLog *audit.Log, logger *slog.Logger) background.Job {
	return background.Job{
		Name:     "audit_events_cleanup",
		Interval: auditEventsCleanupInterval,
		Run: func(ctx context.Context, now time.Time) error {
			deletedCount, err := auditLog.DeleteExpired(ctx, now)
			if err != nil {
				return err
			}

			logger.InfoContext(ctx, "expired audit events deleted", slog.Int64("deleted_count", deletedCount))

			return nil
		},
	}
}
