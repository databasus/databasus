package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamCurrent(sample physical.WalStreamCurrent) (WalStreamCurrent, error) {
	body := WalStreamCurrent{}
	body.ArchiveProgressLsn = sample.ArchiveProgressLsn
	body.ArchiveStalledSince = sample.ArchiveStalledSince
	body.ArchivedSegments = sample.ArchivedSegments
	body.ArchivedStoredBytes = sample.ArchivedStoredBytes
	body.ArchivedWalBytes = sample.ArchivedWalBytes
	body.HighestConcurrentUploads = sample.HighestConcurrentUploads
	body.IsForcedRotationDenied = sample.IsForcedRotationDenied
	body.IsReceiverRunning = sample.IsReceiverRunning
	body.LastArchivedWalAt = sample.LastArchivedWalAt
	body.LocalQueueBytes = sample.LocalQueueBytes
	body.LocalQueuePauseBytes = sample.LocalQueuePauseBytes
	body.ReceiverPauses = sample.ReceiverPauses
	body.ReceiverRestarts = sample.ReceiverRestarts
	body.SampledAt = sample.SampledAt
	body.SegmentWriteAvgMillis = sample.SegmentWriteAvgMillis
	body.SegmentWriteMaxMillis = sample.SegmentWriteMaxMillis
	body.SessionStartedAt = sample.SessionStartedAt
	body.SlotExtraWalSince = sample.SlotExtraWalSince
	body.SlotLagBytes = sample.SlotLagBytes
	body.SlotSafeWalSizeBytes = sample.SlotSafeWalSizeBytes
	body.SlotWalStatus = sample.SlotWalStatus
	body.SourceCurrentLsn = sample.SourceCurrentLsn

	switch string(sample.SourceState) {
	case "REACHABLE", "UNREACHABLE", "SLOT_MISSING":
	default:
		return WalStreamCurrent{}, errUnknownPhysicalValue
	}

	body.SourceState = WalStreamSourceState(sample.SourceState)
	body.SourceUnreachableSince = sample.SourceUnreachableSince
	body.SourceWrittenBytes = sample.SourceWrittenBytes
	body.UnarchivedWalBytes = sample.UnarchivedWalBytes
	body.WalSegmentSizeBytes = sample.WalSegmentSizeBytes

	return body, nil
}
