package walstreaming

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
	walLocalMinHighWatermarkBytes int64 = 512 * 1024 * 1024

	backpressurePollInterval = 1 * time.Second
)

func (s *Streamer) runBackpressureMonitor(ctx context.Context, _ *slog.Logger) {
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

func (s *Streamer) setLocalQueuePauseBytes(pauseBytes int64) {
	s.highWatermarkBytes.Store(pauseBytes)
	s.lowWatermarkBytes.Store(pauseBytes / 5)
}

func (s *Streamer) waitForBacklogBelowLow(
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

func (s *Streamer) backlogBytes() int64 {
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
