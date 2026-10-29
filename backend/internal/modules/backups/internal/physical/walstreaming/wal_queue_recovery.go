package walstreaming

import (
	"context"
	"log/slog"
)

type queuedWalRecoverySpec struct {
	Upload                  segmentUpload
	RecordConcurrentUploads func(int)
	Segments                []queuedWalSegment
}

func recoverQueuedWalSegments(ctx context.Context, logger *slog.Logger, spec queuedWalRecoverySpec) {
	dispatcher := newSegmentUploadDispatcher(spec.Upload, spec.RecordConcurrentUploads)
	defer dispatcher.waitForUploads(ctx)

	notStartedSegments := spec.Segments

	for ctx.Err() == nil {
		notStartedSegments = dispatcher.startUploads(ctx, logger, notStartedSegments)

		if dispatcher.getRunningBackupCount() == 0 {
			return
		}

		dispatcher.markFinished(<-dispatcher.finishedUploads)
	}
}
