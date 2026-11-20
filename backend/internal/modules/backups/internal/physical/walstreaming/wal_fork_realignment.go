package walstreaming

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// realignQueueToFork makes the next pg_receivewal resume at the segment holding the fork point's
// LSN on the timeline the server's history left there. Queued segments of that timeline from the
// fork segment onward hold the former primary's writes after the fork point and are dropped.
// pg_receivewal resumes at the start of the newest partial segment in its directory, so a zero-filled
// partial copy of the fork segment sets the position even when every earlier segment was already
// uploaded and removed. The server then sends the former timeline up to the fork point and continues
// on its own timeline.
func realignQueueToFork(watchDir string, fork pgphysical.WalReceiverFork, segmentSizeBytes int64) (int, error) {
	forkSegmentNo := walmath.NewWalSegmentNo(fork.ForkLSN, uint64(segmentSizeBytes))
	forkSegmentStart := forkSegmentNo.FirstLSN(uint64(segmentSizeBytes))

	droppedCount := 0
	for _, segment := range listQueuedWalSegments(watchDir, segmentSizeBytes) {
		if segment.timelineID != fork.LeftTimelineID || segment.startLSN < forkSegmentStart {
			continue
		}

		if err := os.Remove(segment.localPath); err != nil && !os.IsNotExist(err) {
			return droppedCount, fmt.Errorf("drop %s past the fork point: %w", segment.walFilename, err)
		}

		droppedCount++
	}

	placeholderName := forkSegmentNo.GetFilename(uint32(fork.LeftTimelineID), uint64(segmentSizeBytes)) + ".partial"
	placeholder, err := os.OpenFile(filepath.Join(watchDir, placeholderName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return droppedCount, fmt.Errorf("create the fork segment placeholder: %w", err)
	}

	if err := placeholder.Truncate(segmentSizeBytes); err != nil {
		_ = placeholder.Close()

		return droppedCount, fmt.Errorf("size the fork segment placeholder: %w", err)
	}

	if err := placeholder.Close(); err != nil {
		return droppedCount, fmt.Errorf("close the fork segment placeholder: %w", err)
	}

	return droppedCount, nil
}

// applyPendingForkRealignment runs after partial files from the previous receiver are removed, so
// the placeholder it writes survives until pg_receivewal starts.
func (s *Streamer) applyPendingForkRealignment(ctx context.Context, logger *slog.Logger) {
	fork := s.pendingForkRealignment
	if fork == nil {
		return
	}

	s.pendingForkRealignment = nil

	droppedCount, err := realignQueueToFork(s.watchDir, *fork, walSegmentSizeBytes(s.spec.SourceDB))
	if err != nil {
		logger.ErrorContext(ctx, "failed to realign wal reception to the fork point", slog.Any("error", err))

		return
	}

	logger.WarnContext(
		ctx,
		"realigned wal reception to the fork point of the promoted timeline",
		slog.Int("timeline_id", fork.LeftTimelineID),
		slog.String("fork_lsn", fork.ForkLSN.String()),
		slog.Int("dropped_segment_count", droppedCount),
	)
}
