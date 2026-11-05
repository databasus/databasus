package backuping

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

func Test_ClassifyIncrementalStreamFailureAfterIdentity_WhenTimelineAdvanced_BreaksChain(t *testing.T) {
	originalReason := catalog.BackupErrorPgBasebackupFailed
	originalResult := Result{
		Status:      catalog.BackupStatusError,
		ErrorReason: &originalReason,
	}
	decision := &timelines.TimelineDecision{
		Kind:               timelines.TimelineFailoverDetected,
		ExpectedTimelineID: 3,
		LiveTimelineID:     4,
	}

	result := classifyIncrementalStreamFailureAfterIdentity(originalResult, decision)

	require.Equal(t, catalog.BackupStatusChainBroken, result.Status)
	require.Equal(t, catalog.BackupErrorTimelineSwitchDetected, *result.ErrorReason)
}

func Test_ClassifyFullStreamFailureAfterIdentity_WhenTimelineAdvanced_ClassifiesFailover(t *testing.T) {
	originalReason := catalog.BackupErrorPgBasebackupFailed
	originalResult := Result{
		Status:      catalog.BackupStatusError,
		ErrorReason: &originalReason,
	}
	decision := &timelines.TimelineDecision{
		Kind:               timelines.TimelineFailoverDetected,
		ExpectedTimelineID: 3,
		LiveTimelineID:     4,
	}

	result := classifyFullStreamFailureAfterIdentity(originalResult, 3, decision)

	require.Equal(t, catalog.BackupStatusError, result.Status)
	require.Equal(t, catalog.BackupErrorFailoverDuringBackup, *result.ErrorReason)
}

func Test_RecheckFullStreamFailure_WhenCanceled_PreservesResultWithoutProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reason := catalog.BackupErrorCanceledByUser
	originalResult := Result{
		Status:      catalog.BackupStatusCanceled,
		ErrorReason: &reason,
	}

	probeCalls := 0
	common := CommonBackupSpec{
		ExecutionClock: nil,
		timelineProbe: func(context.Context, int) (*timelines.TimelineDecision, error) {
			probeCalls++

			return nil, errors.New("unexpected identity probe")
		},
	}

	result := recheckFullStreamFailure(ctx, common, 3, originalResult)

	require.Equal(t, originalResult, result)
	require.Zero(t, probeCalls)
}

func Test_RecheckIncrementalStreamFailure_WhenCanceled_PreservesResultWithoutProbe(t *testing.T) {
	reason := catalog.BackupErrorCanceledByUser
	originalResult := Result{
		Status:      catalog.BackupStatusCanceled,
		ErrorReason: &reason,
	}
	probeCalls := 0
	common := CommonBackupSpec{
		ExecutionClock: nil,
		timelineProbe: func(context.Context, int) (*timelines.TimelineDecision, error) {
			probeCalls++

			return nil, errors.New("unexpected identity probe")
		},
	}

	result := recheckIncrementalStreamFailure(t.Context(), common, 3, originalResult)

	require.Equal(t, originalResult, result)
	require.Zero(t, probeCalls)
}

func Test_RecheckFullStreamFailure_WhenTimelineAdvanced_ClassifiesFailover(t *testing.T) {
	originalReason := catalog.BackupErrorPgBasebackupFailed
	originalResult := Result{
		Status:      catalog.BackupStatusError,
		ErrorReason: &originalReason,
	}
	common := CommonBackupSpec{
		ExecutionClock: nil,
		Logger:         slog.New(slog.DiscardHandler),
		timelineProbe: func(_ context.Context, expectedTimelineID int) (*timelines.TimelineDecision, error) {
			require.Equal(t, 3, expectedTimelineID)

			return &timelines.TimelineDecision{
				Kind:               timelines.TimelineFailoverDetected,
				ExpectedTimelineID: expectedTimelineID,
				LiveTimelineID:     4,
			}, nil
		},
	}

	result := recheckFullStreamFailure(t.Context(), common, 3, originalResult)

	require.Equal(t, catalog.BackupStatusError, result.Status)
	require.Equal(t, catalog.BackupErrorFailoverDuringBackup, *result.ErrorReason)
}

func Test_RecheckIncrementalStreamFailure_WhenTimelineAdvanced_BreaksChain(t *testing.T) {
	originalReason := catalog.BackupErrorPgBasebackupFailed
	originalResult := Result{
		Status:      catalog.BackupStatusError,
		ErrorReason: &originalReason,
	}
	common := CommonBackupSpec{
		ExecutionClock: nil,
		Logger:         slog.New(slog.DiscardHandler),
		timelineProbe: func(_ context.Context, expectedTimelineID int) (*timelines.TimelineDecision, error) {
			require.Equal(t, 3, expectedTimelineID)

			return &timelines.TimelineDecision{
				Kind:               timelines.TimelineFailoverDetected,
				ExpectedTimelineID: expectedTimelineID,
				LiveTimelineID:     4,
			}, nil
		},
	}

	result := recheckIncrementalStreamFailure(t.Context(), common, 3, originalResult)

	require.Equal(t, catalog.BackupStatusChainBroken, result.Status)
	require.Equal(t, catalog.BackupErrorTimelineSwitchDetected, *result.ErrorReason)
}

func Test_RecheckStreamFailure_WhenIdentityProbeFails_PreservesOriginalResult(t *testing.T) {
	originalReason := catalog.BackupErrorPgBasebackupFailed
	originalResult := Result{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &originalReason,
		ErrorMessage: "original stream failure",
	}
	common := CommonBackupSpec{
		ExecutionClock: nil,
		Logger:         slog.New(slog.DiscardHandler),
		timelineProbe: func(context.Context, int) (*timelines.TimelineDecision, error) {
			return nil, errors.New("identity probe failed")
		},
	}

	fullResult := recheckFullStreamFailure(t.Context(), common, 3, originalResult)
	incrementalResult := recheckIncrementalStreamFailure(t.Context(), common, 3, originalResult)

	require.Equal(t, originalResult, fullResult)
	require.Equal(t, originalResult, incrementalResult)
}
