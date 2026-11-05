package walstreaming

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

const uploaderPollInterval = 1 * time.Second

func (s *Streamer) runUploaderLoop(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(uploaderPollInterval)
	defer ticker.Stop()

	dispatcher := newSegmentUploadDispatcher(s.uploader.ProcessSegment, s.uploader.RecordConcurrentUploads)
	defer dispatcher.waitForUploads(ctx)

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

func (s *Streamer) recoverLocalSegmentsOnStartup(ctx context.Context, logger *slog.Logger) {
	s.archiveTimelineHistoryFiles(ctx, logger)

	recoverQueuedWalSegments(ctx, logger, queuedWalRecoverySpec{
		Upload:                  s.uploader.RecoverSegment,
		RecordConcurrentUploads: s.uploader.RecordConcurrentUploads,
		Segments:                s.listAndRecordQueuedSegments(),
	})
}

func (s *Streamer) listAndRecordQueuedSegments() []queuedWalSegment {
	queuedSegments := listQueuedWalSegments(s.watchDir, walSegmentSizeBytes(s.spec.SourceDB))
	sortQueuedWalSegments(queuedSegments)
	s.setLowestQueuedSegment(queuedSegments)

	return queuedSegments
}

func (s *Streamer) archiveTimelineHistoryFiles(ctx context.Context, logger *slog.Logger) {
	entries, err := os.ReadDir(s.watchDir)
	if err != nil {
		logger.ErrorContext(ctx, "read wal watch dir", slog.Any("error", err))

		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && pgphysical.IsTimelineHistoryFile(entry.Name()) {
			s.archiveTimelineHistoryFile(ctx, logger, entry.Name())
		}
	}
}

func (s *Streamer) archiveTimelineHistoryFile(ctx context.Context, logger *slog.Logger, name string) {
	timelineID, err := walmath.ParseHistoryFilename(name)
	if err != nil {
		logger.WarnContext(ctx, "skip unparseable history file", slog.Any("name", name), slog.Any("error", err))

		return
	}

	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		logger.WarnContext(ctx, "could not open connection to upload history file; will retry", slog.Any("error", err))

		return
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	if _, err := timelines.UploadHistoryFile(ctx, timelines.HistoryUploadSpec{
		Publication: s.spec.Publication, Clock: s.spec.Clock, Transactor: s.spec.Transactor,
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
		logger.WarnContext(
			ctx,
			"history upload failed; will retry next tick",
			slog.Any("timeline_id", timelineID),
			slog.Any("error", err),
		)

		return
	}

	logger.InfoContext(ctx, "timeline switch observed via .history", slog.Any("timeline_id", timelineID))

	if err := os.Remove(filepath.Join(s.watchDir, name)); err != nil && !os.IsNotExist(err) {
		logger.WarnContext(
			ctx,
			"failed to remove uploaded history file",
			slog.Any("name", name),
			slog.Any("error", err),
		)
	}
}

func (s *Streamer) removePartials(ctx context.Context, logger *slog.Logger) {
	entries, err := os.ReadDir(s.watchDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !pgphysical.IsPartialWalFile(entry.Name()) {
			continue
		}

		if err := os.Remove(filepath.Join(s.watchDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			logger.WarnContext(
				ctx,
				"failed to remove partial wal file",
				slog.Any("name", entry.Name()),
				slog.Any("error", err),
			)
		}
	}
}
