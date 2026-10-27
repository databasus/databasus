package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const runTimeLimit = 23 * time.Hour

// NewBackupRunContext derives the context one backup runs under from the application's context. It
// outlives the request that started the backup and ends with ErrBackupShutdown when the application
// stops, with ErrBackupTimeout after the time limit, and on cancel, which the cancellation registry
// calls. release ends it once the run has written its outcome.
func NewBackupRunContext(appCtx context.Context) (runCtx context.Context, cancel, release func()) {
	return newRunContext(
		appCtx,
		runEnds{shutdown: ErrBackupShutdown, timeout: ErrBackupTimeout, timeLimit: runTimeLimit},
	)
}

// NewRestoreRunContext is NewBackupRunContext for a restore.
func NewRestoreRunContext(appCtx context.Context) (runCtx context.Context, cancel, release func()) {
	return newRunContext(
		appCtx,
		runEnds{shutdown: ErrRestoreShutdown, timeout: ErrRestoreTimeout, timeLimit: runTimeLimit},
	)
}

// runEnds is how a run context ends besides a cancel: the cause of a shutdown and the cause and the
// duration of the time limit.
type runEnds struct {
	shutdown  error
	timeout   error
	timeLimit time.Duration
}

func newRunContext(appCtx context.Context, ends runEnds) (context.Context, func(), func()) {
	runCtx, cancelRun := context.WithCancelCause(context.WithoutCancel(appCtx))

	stopWatchingShutdown := context.AfterFunc(appCtx, func() { cancelRun(ends.shutdown) })
	timeLimit := time.AfterFunc(ends.timeLimit, func() { cancelRun(ends.timeout) })

	context.AfterFunc(runCtx, func() {
		stopWatchingShutdown()
		timeLimit.Stop()
	})

	return runCtx, func() { cancelRun(errCancelRequested) }, func() { cancelRun(errRunEnded) }
}

// DescribeBackupInterruption explains why the backup context ended. An ended run context comes first,
// because a cancellation, a shutdown or the time limit makes the storage fail too. Any other cause is
// the storage write's, which ends the backup context with its own error; that error can still match
// context.Canceled, so the shape of the cause decides nothing.
func DescribeBackupInterruption(runCtx, backupCtx context.Context) error {
	if runCtx.Err() == nil {
		return fmt.Errorf("save to storage: %w", context.Cause(backupCtx))
	}

	switch cause := context.Cause(runCtx); {
	case errors.Is(cause, ErrBackupShutdown):
		return ErrBackupShutdown
	case errors.Is(cause, ErrBackupTimeout):
		return ErrBackupTimeout
	default:
		return ErrBackupCancelled
	}
}

// DescribeRestoreInterruption explains why the restore's run context ended.
func DescribeRestoreInterruption(runCtx context.Context) error {
	switch cause := context.Cause(runCtx); {
	case errors.Is(cause, ErrRestoreShutdown):
		return ErrRestoreShutdown
	case errors.Is(cause, ErrRestoreTimeout):
		return ErrRestoreTimeout
	default:
		return ErrRestoreCancelled
	}
}
