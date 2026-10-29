package walstreaming

import (
	"context"
	"log/slog"

	"databasus-backend/internal/lib/background"
)

const maxConcurrentSegmentUploads = 4

type segmentUpload func(ctx context.Context, localPath, walFilename string) error

type segmentUploadDispatcher struct {
	upload                  segmentUpload
	recordConcurrentUploads func(count int)

	inFlightFilenames map[string]struct{}
	finishedUploads   chan string
	workers           *background.Group
}

func newSegmentUploadDispatcher(
	upload segmentUpload,
	recordConcurrentUploads func(count int),
) *segmentUploadDispatcher {
	return &segmentUploadDispatcher{
		upload:                  upload,
		recordConcurrentUploads: recordConcurrentUploads,
		inFlightFilenames:       make(map[string]struct{}, maxConcurrentSegmentUploads),

		finishedUploads: make(chan string, maxConcurrentSegmentUploads),
	}
}

func (d *segmentUploadDispatcher) startUploads(
	ctx context.Context,
	logger *slog.Logger,
	sortedSegments []queuedWalSegment,
) []queuedWalSegment {
	if d.workers == nil {
		d.workers = background.New(background.Deps{Logger: logger})
	}
	for index, segment := range sortedSegments {
		if _, isInFlight := d.inFlightFilenames[segment.walFilename]; isInFlight {
			continue
		}

		if len(d.inFlightFilenames) >= maxConcurrentSegmentUploads || ctx.Err() != nil {
			return d.dropInFlight(sortedSegments[index:])
		}

		d.inFlightFilenames[segment.walFilename] = struct{}{}
		d.recordConcurrentUploads(len(d.inFlightFilenames))

		d.workers.Go(ctx, "wal_upload", func(ctx context.Context) error {
			defer func() { d.finishedUploads <- segment.walFilename }()
			if err := d.upload(ctx, segment.localPath, segment.walFilename); err != nil {
				logger.WarnContext(
					ctx,
					"wal segment upload failed; will retry on a later pass",
					slog.Any("wal_filename", segment.walFilename),
					slog.Any("error", err),
				)
			}

			return nil
		})
	}

	return nil
}

func (d *segmentUploadDispatcher) dropInFlight(segments []queuedWalSegment) []queuedWalSegment {
	notStarted := make([]queuedWalSegment, 0, len(segments))

	for _, segment := range segments {
		if _, isInFlight := d.inFlightFilenames[segment.walFilename]; !isInFlight {
			notStarted = append(notStarted, segment)
		}
	}

	return notStarted
}

func (d *segmentUploadDispatcher) markFinished(walFilename string) {
	delete(d.inFlightFilenames, walFilename)
}

func (d *segmentUploadDispatcher) getRunningBackupCount() int {
	return len(d.inFlightFilenames)
}

func (d *segmentUploadDispatcher) waitForUploads(ctx context.Context) {
	if d.workers != nil {
		_ = d.workers.Wait(context.WithoutCancel(ctx))
	}
}
