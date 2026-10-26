package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Segments rotate at the source write rate; a tight loop keeps local dwell time
// low without measurable CPU.
const uploaderPollInterval = 1 * time.Second

// A segment's LSN bounds come from its filename, so a file that is not exactly
// one wal_segment_size long would be catalogued as covering WAL it does not hold.
// pg_receivewal only renames a segment into place once it is full, so anything
// short is a torn leftover the receiver will re-stream.
func isCompleteSegmentFile(localPath string, segmentSizeBytes int64) bool {
	info, err := os.Stat(localPath)

	return err == nil && info.Size() == segmentSizeBytes
}

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

	dispatcher := newSegmentUploadDispatcher(s.uploader.RecoverSegment, s.uploader.RecordConcurrentUploads)
	defer dispatcher.waitForUploads()

	notStartedSegments := s.listAndRecordQueuedSegments()

	for ctx.Err() == nil {
		notStartedSegments = dispatcher.startUploads(ctx, logger, notStartedSegments)

		if dispatcher.getInFlightCount() == 0 {
			return
		}

		dispatcher.markFinished(<-dispatcher.finishedUploads)
	}
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
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".history") {
			s.archiveTimelineHistoryFile(ctx, logger, entry.Name())
		}
	}
}

// UploadHistoryFile reads the body from the source cluster and is idempotent on
// (database_id, timeline_id), so re-processing the same file is free.
func (s *WalStreamSupervisor) archiveTimelineHistoryFile(ctx context.Context, logger *slog.Logger, name string) {
	timelineID, err := parseHistoryTimeline(name)
	if err != nil {
		logger.WarnContext(ctx, "skip unparseable history file", "name", name, "error", err)

		return
	}

	conn, err := openInspectionConn(ctx, s.spec.SourceDB)
	if err != nil {
		logger.WarnContext(ctx, "could not open connection to upload history file; will retry", "error", err)

		return
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if _, err := UploadHistoryFile(ctx, HistoryUploadSpec{
		Conn:        conn,
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
		if !strings.HasSuffix(entry.Name(), ".partial") {
			continue
		}

		if err := os.Remove(filepath.Join(s.watchDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			logger.Warn("failed to remove partial wal file", "name", entry.Name(), "error", err)
		}
	}
}

func parseHistoryTimeline(name string) (int, error) {
	trimmed := strings.TrimSuffix(name, ".history")
	if trimmed == name {
		return 0, fmt.Errorf("not a history filename: %q", name)
	}

	timelineID, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("parse timeline from %q: %w", name, err)
	}

	return int(timelineID), nil
}
