package runcontext

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	test_utils "databasus-backend/internal/lib/httpclienttest"
)

func TestDescribeBackupInterruption(t *testing.T) {
	t.Parallel()

	storageTimeoutErr := fmt.Errorf("failed to upload file to S3: %w", test_utils.GetNetworkTimeoutError(t))
	storageCancelledErr := fmt.Errorf("failed to upload file to S3: %w", context.Canceled)

	testCases := []struct {
		name            string
		storageErr      error
		isCancelled     bool
		isAppStopped    bool
		expectedErr     error
		expectedMessage string
	}{
		{
			name:            "reports the shutdown when the application stops",
			isAppStopped:    true,
			expectedErr:     ErrBackupShutdown,
			expectedMessage: "backup interrupted by application shutdown",
		},
		{
			name:            "reports a cancelled backup when a cancel is requested",
			isCancelled:     true,
			expectedErr:     ErrBackupCancelled,
			expectedMessage: "backup cancelled",
		},
		{
			name:            "reports the storage error when the storage times out",
			storageErr:      storageTimeoutErr,
			expectedErr:     storageTimeoutErr,
			expectedMessage: "save to storage: " + storageTimeoutErr.Error(),
		},
		{
			name:            "reports the storage error when it wraps context canceled",
			storageErr:      storageCancelledErr,
			expectedErr:     storageCancelledErr,
			expectedMessage: "save to storage: " + storageCancelledErr.Error(),
		},
		{
			name:            "reports a cancelled backup when it is cancelled after the storage failed",
			storageErr:      storageTimeoutErr,
			isCancelled:     true,
			expectedErr:     ErrBackupCancelled,
			expectedMessage: "backup cancelled",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			appCtx, stopApp := context.WithCancel(t.Context())
			t.Cleanup(stopApp)
			runCtx, cancelRun, releaseRun := New(
				appCtx,
				Limits{Shutdown: ErrBackupShutdown, Timeout: ErrBackupTimeout, TimeLimit: LogicalTimeLimit},
			)
			t.Cleanup(releaseRun)
			backupCtx, cancelBackup := context.WithCancelCause(runCtx)
			t.Cleanup(func() { cancelBackup(nil) })

			if testCase.storageErr != nil {
				cancelBackup(testCase.storageErr)
			}

			if testCase.isCancelled {
				cancelRun()
			}

			if testCase.isAppStopped {
				stopApp()
			}

			<-backupCtx.Done()
			interruptionErr := DescribeBackupInterruption(runCtx, backupCtx)

			require.EqualError(t, interruptionErr, testCase.expectedMessage)
			require.ErrorIs(t, interruptionErr, testCase.expectedErr)
		})
	}

	t.Run("reports the time limit when the run passes it", func(t *testing.T) {
		t.Parallel()
		runCtx, _, releaseRun := New(t.Context(), Limits{
			Shutdown:  ErrBackupShutdown,
			Timeout:   ErrBackupTimeout,
			TimeLimit: time.Millisecond,
		})
		t.Cleanup(releaseRun)

		<-runCtx.Done()
		interruptionErr := DescribeBackupInterruption(runCtx, runCtx)

		require.EqualError(t, interruptionErr, "backup exceeded the time limit of 23 hours")
		require.ErrorIs(t, interruptionErr, ErrBackupTimeout)
	})
}

func TestDescribeRestoreInterruption(t *testing.T) {
	t.Parallel()

	t.Run("reports the shutdown when the application stops", func(t *testing.T) {
		t.Parallel()
		appCtx, stopApp := context.WithCancel(t.Context())
		runCtx, _, releaseRun := New(
			appCtx,
			Limits{Shutdown: ErrRestoreShutdown, Timeout: ErrRestoreTimeout, TimeLimit: LogicalTimeLimit},
		)
		t.Cleanup(releaseRun)

		stopApp()
		<-runCtx.Done()

		require.EqualError(t, DescribeRestoreInterruption(runCtx), "restore interrupted by application shutdown")
	})

	t.Run("reports a cancelled restore when a cancel is requested", func(t *testing.T) {
		t.Parallel()
		runCtx, cancelRun, releaseRun := New(
			t.Context(),
			Limits{Shutdown: ErrRestoreShutdown, Timeout: ErrRestoreTimeout, TimeLimit: LogicalTimeLimit},
		)
		t.Cleanup(releaseRun)

		cancelRun()
		<-runCtx.Done()

		require.ErrorIs(t, DescribeRestoreInterruption(runCtx), ErrRestoreCancelled)
	})

	t.Run("reports the time limit when the run passes it", func(t *testing.T) {
		t.Parallel()
		runCtx, _, releaseRun := New(t.Context(), Limits{
			Shutdown:  ErrRestoreShutdown,
			Timeout:   ErrRestoreTimeout,
			TimeLimit: time.Millisecond,
		})
		t.Cleanup(releaseRun)

		<-runCtx.Done()

		require.ErrorIs(t, DescribeRestoreInterruption(runCtx), ErrRestoreTimeout)
	})
}
