package app

import (
	"context"

	"databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_download "databasus-backend/internal/features/backups/backups/download"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	"databasus-backend/internal/features/restores/restoring"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/features/telemetry"
	users_services "databasus-backend/internal/features/users/services"
	verification_runs "databasus-backend/internal/features/verification/runs"
)

type backgroundLoop struct {
	name string
	run  func(ctx context.Context)
}

func (a *App) startBackgroundLoops(ctx context.Context) {
	loops := []backgroundLoop{
		{"backup background service", backuping_logical.GetBackupsScheduler().Run},
		{"verification scheduler", verification_runs.GetVerificationScheduler().Run},
		{"backup cleaner background service", backuping_logical.GetBackupCleaner().Run},
		{"physical backup scheduler background service", backuping_physical.GetPhysicalBackupsScheduler().Run},
		{"physical backup cleaner background service", backuping_physical.GetPhysicalBackupCleaner().Run},
		{"restore background service", restoring.GetRestoresScheduler().Run},
		{"healthcheck attempt background service", healthcheck_attempt.GetHealthcheckAttemptBackgroundService().Run},
		{"audit log cleanup background service", audit_logs.GetAuditLogBackgroundService().Run},
		{"download token cleanup background service", backups_download.GetDownloadTokenBackgroundService().Run},
		{"physical wal stream supervisor background service", backuping_physical.GetPhysicalWalStreamSupervisor().Run},
		{"storage file deletion background service", storages.GetStorageFileDeletionWorker().Run},
		{"sign-in code cleanup background service", users_services.GetSignInCodeBackgroundService().Run},
		{"telemetry background service", telemetry.GetTelemetryBackgroundService().Run},
	}

	a.logger.InfoContext(ctx, "starting background tasks")

	for _, loop := range loops {
		a.backgroundGroup.Go(ctx, loop.name, func(loopCtx context.Context) error {
			loop.run(loopCtx)

			return nil
		})
	}
}
