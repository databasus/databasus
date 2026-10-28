package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func toWalStreamSeriesPoint(point catalog.WalStreamSeriesPoint) (WalStreamSeriesPoint, error) {
	body := WalStreamSeriesPoint{}
	body.ArchiveBytesPerSecond = point.ArchiveBytesPerSecond
	body.ArchivedStoredBytesPerSecond = point.ArchivedStoredBytesPerSecond

	if point.HighestConcurrentUploads != nil {
		body.HighestConcurrentUploads = new(int(*point.HighestConcurrentUploads))
	}

	body.LocalQueuePauseBytes = point.LocalQueuePauseBytes
	body.MaxLocalQueueBytes = point.MaxLocalQueueBytes
	body.MaxSlotLagBytes = point.MaxSlotLagBytes
	body.MaxUnarchivedWalBytes = point.MaxUnarchivedWalBytes

	if point.Pauses != nil {
		body.Pauses = new(int(*point.Pauses))
	}

	if point.Restarts != nil {
		body.Restarts = new(int(*point.Restarts))
	}

	body.SegmentWriteAvgMillis = point.SegmentWriteAvgMillis
	body.SegmentWriteMaxMillis = point.SegmentWriteMaxMillis

	if point.SessionStarts != nil {
		body.SessionStarts = new(int(*point.SessionStarts))
	}

	body.SourceWriteBytesPerSecond = point.SourceWriteBytesPerSecond
	body.StartedAt = point.StartedAt

	return body, nil
}
