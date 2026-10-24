package usecases_physical_postgresql

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/testing/containers"
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

func Test_ResolveLiveTimelineID_WhenSourcesDisagree_PrefersTheFreshestReading(t *testing.T) {
	promotedPrimaryWalFilename := "000000020000000000000005"

	tests := []struct {
		name               string
		sources            clusterIdentitySources
		resolvedTimelineID int
	}{
		{
			name: "promoted primary whose control file has not caught up",
			sources: clusterIdentitySources{
				CurrentWalFilename:    &promotedPrimaryWalFilename,
				ControlFileTimelineID: 1,
			},
			resolvedTimelineID: 2,
		},
		{
			name:               "cluster in recovery, which can only answer the control file",
			sources:            clusterIdentitySources{ControlFileTimelineID: 3},
			resolvedTimelineID: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timelineID, err := resolveLiveTimelineID(test.sources)

			require.NoError(t, err)
			require.Equal(t, test.resolvedTimelineID, timelineID)
		})
	}
}

func Test_ResolveLiveTimelineID_WhenWalFilenameIsUnreadable_ReturnsError(t *testing.T) {
	unreadableWalFilename := "not-a-wal-file"

	_, err := resolveLiveTimelineID(clusterIdentitySources{
		CurrentWalFilename:    &unreadableWalFilename,
		ControlFileTimelineID: 1,
	})

	require.Error(
		t,
		err,
		"a source naming its WAL file in a shape we cannot read must not fall back to a stale timeline",
	)
}

const timelineTestPostgresVersion = "17"

func Test_ReadClusterIdentity_WhenTheControlFileLagsAPromotion_ReportsTheLiveTimeline(t *testing.T) {
	primary, standby := containers.StartPhysicalPrimaryWithStandby(
		t, "postgres:"+timelineTestPostgresVersion, containers.WithCheckpointsSpreadOverAnHour())

	ctx := t.Context()
	primaryConn := openPhysicalTestConn(t, ctx, primary)
	standbyConn := openPhysicalTestConn(t, ctx, standby)

	// Replaying these rows dirties buffers on the standby, which is what leaves the
	// post-promotion checkpoint work to spread instead of finishing at once.
	_, err := primaryConn.Exec(ctx, "CREATE TABLE promotion_ballast AS SELECT generate_series(1, 300000) AS n")
	require.NoError(t, err)
	waitForReplayedRelation(t, ctx, standbyConn, "promotion_ballast")

	var isPromoted bool
	require.NoError(t, standbyConn.QueryRow(ctx,
		"SELECT pg_promote(wait => true, wait_seconds => 60)").Scan(&isPromoted))
	require.True(t, isPromoted, "pg_promote must finish promotion within its wait window")

	var controlFileTimelineID int
	require.NoError(t, standbyConn.QueryRow(ctx,
		"SELECT timeline_id FROM pg_control_checkpoint()").Scan(&controlFileTimelineID))
	require.Equal(t, 1, controlFileTimelineID,
		"the spread-checkpoint source must keep the control file behind, or this test proves nothing")

	liveTimelineID, _, err := readClusterIdentity(ctx, standbyConn)
	require.NoError(t, err)
	require.Equal(t, 2, liveTimelineID,
		"a promoted source must report the timeline it writes WAL on, not the one its control file was stamped with")
}

func openPhysicalTestConn(t *testing.T, ctx context.Context, endpoint containers.Endpoint) *pgx.Conn {
	t.Helper()

	sourceDB := databases.GetTestPhysicalPostgresConfigWithType(
		endpoint.Host, endpoint.Port, timelineTestPostgresVersion, postgresql_physical.BackupTypeFullOnly)

	conn, err := sourceDB.OpenInspectionConn(ctx, encryption.GetFieldEncryptor())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	return conn
}

func waitForReplayedRelation(t *testing.T, ctx context.Context, standbyConn *pgx.Conn, relation string) {
	t.Helper()

	deadline := time.Now().UTC().Add(60 * time.Second)

	for time.Now().UTC().Before(deadline) {
		var replayedRelation *string

		require.NoError(t, standbyConn.QueryRow(ctx, "SELECT to_regclass($1)::text", relation).Scan(&replayedRelation))

		if replayedRelation != nil {
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("relation %q never replayed onto the standby", relation)
}
