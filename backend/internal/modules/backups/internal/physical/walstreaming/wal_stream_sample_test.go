package walstreaming

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

const testSampleSegmentSize = int64(16 * 1024 * 1024)

func getTestSampleStartedAt() time.Time {
	return time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)
}

func reachableProbe(currentLSN walmath.LSN, walStatus string) slotProbe {
	return slotProbe{
		Outcome: slotProbeOutcomeReachable,
		State: &pgphysical.Slot{
			WalStatus:  walStatus,
			RestartLSN: currentLSN - 4096,
			LagBytes:   new(int64(4096)),
			CurrentLSN: new(currentLSN),
		},
	}
}

func newTestSampleInput(probe slotProbe, archiveProgressLSN walmath.LSN) walStreamSampleInput {
	return walStreamSampleInput{
		DatabaseID:           uuid.New(),
		SampledAt:            getTestSampleStartedAt(),
		SessionStartedAt:     getTestSampleStartedAt().Add(-time.Hour),
		Probe:                probe,
		ArchiveProgressLSN:   archiveProgressLSN,
		WalSegmentSizeBytes:  testSampleSegmentSize,
		LocalQueueBytes:      3 * testSampleSegmentSize,
		LocalQueuePauseBytes: 32 * testSampleSegmentSize,
		IsReceiverRunning:    true,
		Snapshot: ArchivingSnapshot{
			ArchivedSegments:         2,
			ArchivedBytes:            5 * 1024 * 1024,
			SegmentWrite:             uploadStepTiming{Total: 400 * time.Millisecond, Longest: 300 * time.Millisecond},
			HighestConcurrentUploads: 3,
		},
	}
}

// Replays ticks the way the lag monitor does, so the source write volume of each
// sample depends on the ticks before it.
func buildSamplesForProbes(probes []slotProbe) []*catalog.WalStreamSample {
	monitorState := &lagMonitorState{}
	samples := make([]*catalog.WalStreamSample, 0, len(probes))

	for index, probe := range probes {
		sampledAt := getTestSampleStartedAt().Add(time.Duration(index) * 30 * time.Second)
		monitorState.recordProbeOutcome(probe, sampledAt)

		input := newTestSampleInput(probe, walmath.LSN(0x1000000))
		input.SampledAt = sampledAt
		input.PreviousSourceLSN = monitorState.previousSourceLSN
		input.SourceUnreachableSince = monitorState.sourceUnreachableSince

		samples = append(samples, buildWalStreamSample(input))
		monitorState.rememberSourcePosition(probe)
	}

	return samples
}

func Test_BuildWalStreamSample_WhenSlotLost_KeepsRealCurrentPositionWithoutNegativeUnarchivedWal(t *testing.T) {
	const currentLSN = walmath.LSN(0x50000000)

	lostProbe := slotProbe{
		Outcome: slotProbeOutcomeReachable,
		State:   &pgphysical.Slot{LagBytes: new(int64(0)), WalStatus: "lost", CurrentLSN: new(currentLSN)},
	}

	sampleBehindProgress := buildWalStreamSample(newTestSampleInput(lostProbe, currentLSN+0x1000))
	require.NotNil(t, sampleBehindProgress.SourceCurrentLSN)
	assert.Equal(t, currentLSN, *sampleBehindProgress.SourceCurrentLSN)
	assert.Nil(t, sampleBehindProgress.UnarchivedWalBytes)

	sampleAheadOfProgress := buildWalStreamSample(newTestSampleInput(lostProbe, currentLSN-0x1000))
	require.NotNil(t, sampleAheadOfProgress.UnarchivedWalBytes)
	assert.Equal(t, int64(0x1000), *sampleAheadOfProgress.UnarchivedWalBytes)
}

func Test_BuildWalStreamSample_WhenSourcePositionMovesBackwards_RecordsNoWriteVolume(t *testing.T) {
	samples := buildSamplesForProbes([]slotProbe{
		reachableProbe(0x60000000, "reserved"),
		reachableProbe(0x50000000, "reserved"),
		reachableProbe(0x50001000, "reserved"),
	})

	assert.Nil(t, samples[1].SourceWrittenBytes, "a timeline switch must not be recorded as a negative write")
	require.NotNil(t, samples[2].SourceWrittenBytes)
	assert.Equal(t, int64(0x1000), *samples[2].SourceWrittenBytes)
}

func Test_BuildWalStreamSample_AfterStreamerStarts_RecordsWriteVolumeFromSecondSample(t *testing.T) {
	samples := buildSamplesForProbes([]slotProbe{
		reachableProbe(0x50000000, "reserved"),
		reachableProbe(0x50002000, "reserved"),
	})

	assert.Nil(t, samples[0].SourceWrittenBytes)
	require.NotNil(t, samples[1].SourceWrittenBytes)
	assert.Equal(t, int64(0x2000), *samples[1].SourceWrittenBytes)
}

func Test_BuildWalStreamSample_WhenSourceReachableAgain_RecordsWriteVolumeFromSecondReachableSample(t *testing.T) {
	samples := buildSamplesForProbes([]slotProbe{
		reachableProbe(0x50000000, "reserved"),
		{Outcome: slotProbeOutcomeUnreachable},
		reachableProbe(0x50004000, "reserved"),
		reachableProbe(0x50005000, "reserved"),
	})

	assert.Nil(t, samples[1].SourceWrittenBytes)
	assert.Nil(t, samples[2].SourceWrittenBytes, "the write volume across an unreachable tick is unknown")
	require.NotNil(t, samples[3].SourceWrittenBytes)
	assert.Equal(t, int64(0x1000), *samples[3].SourceWrittenBytes)
}

func Test_BuildWalStreamSample_WhileNothingArchived_RecordsNoUnarchivedWal(t *testing.T) {
	sample := buildWalStreamSample(newTestSampleInput(reachableProbe(0x50000000, "reserved"), 0))

	assert.Nil(t, sample.ArchiveProgressLSN)
	assert.Nil(t, sample.UnarchivedWalBytes)
	require.NotNil(t, sample.SlotLagBytes)
}

func Test_BuildWalStreamSample_WithoutSourceSideValues_LeavesThemEmptyAndFillsTheRest(t *testing.T) {
	unreachableSince := getTestSampleStartedAt().Add(-time.Minute)

	testCases := []struct {
		name                           string
		probe                          slotProbe
		expectedState                  catalog.WalStreamSourceState
		expectedSourceUnreachableSince *time.Time
	}{
		{
			name:                           "unreachable",
			probe:                          slotProbe{Outcome: slotProbeOutcomeUnreachable},
			expectedState:                  catalog.WalStreamSourceStateUnreachable,
			expectedSourceUnreachableSince: &unreachableSince,
		},
		{
			name:          "without the slot",
			probe:         slotProbe{Outcome: slotProbeOutcomeSlotMissing},
			expectedState: catalog.WalStreamSourceStateSlotMissing,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := newTestSampleInput(testCase.probe, walmath.LSN(0x1000000))
			input.PreviousSourceLSN = new(walmath.LSN(0x1000))
			input.SourceUnreachableSince = &unreachableSince

			sample := buildWalStreamSample(input)

			assert.Equal(t, testCase.expectedState, sample.SourceState)
			assert.Equal(t, testCase.expectedSourceUnreachableSince, sample.SourceUnreachableSince)

			assert.Nil(t, sample.SourceCurrentLSN)
			assert.Nil(t, sample.SourceWrittenBytes)
			assert.Nil(t, sample.SlotWalStatus)
			assert.Nil(t, sample.SlotLagBytes)
			assert.Nil(t, sample.SlotSafeWalSizeBytes)
			assert.Nil(t, sample.UnarchivedWalBytes)

			require.NotNil(t, sample.ArchiveProgressLSN)
			assert.Equal(t, 2, sample.ArchivedSegments)
			assert.Equal(t, 2*testSampleSegmentSize, sample.ArchivedWalBytes)
			assert.Equal(t, int64(5*1024*1024), sample.ArchivedStoredBytes)
			assert.Equal(t, 3, sample.HighestConcurrentUploads)
			require.NotNil(t, sample.SegmentWriteAvgMillis)
			assert.Equal(t, int64(200), *sample.SegmentWriteAvgMillis)
			assert.Equal(t, int64(300), *sample.SegmentWriteMaxMillis)
			assert.Equal(t, 3*testSampleSegmentSize, sample.LocalQueueBytes)
			assert.Equal(t, 32*testSampleSegmentSize, sample.LocalQueuePauseBytes)
			assert.True(t, sample.IsReceiverRunning)
		})
	}
}
