package walstreaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func streamingSample(restartLSN walmath.LSN, observedAt time.Time) slotLivenessSample {
	return slotLivenessSample{
		RestartLSN:        restartLSN,
		LagBytes:          16 * 1024 * 1024,
		IsReceiverRunning: true,
		ObservedAt:        observedAt,
	}
}

func Test_RecordSampleAndDetectStall_WhenFirstSample_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute),
		"the first sample only arms the clock; it can never be a stall")
}

func Test_RecordSampleAndDetectStall_WhenRestartLsnAdvances_ReArmsAndDoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.False(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(200), base.Add(2*time.Minute)), time.Minute),
		"a changed restart_lsn means progress — the advance clock must reset")
}

func Test_RecordSampleAndDetectStall_WhenFrozenWithinTimeout_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.False(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(30*time.Second)), time.Minute),
		"a frozen restart_lsn within the stall timeout is not yet a stall")
}

func Test_RecordSampleAndDetectStall_WhenFrozenPastTimeout_DetectsStallThenReArms(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.True(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(90*time.Second)), time.Minute),
		"a frozen restart_lsn past the stall timeout must trigger a restart")

	require.False(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(2*time.Minute)), time.Minute),
		"after firing, the clock re-arms so we restart at most once per window")
	require.True(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(4*time.Minute)), time.Minute),
		"a sustained stall fires again only after another full timeout window")
}

func Test_RecordSampleAndDetectStall_WhenLsnFrozenAndNoLag_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	idleSample := func(observedAt time.Time) slotLivenessSample {
		return slotLivenessSample{
			RestartLSN:        walmath.LSN(100),
			LagBytes:          0,
			IsReceiverRunning: true,
			ObservedAt:        observedAt,
		}
	}

	require.False(t, tracker.recordSampleAndDetectStall(idleSample(base), time.Minute))
	require.False(t, tracker.recordSampleAndDetectStall(idleSample(base.Add(10*time.Minute)), time.Minute),
		"an idle database parks restart_lsn legitimately — there is nothing left to flush")
}

func Test_RecordSampleAndDetectStall_WhenLsnFrozenWithLag_DetectsStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.True(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(90*time.Second)), time.Minute),
		"WAL is waiting and restart_lsn has not moved — the receiver is wedged")
}

func Test_RecordSampleAndDetectStall_WhenReceiverNotRunning_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	stoppedSample := func(observedAt time.Time) slotLivenessSample {
		return slotLivenessSample{
			RestartLSN:        walmath.LSN(100),
			LagBytes:          16 * 1024 * 1024,
			IsReceiverRunning: false,
			ObservedAt:        observedAt,
		}
	}

	require.False(t, tracker.recordSampleAndDetectStall(stoppedSample(base), time.Minute))
	require.False(t, tracker.recordSampleAndDetectStall(stoppedSample(base.Add(10*time.Minute)), time.Minute),
		"a receiver we are deliberately holding down is not a stalled one")
}
