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

func newStreamFailure() Result {
	return Result{
		Status:       catalog.BackupStatusError,
		ErrorReason:  new(catalog.BackupErrorPgBasebackupFailed),
		ErrorMessage: "original stream failure",
	}
}

func newProbeSpec(t *testing.T, liveTimelineID int, kind timelines.TimelineDecisionKind) CommonBackupSpec {
	t.Helper()

	return CommonBackupSpec{
		ExecutionClock: nil,
		Logger:         slog.New(slog.DiscardHandler),
		timelineProbe: func(_ context.Context, expectedTimelineID int) (*timelines.TimelineDecision, error) {
			require.Equal(t, 3, expectedTimelineID)

			return &timelines.TimelineDecision{
				Kind:               kind,
				ExpectedTimelineID: expectedTimelineID,
				LiveTimelineID:     liveTimelineID,
			}, nil
		},
	}
}

func Test_RecheckStreamFailure_WhenTimelineAdvancedSincePreflight_ClassifiesFailoverDuringBackup(t *testing.T) {
	common := newProbeSpec(t, 4, timelines.TimelineFailoverDetected)

	result := recheckStreamFailure(t.Context(), common, 3, newStreamFailure())

	require.Equal(t, catalog.BackupStatusError, result.Status,
		"the archived WAL covers the switch window, so the chain stays extendable and no immediate FULL follows")
	require.Equal(t, catalog.BackupErrorFailoverDuringBackup, *result.ErrorReason)
}

func Test_RecheckStreamFailure_WhenTimelineUnchangedSincePreflight_PreservesOriginalResult(t *testing.T) {
	// A chain whose newest INCREMENTAL is already on timeline 3 after crossing a promotion: the
	// preflight timeline is 3, so a plain stream failure is not mistaken for a new promotion.
	common := newProbeSpec(t, 3, timelines.TimelineContinue)
	streamFailure := newStreamFailure()

	result := recheckStreamFailure(t.Context(), common, 3, streamFailure)

	require.Equal(t, streamFailure, result)
}

func Test_RecheckStreamFailure_WhenTimelineRegressed_BreaksChain(t *testing.T) {
	common := newProbeSpec(t, 2, timelines.TimelineRegression)

	result := recheckStreamFailure(t.Context(), common, 3, newStreamFailure())

	require.Equal(t, catalog.BackupStatusChainBroken, result.Status)
	require.Equal(t, catalog.BackupErrorTimelineRegression, *result.ErrorReason)
}

func Test_RecheckStreamFailure_WhenCanceled_PreservesResultWithoutProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	canceled := Result{
		Status:      catalog.BackupStatusCanceled,
		ErrorReason: new(catalog.BackupErrorCanceledByUser),
	}
	probeCalls := 0
	common := CommonBackupSpec{
		ExecutionClock: nil,
		timelineProbe: func(context.Context, int) (*timelines.TimelineDecision, error) {
			probeCalls++

			return nil, errors.New("unexpected identity probe")
		},
	}

	result := recheckStreamFailure(ctx, common, 3, canceled)

	require.Equal(t, canceled, result)
	require.Zero(t, probeCalls)
}

func Test_RecheckStreamFailure_WhenIdentityProbeFails_PreservesOriginalResult(t *testing.T) {
	common := CommonBackupSpec{
		ExecutionClock: nil,
		Logger:         slog.New(slog.DiscardHandler),
		timelineProbe: func(context.Context, int) (*timelines.TimelineDecision, error) {
			return nil, errors.New("identity probe failed")
		},
	}
	streamFailure := newStreamFailure()

	result := recheckStreamFailure(t.Context(), common, 3, streamFailure)

	require.Equal(t, streamFailure, result)
}

func Test_RecheckStreamFailure_WhenIncrementalRefusedAcrossPromotion_KeepsTimelineSwitch(t *testing.T) {
	common := newProbeSpec(t, 3, timelines.TimelineContinue)
	refusal := Result{
		Status:      catalog.BackupStatusChainBroken,
		ErrorReason: new(catalog.BackupErrorTimelineSwitchDetected),
	}

	result := recheckStreamFailure(t.Context(), common, 3, refusal)

	require.Equal(t, refusal, result, "the scheduler replaces this INCREMENTAL with a FULL right away")
}
