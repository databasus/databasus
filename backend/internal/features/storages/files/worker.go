package storage_files

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	// math/rand only adds retry jitter here, not anything secret.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

const (
	jobName = "storage_file_deletion"

	maxPersistedErrorLength = 500
)

// Drains the obligations nobody claimed. It never touches a file whose write is
// still running in this process, and it commits its claim before it talks to a
// provider, so a stalled provider call holds no database lock.
type DeletionWorker struct {
	store          *Store
	repository     *PendingDeletionRepository
	locator        StorageLocator
	fieldEncryptor encryption.FieldEncryptor
	logger         *slog.Logger
	timings        Timings

	// jitter is injectable so a test can assert the backoff instead of the noise.
	jitter func(time.Duration) time.Duration

	hasRun atomic.Bool
}

func NewDeletionWorker(store *Store, dependencies Dependencies) *DeletionWorker {
	return &DeletionWorker{
		store:          store,
		repository:     dependencies.Repository,
		locator:        dependencies.Locator,
		fieldEncryptor: dependencies.FieldEncryptor,
		logger:         dependencies.Logger,
		timings:        dependencies.Timings,
		jitter:         defaultJitter,
	}
}

func (w *DeletionWorker) Run(ctx context.Context) {
	if w.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", w))
	}

	lifecycleLogger := w.logger.With("job_name", jobName)

	lifecycleLogger.InfoContext(ctx, "storage file deletion started")

	if ctx.Err() != nil {
		return
	}

	w.RunOnce(ctx)

	ticker := time.NewTicker(w.timings.WorkerTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "storage file deletion stopped")

			return
		case <-ticker.C:
			w.RunOnce(ctx)
		}
	}
}

// One pass over the due obligations. Exported so a test can observe cleanup as a
// sequence instead of waiting on the ticker.
func (w *DeletionWorker) RunOnce(ctx context.Context) {
	passLogger := w.logger.With("job_id", uuid.New(), "job_name", jobName)

	claimed, err := w.claimDue()
	if err != nil {
		passLogger.ErrorContext(ctx, "failed to claim pending file deletions", "error", err)

		return
	}

	for _, pending := range claimed {
		w.attemptDeletion(ctx, passLogger, pending)
	}

	w.reportSummary(ctx, passLogger)
}

// DrainFiles removes the files of a storage that is being deleted, because its
// obligations go with its row and this is the last moment its credentials exist.
// The caller hands over the remover and the list, since the storage row is already
// gone from the deleting transaction's view. A file whose write is still running in
// this process is skipped, so a deletion never pulls a file from under its writer.
// It returns the names it skipped or could not remove, which the caller has to tell
// the user about. The budget stops it starting more files, because it runs in the
// request that deletes the storage; a call already under way runs to the
// provider's own deadline.
func (w *DeletionWorker) DrainFiles(ctx context.Context, request DrainRequest) []string {
	liveWriteIDs := w.store.getLiveWriteIDs()

	deadline := time.Now().Add(request.Budget)
	remaining := make([]string, 0, len(request.PendingDeletions))

	for _, pending := range request.PendingDeletions {
		if _, isLiveWrite := liveWriteIDs[pending.ID]; isLiveWrite || time.Now().After(deadline) {
			remaining = append(remaining, pending.FileName)

			continue
		}

		attemptCtx, cancel := context.WithTimeout(ctx, w.timings.AttemptLease)
		deleteErr := request.Remover.DeleteFile(
			attemptCtx, w.fieldEncryptor, w.logger.With("storage_id", pending.StorageID), pending.FileName,
		)

		cancel()

		if deleteErr != nil {
			w.logger.WarnContext(ctx, "failed to remove a file of a deleted storage",
				"storage_id", pending.StorageID, "file_name", pending.FileName, "error", deleteErr)

			remaining = append(remaining, pending.FileName)
		}
	}

	return remaining
}

func (w *DeletionWorker) claimDue() ([]PendingDeletion, error) {
	var claimed []PendingDeletion

	err := w.store.withLiveWrites(func(excluded []uuid.UUID) error {
		batch, claimErr := w.repository.ClaimDue(db.GetDb(), ClaimRequest{
			Limit:        w.timings.ClaimBatchSize,
			AttemptLease: w.timings.AttemptLease,
			ExcludedIDs:  excluded,
		})
		claimed = batch

		return claimErr
	})

	return claimed, err
}

func (w *DeletionWorker) attemptDeletion(ctx context.Context, passLogger *slog.Logger, pending PendingDeletion) {
	attemptLogger := passLogger.With("storage_id", pending.StorageID, "file_name", pending.FileName)

	attemptCtx, cancel := context.WithTimeout(ctx, w.timings.AttemptLease)
	defer cancel()

	err := w.deleteFromProvider(attemptCtx, attemptLogger, pending)
	if err == nil {
		released, completeErr := w.repository.CompleteIfGeneration(
			db.GetDb(), pending.ID, pending.Generation,
		)
		if completeErr != nil {
			attemptLogger.ErrorContext(ctx, "failed to release a completed file deletion", "error", completeErr)

			return
		}

		if !released {
			attemptLogger.InfoContext(ctx, fmt.Sprintf(
				"deleted stored file on attempt %d, a newer request now owns it", pending.AttemptCount))

			return
		}

		attemptLogger.InfoContext(ctx, fmt.Sprintf("deleted stored file on attempt %d", pending.AttemptCount))

		return
	}

	retryDelay := w.retryDelay(pending.AttemptCount)

	rescheduled, rescheduleErr := w.repository.RescheduleIfGeneration(
		db.GetDb(), pending.ID, pending.Generation, retryDelay, sanitizeError(err),
	)
	if rescheduleErr != nil {
		attemptLogger.ErrorContext(ctx, "failed to reschedule a file deletion", "error", rescheduleErr)

		return
	}

	if !rescheduled {
		attemptLogger.WarnContext(ctx, fmt.Sprintf(
			"file deletion attempt %d failed, a newer request now owns it", pending.AttemptCount), "error", err)

		return
	}

	attemptLogger.WarnContext(ctx, fmt.Sprintf(
		"file deletion attempt %d failed, retrying in %s", pending.AttemptCount, retryDelay), "error", err)
}

func (w *DeletionWorker) deleteFromProvider(
	ctx context.Context,
	attemptLogger *slog.Logger,
	pending PendingDeletion,
) error {
	remover, err := w.locator.GetFileRemover(ctx, pending.StorageID)
	if err != nil {
		return fmt.Errorf("locate storage for deletion: %w", err)
	}

	return remover.DeleteFile(ctx, w.fieldEncryptor, attemptLogger, pending.FileName)
}

func (w *DeletionWorker) reportSummary(ctx context.Context, passLogger *slog.Logger) {
	summary, err := w.repository.GetPendingDeletionSummary(db.GetDb())
	if err != nil {
		passLogger.ErrorContext(ctx, "failed to summarize pending file deletions", "error", err)

		return
	}

	if summary.Pending == 0 {
		return
	}

	passLogger.InfoContext(ctx, fmt.Sprintf(
		"pending file deletions: %d pending, %d overdue, oldest %s",
		summary.Pending, summary.Overdue, time.Duration(summary.OldestAgeMs)*time.Millisecond))
}

// Growth stops at the maximum, so an unreachable provider settles into one attempt
// per maximum delay rather than an ever-widening gap.
func (w *DeletionWorker) retryDelay(attemptCount int) time.Duration {
	if attemptCount < 1 {
		attemptCount = 1
	}

	growth := math.Pow(2, float64(attemptCount-1))

	delay := time.Duration(float64(w.timings.RetryBaseDelay) * growth)
	if delay > w.timings.RetryMaxDelay || delay <= 0 {
		delay = w.timings.RetryMaxDelay
	}

	return w.jitter(delay)
}

func defaultJitter(delay time.Duration) time.Duration {
	return delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
}

// Truncation counts runes, because cutting a multi-byte character in half makes
// Postgres reject the whole update and lose the backoff with it.
func sanitizeError(err error) string {
	message := []rune(logger.Redact(err.Error()))
	if len(message) > maxPersistedErrorLength {
		message = message[:maxPersistedErrorLength]
	}

	return string(message)
}
