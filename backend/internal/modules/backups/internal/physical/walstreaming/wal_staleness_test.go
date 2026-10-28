package walstreaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

const testStalenessThreshold = 30 * time.Minute

// The source keeps writing while the newest restorable segment stays where it is.
func fallingBehindSample(sourceCurrentLSN walmath.LSN, observedAt time.Time) walArchiveSample {
	return walArchiveSample{
		SourceCurrentLSN:     sourceCurrentLSN,
		ArchiveProgressPoint: walmath.LSN(testWalSegmentSize),
		ObservedAt:           observedAt,
		StalenessThreshold:   testStalenessThreshold,
	}
}

func Test_RecordSampleAndDetectStaleness_WhenSourceKeepsWritingAndNothingCommitsPastThreshold_DetectsStaleness(
	t *testing.T,
) {
	var stalenessTracker walArchiveStalenessTracker

	base := time.Now().UTC()
	writtenLSN := walmath.LSN(2 * testWalSegmentSize)

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(fallingBehindSample(writtenLSN, base)),
		"the first sample cannot tell whether the source is writing")
	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+4096, base.Add(time.Minute)),
	), "the source is writing now, so the clock starts")

	require.True(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+8192, base.Add(testStalenessThreshold+2*time.Minute)),
	), "WAL kept being written and none of it became restorable — the operator has to hear about it")

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+12288, base.Add(2*testStalenessThreshold)),
	), "one alert per incident; the throttle in the wiring layer must not be the only guard")
}

func Test_RecordSampleAndDetectStaleness_WhenSourceIsIdle_DoesNotDetectStaleness(t *testing.T) {
	var stalenessTracker walArchiveStalenessTracker

	base := time.Now().UTC()

	// An idle source parks its insert point inside the open segment, so it sits
	// permanently ahead of the newest archived one. That gap is what forced
	// rotation is for, not an incident.
	idleSourceLSN := walmath.LSN(testWalSegmentSize + 4096)

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(fallingBehindSample(idleSourceLSN, base)))
	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(idleSourceLSN, base.Add(24*time.Hour)),
	), "a database nobody writes to can never fall further behind")
}

func Test_RecordSampleAndDetectStaleness_WhenSegmentCommits_ReArmsClock(t *testing.T) {
	var stalenessTracker walArchiveStalenessTracker

	base := time.Now().UTC()
	writtenLSN := walmath.LSN(2 * testWalSegmentSize)

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(fallingBehindSample(writtenLSN, base)))
	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+4096, base.Add(time.Minute)),
	))

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(walArchiveSample{
		SourceCurrentLSN:     writtenLSN + 8192,
		ArchiveProgressPoint: walmath.LSN(2 * testWalSegmentSize),
		ObservedAt:           base.Add(testStalenessThreshold / 2),
		StalenessThreshold:   testStalenessThreshold,
	}), "a segment reached storage")

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(walArchiveSample{
		SourceCurrentLSN:     writtenLSN + 12288,
		ArchiveProgressPoint: walmath.LSN(2 * testWalSegmentSize),
		ObservedAt:           base.Add(testStalenessThreshold + time.Minute),
		StalenessThreshold:   testStalenessThreshold,
	}), "a slow but progressing archive must not accumulate staleness across the segments it did deliver")
}

func Test_RecordSampleAndDetectStaleness_WithinThreshold_DoesNotDetectStaleness(t *testing.T) {
	var stalenessTracker walArchiveStalenessTracker

	base := time.Now().UTC()
	writtenLSN := walmath.LSN(2 * testWalSegmentSize)

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(fallingBehindSample(writtenLSN, base)))
	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+4096, base.Add(time.Minute)),
	))
	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+8192, base.Add(testStalenessThreshold-time.Minute)),
	), "a segment being filled right now is normal, not a stalled recovery point")
}

func Test_RecordSampleAndDetectStaleness_WhenWritesAreBursty_KeepsCountingAcrossQuietTicks(t *testing.T) {
	var stalenessTracker walArchiveStalenessTracker

	base := time.Now().UTC()
	writtenLSN := walmath.LSN(2 * testWalSegmentSize)

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(fallingBehindSample(writtenLSN, base)))
	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+4096, base.Add(time.Minute)),
	))

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+4096, base.Add(2*time.Minute)),
	), "a quiet tick mid-incident is not recovery")

	require.True(t, stalenessTracker.recordSampleAndDetectStaleness(
		fallingBehindSample(writtenLSN+4096, base.Add(testStalenessThreshold+2*time.Minute)),
	), "a broken archive must not escape detection just because the writes came in bursts")
}

func Test_RecordSampleAndDetectStaleness_WhenThresholdDisabled_DoesNotDetectStaleness(t *testing.T) {
	var stalenessTracker walArchiveStalenessTracker

	base := time.Now().UTC()
	writtenLSN := walmath.LSN(2 * testWalSegmentSize)

	disabledSample := fallingBehindSample(writtenLSN, base)
	disabledSample.StalenessThreshold = 0

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(disabledSample))

	disabledSample.SourceCurrentLSN = writtenLSN + 4096
	disabledSample.ObservedAt = base.Add(24 * time.Hour)

	require.False(t, stalenessTracker.recordSampleAndDetectStaleness(disabledSample))
}

func segmentStart(timelineID int, segmentNo uint64) *walPosition {
	return &walPosition{TimelineID: timelineID, LSN: walmath.LSN(segmentNo * uint64(testWalSegmentSize))}
}

func segmentEnd(timelineID int, segmentNo uint64) *walPosition {
	return segmentStart(timelineID, segmentNo+1)
}

func Test_GetArchiveProgressPoint_WhenOneSegmentKeepsFailingWhileOthersSucceed_HoldsAndDetectsStaleness(t *testing.T) {
	var stalenessTracker walArchiveStalenessTracker

	base := time.Now().UTC()
	sourceLSN := walmath.LSN(100 * uint64(testWalSegmentSize))
	stuckSegmentStart := segmentStart(1, 50)

	for minute, newestCommittedSegmentNo := range []uint64{51, 60, 70, 80, 90} {
		progressPoint := getArchiveProgressPoint(stuckSegmentStart, segmentEnd(1, newestCommittedSegmentNo))

		require.Equal(t, stuckSegmentStart.LSN, progressPoint,
			"segments archived after a stuck one must not move the gap-free archive")

		require.False(t, stalenessTracker.recordSampleAndDetectStaleness(walArchiveSample{
			SourceCurrentLSN:     sourceLSN + walmath.LSN(minute*4096),
			ArchiveProgressPoint: progressPoint,
			ObservedAt:           base.Add(time.Duration(minute) * time.Minute),
			StalenessThreshold:   testStalenessThreshold,
		}))
	}

	require.True(t, stalenessTracker.recordSampleAndDetectStaleness(walArchiveSample{
		SourceCurrentLSN:     sourceLSN + 1024*1024,
		ArchiveProgressPoint: getArchiveProgressPoint(stuckSegmentStart, segmentEnd(1, 90)),
		ObservedAt:           base.Add(testStalenessThreshold + 5*time.Minute),
		StalenessThreshold:   testStalenessThreshold,
	}), "one stuck segment must trip the alert even while later segments keep arriving")
}

func Test_GetArchiveProgressPoint_WhenNothingIsPendingBelowNewestCommitted_ReturnsNewestCommittedEnd(t *testing.T) {
	require.Equal(t, segmentEnd(1, 12).LSN, getArchiveProgressPoint(segmentStart(1, 13), segmentEnd(1, 12)),
		"the next segment waiting right behind the archive does not hold the point back")
	require.Equal(t, segmentEnd(1, 12).LSN, getArchiveProgressPoint(segmentStart(1, 20), segmentEnd(1, 12)))
}

func Test_GetArchiveProgressPoint_WhenSourceIsIdle_ReturnsNewestCommittedEnd(t *testing.T) {
	require.Equal(t, segmentEnd(1, 12).LSN, getArchiveProgressPoint(nil, segmentEnd(1, 12)))
	require.Zero(t, getArchiveProgressPoint(nil, nil))
}

func Test_GetArchiveProgressPoint_WhenPendingSegmentIsOnOlderTimeline_HoldsPoint(t *testing.T) {
	olderTimelinePending := segmentStart(1, 40)
	newerTimelineCommitted := segmentEnd(2, 30)

	require.Equal(t, olderTimelinePending.LSN, getArchiveProgressPoint(olderTimelinePending, newerTimelineCommitted),
		"restore cannot reach the newer timeline past WAL the older one still owes")
}
