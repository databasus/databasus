package walstreaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
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
