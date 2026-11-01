package runcontext

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const LogicalTimeLimit = 23 * time.Hour

type Limits struct {
	Shutdown  error
	Timeout   error
	TimeLimit time.Duration
}

func New(appCtx context.Context, limits Limits) (context.Context, func(), func()) {
	runCtx, cancelRun := context.WithCancelCause(context.WithoutCancel(appCtx))
	stopWatchingShutdown := context.AfterFunc(appCtx, func() { cancelRun(limits.Shutdown) })

	stopTimeLimit := func() {}
	if limits.TimeLimit > 0 {
		timer := time.AfterFunc(limits.TimeLimit, func() { cancelRun(limits.Timeout) })
		stopTimeLimit = func() { timer.Stop() }
	}

	context.AfterFunc(runCtx, func() {
		stopWatchingShutdown()
		stopTimeLimit()
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
