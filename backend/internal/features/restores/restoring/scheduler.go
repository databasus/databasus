package restoring

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/lib/cache"
)

const jobName = "restore_scheduler"

const (
	schedulerTickerInterval       = 1 * time.Minute
	schedulerHealthcheckThreshold = 5 * time.Minute
)

type RestoresScheduler struct {
	restoreRepository    *restores_core.RestoreRepository
	lastCheckTime        time.Time
	logger               *slog.Logger
	restorer             restoreExecutor
	restoreDatabaseCache *cache.JSONStore[RestoreDatabaseCache]

	hasRun atomic.Bool
}

type restoreExecutor interface {
	MakeRestore(ctx context.Context, restoreID uuid.UUID, databaseCache *RestoreDatabaseCache)
}

func (s *RestoresScheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	s.lastCheckTime = time.Now().UTC()

	runLogger := s.logger.With("job_id", uuid.New(), "job_name", jobName)

	if err := s.failRestoresInProgress(ctx, runLogger); err != nil {
		runLogger.ErrorContext(ctx, "failed to fail restores in progress", "error", err)
		panic(err)
	}

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(schedulerTickerInterval)
	defer ticker.Stop()

	runLogger.InfoContext(ctx, "restore scheduler started")

	for {
		select {
		case <-ctx.Done():
			runLogger.InfoContext(ctx, "restore scheduler stopped")

			return
		case <-ticker.C:
			s.lastCheckTime = time.Now().UTC()
		}
	}
}

func (s *RestoresScheduler) IsSchedulerRunning() bool {
	return s.lastCheckTime.After(time.Now().UTC().Add(-schedulerHealthcheckThreshold))
}

func (s *RestoresScheduler) StartRestore(
	ctx context.Context,
	restoreID uuid.UUID,
	databaseCache *RestoreDatabaseCache,
) error {
	if err := s.restoreDatabaseCache.SetWithLifetime(ctx, cache.ExpiringValue[RestoreDatabaseCache]{
		Key:      restoreID.String(),
		Value:    *databaseCache,
		Lifetime: time.Hour,
	}); err != nil {
		s.logger.WarnContext(ctx, "failed to cache restore metadata", "restore_id", restoreID, "error", err)
	}

	executionContext := context.WithoutCancel(ctx)
	go s.restorer.MakeRestore(executionContext, restoreID, databaseCache)

	s.logger.InfoContext(ctx, "triggered restore", "restore_id", restoreID)

	return nil
}

func (s *RestoresScheduler) failRestoresInProgress(ctx context.Context, logger *slog.Logger) error {
	restoresInProgress, err := s.restoreRepository.FindByStatus(
		restores_core.RestoreStatusInProgress,
	)
	if err != nil {
		return err
	}

	if len(restoresInProgress) > 0 {
		logger.InfoContext(ctx, fmt.Sprintf(
			"failing %d restores left in progress by the previous run", len(restoresInProgress)))
	}

	for _, restore := range restoresInProgress {
		failMessage := "Restore failed due to application restart"
		restore.FailMessage = &failMessage
		restore.Status = restores_core.RestoreStatusFailed

		if err := s.restoreRepository.Save(restore); err != nil {
			return err
		}

		logger.InfoContext(ctx, "failed a restore left in progress by the previous run",
			"restore_id", restore.ID)
	}

	return nil
}
