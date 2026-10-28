package backuping

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/databases"
)

type SlotCleanupSpec struct {
	Databases databases.Module
	Logger    *slog.Logger
}

func DropOrphanBackupSlots(ctx context.Context, spec SlotCleanupSpec) error {
	listedDatabases, err := spec.Databases.ListAllDatabases(ctx)
	if err != nil {
		return fmt.Errorf("list databases for slot cleanup: %w", err)
	}

	runs := background.New(background.Deps{Logger: spec.Logger})
	semaphore := make(chan struct{}, startupCleanupConcurrency)
	var droppedCount, skippedCount, failedCount atomic.Int64

	for _, backupDatabase := range listedDatabases {
		if backupDatabase.Type != databases.TypePostgresqlPhysical {
			continue
		}

		connectionSettings, err := spec.Databases.GetPostgresqlPhysicalConnectionSettings(ctx, backupDatabase.ID)
		if err != nil {
			spec.Logger.WarnContext(ctx, "cannot read connection settings for slot cleanup",
				slog.Any("database_id", backupDatabase.ID), slog.Any("error", err))
			skippedCount.Add(1)
			continue
		}

		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}

		runs.Go(ctx, "drop orphan backup slot", func(ctx context.Context) error {
			defer func() { <-semaphore }()
			cleanupCtx, cancel := context.WithTimeout(ctx, startupCleanupTimeout)
			defer cancel()

			logger := spec.Logger.With(slog.Any("database_id", connectionSettings.DatabaseID))
			tunnel, err := pgphysical.NewSource(connectionSettings.Engine).OpenTunnel(cleanupCtx, logger)
			if err != nil {
				logger.WarnContext(ctx, "cannot reach bastion for slot cleanup", slog.Any("error", err))
				skippedCount.Add(1)
				return nil
			}
			defer tunnel.Close()

			inspection, err := tunnel.GetSource().OpenInspection(cleanupCtx)
			if err != nil {
				logger.WarnContext(ctx, "cannot reach source for slot cleanup", slog.Any("error", err))
				skippedCount.Add(1)
				return nil
			}
			defer func() { _ = inspection.Close(context.WithoutCancel(ctx)) }()

			if err := inspection.DropBackupSlotIfExists(cleanupCtx, connectionSettings.BackupSlotName); err != nil {
				logger.WarnContext(ctx, "cannot drop orphan backup slot", slog.Any("error", err))
				failedCount.Add(1)
				return nil
			}

			droppedCount.Add(1)
			return nil
		})
	}

	if err := runs.Wait(ctx); err != nil {
		return err
	}

	spec.Logger.InfoContext(ctx, "orphan backup slot cleanup completed",
		slog.Int64("dropped_count", droppedCount.Load()),
		slog.Int64("skipped_count", skippedCount.Load()),
		slog.Int64("failed_count", failedCount.Load()))
	return nil
}
