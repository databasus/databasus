package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamLiveValues(live physical.WalStreamLive) (WalStreamLiveValues, error) {
	body := WalStreamLiveValues{}
	body.ArchiveProgressLsn = live.ArchiveProgressLsn
	body.IsReceiverRunning = live.IsReceiverRunning
	body.LastArchivedWalAt = live.LastArchivedWalAt
	body.LocalQueueBytes = live.LocalQueueBytes
	body.LocalQueuePauseBytes = live.LocalQueuePauseBytes
	body.MeasuredAt = live.MeasuredAt
	body.SlotLagBytes = live.SlotLagBytes
	body.SlotSafeWalSizeBytes = live.SlotSafeWalSizeBytes
	body.SlotWalStatus = live.SlotWalStatus
	body.SourceCurrentLsn = live.SourceCurrentLsn
	body.SourceProbedAt = live.SourceProbedAt

	if live.SourceState != nil {
		switch string(*live.SourceState) {
		case "REACHABLE", "UNREACHABLE", "SLOT_MISSING":
		default:
			return WalStreamLiveValues{}, errUnknownPhysicalValue
		}

		body.SourceState = new(WalStreamSourceState(*live.SourceState))
	}

	body.UnarchivedWalBytes = live.UnarchivedWalBytes

	return body, nil
}
