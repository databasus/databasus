package files

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	// math/rand only adds retry jitter here, not anything secret.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand/v2"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
)

const (
	DeletionJobName = "storage_file_deletion"

	maxPersistedErrorLength = 500
)

// DrainRequest names the files of a storage that is being deleted. PrepareDrain builds it before the
// row goes, since the row is gone from the deleting transaction's view afterwards.
type DrainRequest struct {
	Storage          LocatedStorage
	PendingDeletions []PendingDeletion
	Budget           time.Duration
}

// DeletionJob drains the obligations nobody claimed. It never touches a file whose write is still
// running in this process, and it commits its claim before it talks to a provider, so a stalled
// provider call holds no database lock.
type DeletionJob struct {
	store      *Store
	repository *PendingDeletionRepository
	batches    *DeletionBatchRepository
	transactor *database.Transactor
	locator    StorageLocator
	logger     *slog.Logger
	clock      clock.Source
	timings    Timings

	// jitter is replaceable so a test can assert the backoff instead of the noise.
	jitter func(time.Duration) time.Duration
}

func NewDeletionJob(store *Store, deps Deps) *DeletionJob {
	return &DeletionJob{
		store:      store,
		repository: deps.Repository,
		batches:    deps.Batches,
		transactor: deps.Transactor,
		locator:    deps.Locator,
		logger:     deps.Logger,
		clock:      deps.Clock,
		timings:    deps.Timings,
		jitter:     getJitteredDelay,
	}
}

func (j *DeletionJob) GetJob() background.Job {
	return background.Job{Name: DeletionJobName, Interval: j.timings.WorkerTick, Run: j.RunPass}
}

// RunPass makes one pass over the due obligations.
func (j *DeletionJob) RunPass(ctx context.Context, now time.Time) error {
	err := j.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := j.batches.Expire(ctx, now); err != nil {
			return err
		}
		return j.batches.RemoveUnownedPending(ctx)
	})
	if err != nil {
		return err
	}
	claimed, err := j.claimDue(ctx, now)
	if err != nil {
		return err
	}

	for _, pending := range claimed {
		j.attemptDeletion(ctx, pending)
	}

	j.reportSummary(ctx)

	return nil
}

// PrepareDrain locates the storage and reads its pending deletions. The caller runs it before it
// deletes the storage's row, because both go with the row.
func (j *DeletionJob) PrepareDrain(
	ctx context.Context,
	storageID uuid.UUID,
	budget time.Duration,
) (DrainRequest, error) {
	located, err := j.locator.LocateStorage(ctx, storageID)
	if err != nil {
		return DrainRequest{}, err
	}

	pendingDeletions, err := j.repository.ListByStorage(ctx, storageID)
	if err != nil {
		return DrainRequest{}, err
	}
	remainingFileNames, err := j.batches.ListRemainingFiles(ctx, storageID)
	if err != nil {
		return DrainRequest{}, err
	}
	for _, fileName := range remainingFileNames {
		if slices.ContainsFunc(
			pendingDeletions,
			func(pending PendingDeletion) bool { return pending.FileName == fileName },
		) {
			continue
		}
		var remaining PendingDeletion
		remaining.StorageID, remaining.FileName = storageID, fileName
		pendingDeletions = append(pendingDeletions, remaining)
	}

	return DrainRequest{Storage: located, PendingDeletions: pendingDeletions, Budget: budget}, nil
}

// DrainFiles removes the files of a storage that is being deleted, because its obligations go with
// its row and this is the last moment its credentials exist. A file whose write is still running in
// this process is skipped, so a deletion never pulls a file from under its writer. It returns the
// names it skipped or could not remove, which the caller has to tell the user about. The budget,
// measured on the module's clock, stops it starting more files, because it runs in the request that
// deletes the storage; a call already under way runs to the provider's own deadline.
func (j *DeletionJob) DrainFiles(ctx context.Context, request DrainRequest) []string {
	liveWriteIDs := j.store.getLiveWriteIDs()

	deadline := j.clock.Now().Add(request.Budget)
	remainingFileNames := make([]string, 0, len(request.PendingDeletions))

	for _, pending := range request.PendingDeletions {
		if _, isLiveWrite := liveWriteIDs[pending.ID]; isLiveWrite || j.clock.Now().After(deadline) {
			remainingFileNames = append(remainingFileNames, pending.FileName)

			continue
		}
		canDelete, protectionErr := j.store.writeProtection.TryLockFileForDeletion(ctx, pending.GetReference())
		if protectionErr != nil || !canDelete {
			remainingFileNames = append(remainingFileNames, pending.FileName)
			if protectionErr != nil {
				j.logger.WarnContext(
					ctx,
					"failed to check storage file write protection",
					slog.Any("error", protectionErr),
				)
			}
			continue
		}

		attemptCtx, cancel := context.WithTimeout(ctx, j.timings.AttemptLease)
		deleteErr := request.Storage.Provider.DeleteFile(
			attemptCtx, request.Storage.getFileAccess(pending.FileName, j.logger),
		)

		cancel()

		if deleteErr != nil {
			j.logger.WarnContext(ctx, "failed to remove a file of a deleted storage",
				slog.String("storage_id", pending.StorageID.String()),
				slog.String("file_name", pending.FileName),
				slog.Any("error", deleteErr))

			remainingFileNames = append(remainingFileNames, pending.FileName)
		}
	}

	return remainingFileNames
}

func (j *DeletionJob) SettleStorageDeletion(
	ctx context.Context,
	request DrainRequest,
	remainingFileNames []string,
) error {
	confirmedFiles := []string{}
	for _, pending := range request.PendingDeletions {
		if !slices.Contains(remainingFileNames, pending.FileName) {
			confirmedFiles = append(confirmedFiles, pending.FileName)
		}
	}
	return j.batches.SettleStorage(
		ctx,
		StorageDeletionSettlement{
			StorageID:      request.Storage.Location.StorageID,
			ConfirmedFiles: confirmedFiles,
			Now:            j.clock.Now(),
		},
	)
}

func (j *DeletionJob) claimDue(ctx context.Context, now time.Time) ([]PendingDeletion, error) {
	var claimed []PendingDeletion

	err := j.store.withLiveWrites(func(excludedIDs []uuid.UUID) error {
		batch, claimErr := j.repository.ClaimDue(ctx, ClaimRequest{
			Limit:        j.timings.ClaimBatchSize,
			AttemptLease: j.timings.AttemptLease,
			ExcludedIDs:  excludedIDs,
			Now:          now,
		})
		claimed = batch

		return claimErr
	})

	return claimed, err
}

func (j *DeletionJob) attemptDeletion(ctx context.Context, pending PendingDeletion) {
	ctx = logging.With(ctx,
		slog.String("storage_id", pending.StorageID.String()),
		slog.String("file_name", pending.FileName),
		slog.Int("attempt", pending.AttemptCount))

	duration, err := j.prepareAttempt(ctx, pending)
	if err != nil {
		j.logger.ErrorContext(ctx, "failed to prepare deletion attempt", slog.Any("error", err))
		return
	}
	if duration <= 0 {
		return
	}
	attemptCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	err = j.deleteFromProvider(attemptCtx, pending)
	if err == nil {
		j.releaseDeletedFile(ctx, pending)

		return
	}

	retryDelay := j.getRetryDelay(pending.AttemptCount)
	now := j.clock.Now()

	var isRescheduled bool
	rescheduleErr := j.transactor.Transaction(ctx, func(ctx context.Context) error {
		isCurrent, lockErr := j.repository.LockAttempt(
			ctx,
			AttemptLock{Pending: pending, Now: now, ShouldAllowExpired: true},
		)
		if lockErr != nil {
			return lockErr
		}
		if batchErr := j.batches.FailFile(ctx, pending, now); batchErr != nil {
			return batchErr
		}
		if !isCurrent {
			if batchErr := j.batches.Expire(ctx, now); batchErr != nil {
				return batchErr
			}
			return j.batches.RemoveUnownedPending(ctx)
		}
		var updateErr error
		isRescheduled, updateErr = j.repository.RescheduleIfGeneration(
			ctx,
			Rescheduling{
				PendingDeletionID: pending.ID,
				Generation:        pending.Generation,
				NotBefore:         now.Add(retryDelay),
				Now:               now,
			},
			sanitizeError(err),
		)
		if updateErr != nil {
			return updateErr
		}
		if batchErr := j.batches.Expire(ctx, now); batchErr != nil {
			return batchErr
		}
		return j.batches.RemoveUnownedPending(ctx)
	})
	if rescheduleErr != nil {
		j.logger.ErrorContext(ctx, "failed to reschedule a file deletion", slog.Any("error", rescheduleErr))

		return
	}

	if !isRescheduled {
		j.logger.WarnContext(ctx, "file deletion attempt failed, a newer request now owns it", slog.Any("error", err))

		return
	}

	j.logger.WarnContext(ctx, "file deletion attempt failed, retrying",
		slog.Duration("retry_in", retryDelay),
		slog.Any("error", err))
}

func (j *DeletionJob) releaseDeletedFile(ctx context.Context, pending PendingDeletion) {
	var isReleased bool
	err := j.transactor.Transaction(ctx, func(ctx context.Context) error {
		now := j.clock.Now()
		isCurrent, lockErr := j.repository.LockAttempt(
			ctx,
			AttemptLock{Pending: pending, Now: now, ShouldAllowExpired: false},
		)
		if lockErr != nil || !isCurrent {
			return lockErr
		}
		if batchErr := j.batches.CompleteFile(ctx, pending, now); batchErr != nil {
			return batchErr
		}
		var deleteErr error
		isReleased, deleteErr = j.repository.CompleteIfGeneration(ctx, pending.ID, pending.Generation)
		return deleteErr
	})
	if err != nil {
		j.logger.ErrorContext(ctx, "failed to release a completed file deletion", slog.Any("error", err))

		return
	}

	if !isReleased {
		j.logger.InfoContext(ctx, "deleted stored file, a newer request now owns it")

		return
	}

	j.logger.InfoContext(ctx, "deleted stored file")
}

func (j *DeletionJob) deleteFromProvider(ctx context.Context, pending PendingDeletion) error {
	located, err := j.locator.LocateStorage(ctx, pending.StorageID)
	if err != nil {
		return fmt.Errorf("locate storage for deletion: %w", err)
	}

	return located.Provider.DeleteFile(ctx, located.getFileAccess(pending.FileName, j.logger))
}

func (j *DeletionJob) reportSummary(ctx context.Context) {
	summary, err := j.repository.GetPendingDeletionSummary(ctx, j.clock.Now())
	if err != nil {
		j.logger.ErrorContext(ctx, "failed to summarize pending file deletions", slog.Any("error", err))

		return
	}

	if summary.Pending == 0 {
		return
	}

	j.logger.InfoContext(ctx, "pending file deletions remain",
		slog.Int64("pending_count", summary.Pending),
		slog.Int64("overdue_count", summary.Overdue),
		slog.Duration("oldest_age", time.Duration(summary.OldestAgeMs)*time.Millisecond))
}

// getRetryDelay stops growing at the maximum, so an unreachable provider settles into one attempt per
// maximum delay rather than an ever-widening gap.
func (j *DeletionJob) getRetryDelay(attemptCount int) time.Duration {
	attemptCount = max(attemptCount, 1)

	growth := math.Pow(2, float64(attemptCount-1))

	delay := time.Duration(float64(j.timings.RetryBaseDelay) * growth)
	if delay > j.timings.RetryMaxDelay || delay <= 0 {
		delay = j.timings.RetryMaxDelay
	}

	return j.jitter(delay)
}

func getJitteredDelay(delay time.Duration) time.Duration {
	return delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
}

// sanitizeError counts runes, because cutting a multi-byte character in half makes Postgres reject
// the whole update and lose the backoff with it.
func sanitizeError(err error) string {
	message := []rune(logging.Redact(err.Error()))
	if len(message) > maxPersistedErrorLength {
		message = message[:maxPersistedErrorLength]
	}

	return string(message)
}

func (j *DeletionJob) prepareAttempt(ctx context.Context, pending PendingDeletion) (time.Duration, error) {
	var duration time.Duration
	err := j.transactor.Transaction(ctx, func(ctx context.Context) error {
		now := j.clock.Now()
		isCurrent, err := j.repository.LockAttempt(
			ctx,
			AttemptLock{Pending: pending, Now: now, ShouldAllowExpired: false},
		)
		if err != nil || !isCurrent {
			return err
		}
		duration, err = j.batches.ClaimFile(ctx, pending, now)
		return err
	})
	return duration, err
}
