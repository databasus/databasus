package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// Segments rotate at the source write rate; a tight loop keeps local dwell time
// low without measurable CPU.
const uploaderPollInterval = 1 * time.Second

func (s *WalStreamSupervisor) runUploaderLoop(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(uploaderPollInterval)
	defer ticker.Stop()

	dispatcher := newSegmentUploadDispatcher(s.uploader.ProcessSegment, s.uploader.RecordConcurrentUploads)
	defer dispatcher.waitForUploads()

	for {
		s.archiveTimelineHistoryFiles(ctx, logger)
		dispatcher.startUploads(ctx, logger, s.listAndRecordQueuedSegments())

		select {
		case <-ctx.Done():
			return

		case <-ticker.C:

		case finishedFilename := <-dispatcher.finishedUploads:
			dispatcher.markFinished(finishedFilename)
		}
	}
}

// Uses the uploader's takeover path so a segment whose pre-crash claim row is still file_name NULL
// gets finished rather than left for the cleaner. It is one pass over the files found at startup and
// returns once their uploads have finished, whatever the outcome: a failed segment waits for the live
// loop, so a storage outage at startup never keeps the receiver from spawning.
func (s *WalStreamSupervisor) recoverLocalSegmentsOnStartup(ctx context.Context, logger *slog.Logger) {
	s.archiveTimelineHistoryFiles(ctx, logger)

	recoverQueuedWalSegments(ctx, logger, queuedWalRecoverySpec{
		Upload:                  s.uploader.RecoverSegment,
		RecordConcurrentUploads: s.uploader.RecordConcurrentUploads,
		Segments:                s.listAndRecordQueuedSegments(),
	})
}

func (s *WalStreamSupervisor) listAndRecordQueuedSegments() []queuedWalSegment {
	queuedSegments := listQueuedWalSegments(s.watchDir, walSegmentSizeBytes(s.spec.SourceDB))
	sortQueuedWalSegments(queuedSegments)
	s.setLowestQueuedSegment(queuedSegments)

	return queuedSegments
}

// History files are rare and need a source connection, so they stay on the dispatching goroutine.
func (s *WalStreamSupervisor) archiveTimelineHistoryFiles(ctx context.Context, logger *slog.Logger) {
	entries, err := os.ReadDir(s.watchDir)
	if err != nil {
		logger.ErrorContext(ctx, "read wal watch dir", "error", err)

		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && pgphysical.IsTimelineHistoryFile(entry.Name()) {
			s.archiveTimelineHistoryFile(ctx, logger, entry.Name())
		}
	}
}

// UploadHistoryFile reads the body from the source cluster and is idempotent on
// (database_id, timeline_id), so re-processing the same file is free.
func (s *WalStreamSupervisor) archiveTimelineHistoryFile(ctx context.Context, logger *slog.Logger, name string) {
	timelineID, err := walmath.ParseHistoryFilename(name)
	if err != nil {
		logger.WarnContext(ctx, "skip unparseable history file", "name", name, "error", err)

		return
	}

	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		logger.WarnContext(ctx, "could not open connection to upload history file; will retry", "error", err)

		return
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if _, err := UploadHistoryFile(ctx, HistoryUploadSpec{
		Inspection:  conn,
		TimelineID:  timelineID,
		FileStore:   s.spec.FileStore,
		SourceDB:    s.spec.SourceDB,
		StorageID:   s.spec.StorageID,
		HistoryRepo: s.spec.HistoryRepo,
		Encryption:  s.spec.Encryption,
		MasterKey:   s.spec.MasterKey,
		Logger:      logger,
	}); err != nil {
		logger.WarnContext(ctx, "history upload failed; will retry next tick", "timeline_id", timelineID, "error", err)

		return
	}

	logger.InfoContext(ctx, "timeline switch observed via .history", "timeline_id", timelineID)

	if err := os.Remove(filepath.Join(s.watchDir, name)); err != nil && !os.IsNotExist(err) {
		logger.WarnContext(ctx, "failed to remove uploaded history file", "name", name, "error", err)
	}
}

func (s *WalStreamSupervisor) removePartials(logger *slog.Logger) {
	entries, err := os.ReadDir(s.watchDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !pgphysical.IsPartialWalFile(entry.Name()) {
			continue
		}

		if err := os.Remove(filepath.Join(s.watchDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			logger.Warn("failed to remove partial wal file", "name", entry.Name(), "error", err)
		}
	}
}
