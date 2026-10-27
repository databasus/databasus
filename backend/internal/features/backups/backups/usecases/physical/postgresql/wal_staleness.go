package usecases_physical_postgresql

import (
	"log/slog"
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type walArchiveSample struct {
	SourceCurrentLSN     walmath.LSN
	ArchiveProgressPoint walmath.LSN
	ObservedAt           time.Time
	StalenessThreshold   time.Duration
}

type walArchiveStalenessTracker struct {
	previousSourceCurrentLSN walmath.LSN
	archiveProgressPoint     walmath.LSN
	hasSample                bool

	fallingBehindSince time.Time
	isStaleReported    bool
}

// Falling behind means the source keeps writing while nothing new becomes
// restorable. Both halves are needed: an idle source always leaves its open
// segment slightly ahead of the last archived one — that is the RPO forced
// rotation exists for, not an incident — and a slow archive that still delivers
// segments is not stale either. The clock survives quiet ticks, so bursty writes
// with a broken archive are still caught.
func (t *walArchiveStalenessTracker) recordSampleAndDetectStaleness(sample walArchiveSample) bool {
	if sample.StalenessThreshold <= 0 {
		*t = walArchiveStalenessTracker{}

		return false
	}

	isSourceWriting := t.hasSample && sample.SourceCurrentLSN > t.previousSourceCurrentLSN
	isArchiveProgressing := sample.ArchiveProgressPoint > t.archiveProgressPoint

	t.previousSourceCurrentLSN = sample.SourceCurrentLSN
	t.archiveProgressPoint = max(t.archiveProgressPoint, sample.ArchiveProgressPoint)
	t.hasSample = true

	if isArchiveProgressing {
		t.fallingBehindSince = time.Time{}
		t.isStaleReported = false

		return false
	}

	if t.fallingBehindSince.IsZero() {
		if !isSourceWriting {
			return false
		}

		t.fallingBehindSince = sample.ObservedAt

		return false
	}

	if t.isStaleReported || sample.ObservedAt.Sub(t.fallingBehindSince) < sample.StalenessThreshold {
		return false
	}

	t.isStaleReported = true

	return true
}

// A WAL position as restore reads it: timeline first, then the position within it.
type walPosition struct {
	TimelineID int
	LSN        walmath.LSN
}

// The archive progress point is where WAL is archived without holes. Newer segments that finish
// while an older one is still pending do not move it, so one stuck segment shows up as staleness even
// while later segments keep arriving. A pending segment on an older timeline counts as older.
func getArchiveProgressPoint(lowestQueuedStart, newestCommittedEnd *walPosition) walmath.LSN {
	if lowestQueuedStart == nil {
		if newestCommittedEnd == nil {
			return 0
		}

		return newestCommittedEnd.LSN
	}

	if newestCommittedEnd == nil ||
		compareWalOrder(
			lowestQueuedStart.TimelineID, lowestQueuedStart.LSN,
			newestCommittedEnd.TimelineID, newestCommittedEnd.LSN,
		) < 0 {
		return lowestQueuedStart.LSN
	}

	return newestCommittedEnd.LSN
}

func (s *WalStreamSupervisor) getLastCommittedWal(
	logger *slog.Logger,
) (newestCommittedEnd *walPosition, committedAt *time.Time) {
	segment, err := s.spec.WalSegmentRepo.FindLatestCommittedSegment(s.spec.DatabaseID)
	if err != nil {
		logger.Error("failed to read the last committed wal segment", "error", err)

		return nil, nil
	}

	if segment == nil {
		return nil, nil
	}

	receivedAt := segment.ReceivedAt

	return &walPosition{TimelineID: segment.TimelineID, LSN: segment.EndLSN}, &receivedAt
}

// The dispatcher stores the lowest segment of each listing, short files included: restore cannot pass
// that position until the segment is archived, so the progress point must not pass it either.
// Between listings the stored value can only be lower than the truth, which errs towards reporting
// staleness rather than hiding it.
func (s *WalStreamSupervisor) setLowestQueuedSegment(sortedSegments []queuedWalSegment) {
	s.lowestQueuedSegmentMutex.Lock()
	defer s.lowestQueuedSegmentMutex.Unlock()

	if len(sortedSegments) == 0 {
		s.lowestQueuedSegment = nil
		return
	}

	s.lowestQueuedSegment = &walPosition{TimelineID: sortedSegments[0].timelineID, LSN: sortedSegments[0].startLSN}
}

func (s *WalStreamSupervisor) getLowestQueuedSegment() *walPosition {
	s.lowestQueuedSegmentMutex.Lock()
	defer s.lowestQueuedSegmentMutex.Unlock()

	return s.lowestQueuedSegment
}
