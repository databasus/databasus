package usecases_physical_postgresql

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

func Test_DecideTimelineCompatibility_WithMatchingAndConflictingIdentity_ReturnsExpectedDecision(t *testing.T) {
	expectedSystemIdentifier := "123"

	tests := []struct {
		name                 string
		liveTimelineID       int
		liveSystemIdentifier string
		expectedTimelineID   int
		decisionKind         TimelineDecisionKind
	}{
		{
			name:                 "matching timeline",
			liveTimelineID:       3,
			liveSystemIdentifier: "123",
			expectedTimelineID:   3,
			decisionKind:         TimelineContinue,
		},
		{
			name:                 "newer timeline",
			liveTimelineID:       4,
			liveSystemIdentifier: "123",
			expectedTimelineID:   3,
			decisionKind:         TimelineFailoverDetected,
		},
		{
			name:                 "older timeline",
			liveTimelineID:       2,
			liveSystemIdentifier: "123",
			expectedTimelineID:   3,
			decisionKind:         TimelineRegression,
		},
		{
			name:                 "different cluster",
			liveTimelineID:       3,
			liveSystemIdentifier: "456",
			expectedTimelineID:   3,
			decisionKind:         TimelineDifferentCluster,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := decideTimelineCompatibility(timelineComparison{
				ExpectedSystemIdentifier: &expectedSystemIdentifier,
				LiveTimelineID:           test.liveTimelineID,
				LiveSystemIdentifier:     test.liveSystemIdentifier,
				ExpectedTimelineID:       test.expectedTimelineID,
			})

			require.Equal(t, test.decisionKind, decision.Kind)
			require.Equal(t, test.expectedTimelineID, decision.ExpectedTimelineID)
			require.Equal(t, test.liveTimelineID, decision.LiveTimelineID)
		})
	}
}

func Test_ClassifyIncrementalStreamFailureAfterIdentity_WhenTimelineAdvanced_BreaksChain(t *testing.T) {
	originalReason := physical_enums.PhysicalBackupErrorPgBasebackupFailed
	originalResult := PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusError,
		ErrorReason: &originalReason,
	}
	decision := &TimelineDecision{
		Kind:               TimelineFailoverDetected,
		ExpectedTimelineID: 3,
		LiveTimelineID:     4,
	}

	result := classifyIncrementalStreamFailureAfterIdentity(originalResult, decision)

	require.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, result.Status)
	require.Equal(t, physical_enums.PhysicalBackupErrorTimelineSwitchDetected, *result.ErrorReason)
}

func Test_ClassifyFullStreamFailureAfterIdentity_WhenTimelineAdvanced_ClassifiesFailover(t *testing.T) {
	originalReason := physical_enums.PhysicalBackupErrorPgBasebackupFailed
	originalResult := PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusError,
		ErrorReason: &originalReason,
	}
	decision := &TimelineDecision{
		Kind:               TimelineFailoverDetected,
		ExpectedTimelineID: 3,
		LiveTimelineID:     4,
	}

	result := classifyFullStreamFailureAfterIdentity(originalResult, 3, decision)

	require.Equal(t, physical_enums.PhysicalBackupStatusError, result.Status)
	require.Equal(t, physical_enums.PhysicalBackupErrorFailoverDuringBackup, *result.ErrorReason)
}

func Test_RecheckFullStreamFailure_WhenCanceled_PreservesResultWithoutProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reason := physical_enums.PhysicalBackupErrorCanceledByUser
	originalResult := PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusCanceled,
		ErrorReason: &reason,
	}

	probeCalls := 0
	common := CommonBackupSpec{
		timelineProbe: func(context.Context, int) (*TimelineDecision, error) {
			probeCalls++

			return nil, errors.New("unexpected identity probe")
		},
	}

	result := recheckFullStreamFailure(ctx, common, 3, originalResult)

	require.Equal(t, originalResult, result)
	require.Zero(t, probeCalls)
}

func Test_RecheckIncrementalStreamFailure_WhenCanceled_PreservesResultWithoutProbe(t *testing.T) {
	reason := physical_enums.PhysicalBackupErrorCanceledByUser
	originalResult := PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusCanceled,
		ErrorReason: &reason,
	}
	probeCalls := 0
	common := CommonBackupSpec{
		timelineProbe: func(context.Context, int) (*TimelineDecision, error) {
			probeCalls++

			return nil, errors.New("unexpected identity probe")
		},
	}

	result := recheckIncrementalStreamFailure(context.Background(), common, 3, originalResult)

	require.Equal(t, originalResult, result)
	require.Zero(t, probeCalls)
}

func Test_RecheckFullStreamFailure_WhenTimelineAdvanced_ClassifiesFailover(t *testing.T) {
	originalReason := physical_enums.PhysicalBackupErrorPgBasebackupFailed
	originalResult := PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusError,
		ErrorReason: &originalReason,
	}
	common := CommonBackupSpec{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		timelineProbe: func(_ context.Context, expectedTimelineID int) (*TimelineDecision, error) {
			require.Equal(t, 3, expectedTimelineID)

			return &TimelineDecision{
				Kind:               TimelineFailoverDetected,
				ExpectedTimelineID: expectedTimelineID,
				LiveTimelineID:     4,
			}, nil
		},
	}

	result := recheckFullStreamFailure(context.Background(), common, 3, originalResult)

	require.Equal(t, physical_enums.PhysicalBackupStatusError, result.Status)
	require.Equal(t, physical_enums.PhysicalBackupErrorFailoverDuringBackup, *result.ErrorReason)
}

func Test_RecheckIncrementalStreamFailure_WhenTimelineAdvanced_BreaksChain(t *testing.T) {
	originalReason := physical_enums.PhysicalBackupErrorPgBasebackupFailed
	originalResult := PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusError,
		ErrorReason: &originalReason,
	}
	common := CommonBackupSpec{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		timelineProbe: func(_ context.Context, expectedTimelineID int) (*TimelineDecision, error) {
			require.Equal(t, 3, expectedTimelineID)

			return &TimelineDecision{
				Kind:               TimelineFailoverDetected,
				ExpectedTimelineID: expectedTimelineID,
				LiveTimelineID:     4,
			}, nil
		},
	}

	result := recheckIncrementalStreamFailure(context.Background(), common, 3, originalResult)

	require.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, result.Status)
	require.Equal(t, physical_enums.PhysicalBackupErrorTimelineSwitchDetected, *result.ErrorReason)
}

func Test_RecheckStreamFailure_WhenIdentityProbeFails_PreservesOriginalResult(t *testing.T) {
	originalReason := physical_enums.PhysicalBackupErrorPgBasebackupFailed
	originalResult := PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &originalReason,
		ErrorMessage: "original stream failure",
	}
	common := CommonBackupSpec{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		timelineProbe: func(context.Context, int) (*TimelineDecision, error) {
			return nil, errors.New("identity probe failed")
		},
	}

	fullResult := recheckFullStreamFailure(context.Background(), common, 3, originalResult)
	incrementalResult := recheckIncrementalStreamFailure(context.Background(), common, 3, originalResult)

	require.Equal(t, originalResult, fullResult)
	require.Equal(t, originalResult, incrementalResult)
}
