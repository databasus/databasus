package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/databases"
)

const (
	// pg_receivewal has no kernel-pipe back pressure (it writes files), so the
	// watch dir IS the buffer. Above HIGH we stop the receiver; we resume only once
	// uploads drain below LOW. The 5x hysteresis prevents flapping on the
	// boundary. HIGH scales up for clusters with non-default wal_segment_size so
	// one segment does not stop the receiver.
	walLocalMinHighWatermarkBytes int64 = 512 * 1024 * 1024

	backpressurePollInterval = 1 * time.Second
)

func (s *WalStreamSupervisor) runBackpressureMonitor(ctx context.Context, _ *slog.Logger) {
	ticker := time.NewTicker(backpressurePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if s.backlogBytes() >= s.highWatermarkBytes.Load() {
				s.isBackpressureStopPending.Store(true)
				s.signalRestart()
			}
		}
	}
}

func getDefaultLocalQueuePauseBytes(sourceDB databases.PostgresqlPhysicalConnectionSettings) int64 {
	return max(walLocalMinHighWatermarkBytes, 8*walSegmentSizeBytes(sourceDB))
}

func (s *WalStreamSupervisor) setLocalQueuePauseBytes(pauseBytes int64) {
	s.highWatermarkBytes.Store(pauseBytes)
	s.lowWatermarkBytes.Store(pauseBytes / 5)
}

func (s *WalStreamSupervisor) waitForBacklogBelowLow(
	ctx context.Context,
	logger *slog.Logger,
) (hasWaited, isRunning bool) {
	if s.backlogBytes() < s.highWatermarkBytes.Load() {
		return false, true
	}

	logger.WarnContext(ctx, "wal backlog over high watermark; pausing pg_receivewal until it drains")

	ticker := time.NewTicker(backpressurePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return true, false

		case <-ticker.C:
			if s.backlogBytes() < s.lowWatermarkBytes.Load() {
				logger.InfoContext(ctx, "wal backlog drained below low watermark; resuming pg_receivewal")

				return true, true
			}
		}
	}
}

// Uploaded segments are removed by the uploader, so summing what is left is the
// local queue depth; .partial and .history files are excluded. Segments staged out
// of pg_receivewal's resume path (wal_resume.go) are still waiting for the same
// uploader on the same disk, so they count too — otherwise a storage outage plus a
// realign grows the queue past the watermark with nothing throttling the receiver.
func (s *WalStreamSupervisor) backlogBytes() int64 {
	return getWalBacklogBytes(s.watchDir)
}

func sumWalSegmentBytes(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	var total int64

	for _, entry := range entries {
		if entry.IsDir() || !walmath.IsWalFilename(entry.Name(), walmath.DefaultSegmentSizeBytes) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		total += info.Size()
	}

	return total
}

func getWalBacklogBytes(watchDir string) int64 {
	return sumWalSegmentBytes(watchDir) + sumWalSegmentBytes(filepath.Join(watchDir, pendingUploadDirName))
}
