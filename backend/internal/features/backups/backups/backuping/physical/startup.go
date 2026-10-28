package backuping_physical

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups"
)

func RunStartupWork(ctx context.Context) error {
	scheduler := NewPhysicalBackupsScheduler()
	logger := scheduler.logger.With("job_id", uuid.New(), "job_name", schedulerJobName)
	if err := scheduler.recoverInFlightBackupsOnRestart(ctx, logger); err != nil {
		return fmt.Errorf("recover in-flight physical backups on restart: %w", err)
	}
	if err := NewPhysicalWalStreamSupervisor(
		backups.ProductionWalStreamTimings(),
	).recoverStreamersOnStartup(); err != nil {
		return fmt.Errorf("recover stale WAL streamers on restart: %w", err)
	}
	NewPhysicalBackupCleaner().runStartupSlotCleanup(ctx)

	return nil
}
