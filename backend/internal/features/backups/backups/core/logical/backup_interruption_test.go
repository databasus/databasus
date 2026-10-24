package backups_core_logical

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	test_utils "databasus-backend/internal/util/testing"
)

func Test_DescribeBackupInterruption_ReportsWhoInterruptedTheBackup(t *testing.T) {
	storageTimeoutErr := fmt.Errorf("failed to upload file to S3: %w", test_utils.GetNetworkTimeoutError(t))
	storageCancelledErr := fmt.Errorf("failed to upload file to S3: %w", context.Canceled)

	testCases := []struct {
		name                  string
		backupCancelCause     error
		isCallerCancelled     bool
		isShutdownSignalled   bool
		expectedErr           error
		expectedMessage       string
		expectedWrappedCause  error
		isReportedAsCancelled bool
	}{
		{
			name:              "Test_DescribeBackupInterruption_WhenShuttingDown_ReturnsShutdownError",
			backupCancelCause: ErrBackupShutdown,
			expectedErr:       ErrBackupShutdown,
			expectedMessage:   "backup interrupted by application shutdown",
		},
		{
			name:                "Test_DescribeBackupInterruption_WhenShutdownSignalCloses_ReturnsShutdownError",
			isShutdownSignalled: true,
			expectedErr:         ErrBackupShutdown,
			expectedMessage:     "backup interrupted by application shutdown",
		},
		{
			name:              "Test_DescribeBackupInterruption_WhenTimeLimitPassed_ReturnsTimeLimitError",
			backupCancelCause: ErrBackupTimeout,
			expectedErr:       ErrBackupTimeout,
			expectedMessage:   "backup exceeded the time limit of 23 hours",
		},
		{
			name:                  "Test_DescribeBackupInterruption_WhenCallerCancelled_ReturnsBackupCancelled",
			isCallerCancelled:     true,
			expectedMessage:       "backup cancelled",
			isReportedAsCancelled: true,
		},
		{
			name:                 "Test_DescribeBackupInterruption_WhenStorageTimedOut_ReturnsStorageError",
			backupCancelCause:    storageTimeoutErr,
			expectedMessage:      "save to storage: " + storageTimeoutErr.Error(),
			expectedWrappedCause: storageTimeoutErr,
		},
		{
			name:                 "Test_DescribeBackupInterruption_WhenStorageErrorWrapsCanceled_ReturnsStorageError",
			backupCancelCause:    storageCancelledErr,
			expectedMessage:      "save to storage: " + storageCancelledErr.Error(),
			expectedWrappedCause: storageCancelledErr,
		},
		{
			name:                  "Test_DescribeBackupInterruption_WhenCallerCancelledAfterStorageFailed_ReturnsBackupCancelled",
			backupCancelCause:     storageTimeoutErr,
			isCallerCancelled:     true,
			expectedMessage:       "backup cancelled",
			isReportedAsCancelled: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			callerCtx, cancelCaller := context.WithCancel(t.Context())
			defer cancelCaller()

			shutdownSignal := make(chan struct{})

			backupCtx, cancelBackup := NewBackupContext(callerCtx, shutdownSignal)
			defer cancelBackup(nil)

			if testCase.isShutdownSignalled {
				close(shutdownSignal)
			}

			if testCase.backupCancelCause != nil {
				cancelBackup(testCase.backupCancelCause)
			}

			if testCase.isCallerCancelled {
				cancelCaller()
			}

			<-backupCtx.Done()

			interruptionErr := DescribeBackupInterruption(callerCtx, backupCtx)

			assert.EqualError(t, interruptionErr, testCase.expectedMessage)

			if testCase.expectedErr != nil {
				assert.ErrorIs(t, interruptionErr, testCase.expectedErr)
			}

			if testCase.expectedWrappedCause != nil {
				assert.ErrorIs(t, interruptionErr, testCase.expectedWrappedCause)
			}

			if !testCase.isReportedAsCancelled {
				assert.NotEqual(t, "backup cancelled", interruptionErr.Error())
			}
		})
	}
}
