package chains

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type timelineLineWalkSpec struct {
	Line      TimelineLine
	StorageID uuid.UUID
	// Segments are the line's segments, range by range and by position within a range, with no time
	// evidence on a former timeline's copy of a fork segment (withoutForkSegmentEvidence).
	Segments []*catalog.WalSegment

	LastStopLSN          walmath.LSN
	LastBackupTime       time.Time
	LastBackupTimelineID int
}

type timelineLineWalk struct {
	Run           []*catalog.WalSegment
	ReachableLSN  walmath.LSN
	ReachableTime time.Time
	// ReachedRangeIndex is the line range of the last segment in Run, or the range of the newest
	// included backup when Run is empty.
	ReachedRangeIndex int
}

// walkTimelineLine collects the committed WAL replay needs after the newest included backup, along
// the line and up to the first gap. It starts on the timeline of that backup, so a backup already
// on a promoted timeline ships no WAL of the former one. A fork point is crossed only when the
// former timeline's WAL reaches the segment holding the fork point's LSN and the next timeline has
// that segment in the FULL's storage. Replay reads the promoted timeline's copy of that segment,
// which carries the former timeline's WAL up to the fork point, so the former copy is optional.
func walkTimelineLine(spec timelineLineWalkSpec) timelineLineWalk {
	startRangeIndex := spec.Line.getRangeIndexHoldingStop(spec.LastBackupTimelineID, spec.LastStopLSN)
	walk := timelineLineWalk{
		Run:               make([]*catalog.WalSegment, 0),
		ReachableLSN:      spec.LastStopLSN,
		ReachableTime:     spec.LastBackupTime,
		ReachedRangeIndex: max(0, startRangeIndex),
	}

	// A backup that stopped past the fork point at which the line leaves its timeline holds writes
	// the line does not have, so no WAL of the line can follow it.
	if startRangeIndex < 0 {
		return walk
	}

	segmentsByTimeline := groupCommittedSegmentsByTimeline(spec.Segments)

	for rangeIndex := startRangeIndex; rangeIndex < len(spec.Line.Ranges); rangeIndex++ {
		lineRange := spec.Line.Ranges[rangeIndex]
		segments := segmentsByTimeline[lineRange.TimelineID]

		if rangeIndex > startRangeIndex {
			segments = keepSegmentsInStorage(segments, spec.StorageID)
			if !canCrossFork(segments, lineRange.StartLSN, walk.ReachableLSN) {
				return walk
			}
		}

		for segmentIndex, segment := range segments {
			isForkSegment := rangeIndex > startRangeIndex && segmentIndex == 0
			if rangeIndex == startRangeIndex && segment.EndLSN <= spec.LastStopLSN {
				continue
			}

			if !isForkSegment && segment.StartLSN > walk.ReachableLSN {
				return walk
			}

			walk.Run = append(walk.Run, segment)
			walk.ReachableLSN = max(walk.ReachableLSN, segment.EndLSN)
			walk.ReachedRangeIndex = rangeIndex

			if segment.LastTransactionAt != nil {
				// Inclusive time recovery stops before a later transaction, not at the last timestamp.
				transactionBound := segment.LastTransactionAt.Add(-time.Microsecond)
				if transactionBound.After(walk.ReachableTime) {
					walk.ReachableTime = transactionBound
				}
			}
		}
	}

	return walk
}

// canCrossFork reports whether replay can continue onto a timeline entered at forkLSN: its first
// segment holds the fork point's LSN and starts where the former timeline's WAL reached.
func canCrossFork(segments []*catalog.WalSegment, forkLSN, reachableLSN walmath.LSN) bool {
	if len(segments) == 0 {
		return false
	}

	forkSegment := segments[0]

	return forkSegment.StartLSN <= forkLSN && forkLSN < forkSegment.EndLSN && forkSegment.StartLSN <= reachableLSN
}

// withoutForkSegmentEvidence drops the time evidence of a former timeline's copy of a fork segment.
// Its newest commit can be one the former primary made after the fork point, which never became
// part of the line. The promoted timeline's copy carries the commits up to the fork point.
func withoutForkSegmentEvidence(line TimelineLine, segments []*catalog.WalSegment) []*catalog.WalSegment {
	evidenceSegments := make([]*catalog.WalSegment, 0, len(segments))

	for _, segment := range segments {
		if isFormerForkSegment(line, segment) && segment.LastTransactionAt != nil {
			withoutEvidence := *segment
			withoutEvidence.LastTransactionAt = nil
			segment = &withoutEvidence
		}

		evidenceSegments = append(evidenceSegments, segment)
	}

	return evidenceSegments
}

func isFormerForkSegment(line TimelineLine, segment *catalog.WalSegment) bool {
	for rangeIndex := 0; rangeIndex+1 < len(line.Ranges); rangeIndex++ {
		forkLSN := line.Ranges[rangeIndex+1].StartLSN
		if line.Ranges[rangeIndex].TimelineID == segment.TimelineID &&
			segment.StartLSN <= forkLSN && forkLSN < segment.EndLSN {
			return true
		}
	}

	return false
}

func groupCommittedSegmentsByTimeline(segments []*catalog.WalSegment) map[int][]*catalog.WalSegment {
	segmentsByTimeline := make(map[int][]*catalog.WalSegment)

	for _, segment := range segments {
		if segment.FileName != nil {
			segmentsByTimeline[segment.TimelineID] = append(segmentsByTimeline[segment.TimelineID], segment)
		}
	}

	return segmentsByTimeline
}

func keepSegmentsInStorage(segments []*catalog.WalSegment, storageID uuid.UUID) []*catalog.WalSegment {
	kept := make([]*catalog.WalSegment, 0, len(segments))

	for _, segment := range segments {
		if segment.StorageID == storageID {
			kept = append(kept, segment)
		}
	}

	return kept
}
