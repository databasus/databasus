package walstreaming

import (
	"context"
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

type walPosition struct {
	TimelineID int
	LSN        walmath.LSN
}

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

func (s *Streamer) getLastCommittedWal(ctx context.Context,
	logger *slog.Logger,
) (newestCommittedEnd *walPosition, committedAt *time.Time) {
	segment, _, err := s.spec.WalSegmentRepo.FindLatestCommittedSegment(ctx, s.spec.DatabaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read the last committed wal segment", slog.Any("error", err))

		return nil, nil
	}

	if segment == nil {
		return nil, nil
	}

	receivedAt := segment.ReceivedAt

	return &walPosition{TimelineID: segment.TimelineID, LSN: segment.EndLSN}, &receivedAt
}

func (s *Streamer) setLowestQueuedSegment(sortedSegments []queuedWalSegment) {
	s.lowestQueuedSegmentMutex.Lock()
	defer s.lowestQueuedSegmentMutex.Unlock()

	if len(sortedSegments) == 0 {
		s.lowestQueuedSegment = nil
		return
	}

	s.lowestQueuedSegment = &walPosition{TimelineID: sortedSegments[0].timelineID, LSN: sortedSegments[0].startLSN}
}

func (s *Streamer) getLowestQueuedSegment() *walPosition {
	s.lowestQueuedSegmentMutex.Lock()
	defer s.lowestQueuedSegmentMutex.Unlock()

	return s.lowestQueuedSegment
}
