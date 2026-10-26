package usecases_physical_postgresql

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

const testRotationInterval = 5 * time.Minute

func Test_RecordSampleAndDecideRotation_WhenWalWrittenSinceLastRotation_RotatesSegment(t *testing.T) {
	var rotationTracker walRotationTracker

	base := time.Now().UTC()

	require.False(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: walmath.LSN(1_000),
		ObservedAt: base,
	}, testRotationInterval), "the first sample only anchors the interval")

	require.True(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: walmath.LSN(2_000),
		ObservedAt: base.Add(testRotationInterval + time.Second),
	}, testRotationInterval), "WAL was written and the interval elapsed, so it must not wait for the segment to fill")
}

func Test_RecordSampleAndDecideRotation_WhenNothingWrittenSinceLastRotation_SkipsRotation(t *testing.T) {
	var rotationTracker walRotationTracker

	base := time.Now().UTC()
	idleLSN := walmath.LSN(1_000)

	require.False(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: idleLSN,
		ObservedAt: base,
	}, testRotationInterval))

	rotationTracker.recordRotation(idleLSN, base.Add(time.Second))

	require.False(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: idleLSN,
		ObservedAt: base.Add(3 * testRotationInterval),
	}, testRotationInterval), "a dead-quiet source must not be charged a padded segment per interval")
}

func Test_RecordSampleAndDecideRotation_WhenIntervalNotElapsed_SkipsRotation(t *testing.T) {
	var rotationTracker walRotationTracker

	base := time.Now().UTC()

	require.False(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: walmath.LSN(1_000),
		ObservedAt: base,
	}, testRotationInterval))

	require.False(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: walmath.LSN(2_000),
		ObservedAt: base.Add(testRotationInterval / 2),
	}, testRotationInterval), "a busy source rotates on its own; forcing it more often than the interval is waste")
}

func Test_IsInsufficientPrivilegeError_WhenSwitchWalRefused_DetectsMissingPrivilege(t *testing.T) {
	refusal := fmt.Errorf("pg_switch_wal: %w", &pgconn.PgError{
		Code:    pgErrorCodeInsufficientPrivilege,
		Message: "permission denied for function pg_switch_wal",
	})

	require.True(t, isInsufficientPrivilegeError(refusal),
		"physical backups only require REPLICATION, so a refusal is the expected case, not a retryable blip")
}

func Test_IsInsufficientPrivilegeError_WhenSourceUnreachable_ReportsRetryable(t *testing.T) {
	require.False(t, isInsufficientPrivilegeError(errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")))
}

func Test_RecordRotation_WhenSwitchMovedTheInsertPoint_DoesNotCountItAsFreshWal(t *testing.T) {
	var rotationTracker walRotationTracker

	base := time.Now().UTC()

	require.False(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: walmath.LSN(1_000),
		ObservedAt: base,
	}, testRotationInterval))

	require.True(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: walmath.LSN(1_000),
		ObservedAt: base.Add(testRotationInterval),
	}, testRotationInterval), "WAL written before the streamer started still has to reach storage")

	currentLSNAfterRotation := walmath.LSN(16 * 1024 * 1024)
	rotationTracker.recordRotation(currentLSNAfterRotation, base.Add(testRotationInterval))

	require.False(t, rotationTracker.recordSampleAndDecideRotation(walRotationSample{
		CurrentLSN: currentLSNAfterRotation,
		ObservedAt: base.Add(10 * testRotationInterval),
	}, testRotationInterval), "the switch itself moved the insert point; that is not the source writing")
}

func Test_RunForcedWalRotation_WhenSourceRefusesSwitchWal_SetsRefusalFlag(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:             uuid.New(),
		SourceDB:               CreateReplicationOnlyRole(t, adminConn, fixture),
		WatchDirRoot:           t.TempDir(),
		ForcedRotationInterval: time.Second,
		Logger:                 logger.GetLogger(),
	})

	rotationDone := make(chan struct{})

	go func() {
		defer close(rotationDone)

		supervisor.runForcedWalRotation(t.Context(), logger.GetLogger())
	}()

	require.False(t, supervisor.isForcedRotationDenied.Load())

	require.Eventually(t, func() bool {
		_, err := GenerateWalActivity(t.Context(), adminConn, 4096)

		return err == nil && supervisor.isForcedRotationDenied.Load()
	}, 60*time.Second, time.Second, "a refused pg_switch_wal must be visible to the samples")

	select {
	case <-rotationDone:
	case <-time.After(10 * time.Second):
		t.Fatal("the rotation loop must stop after a permanent refusal")
	}
}
