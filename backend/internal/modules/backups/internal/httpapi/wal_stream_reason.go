package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
)

func toWalStreamReason(reason streamhealth.Reason) (WalStreamReason, error) {
	body := WalStreamReason{}

	switch string(reason.Code) {
	case "STREAMING_STOPPED",
		"SOURCE_UNREACHABLE",
		"WAL_CHAIN_BROKEN",
		"SLOT_MISSING",
		"SLOT_LOST",
		"ARCHIVE_NOT_ADVANCING",
		"ARCHIVE_SLOWER_THAN_SOURCE",
		"SLOT_KEEPS_EXTRA_WAL",
		"SLOT_LIMIT_APPROACHING",
		"NEAR_SLOT_REBUILD",
		"LOCAL_QUEUE_FILLING",
		"FREQUENT_RESTARTS",
		"FORCED_ROTATION_REFUSED":
	default:
		return WalStreamReason{}, errUnknownPhysicalValue
	}

	body.Code = WalStreamReasonCode(reason.Code)
	body.Count = reason.Count
	body.EstimatedSeconds = reason.EstimatedSeconds

	switch string(reason.Level) {
	case "BROKEN", "NEEDS_ATTENTION", "INFO":
	default:
		return WalStreamReason{}, errUnknownPhysicalValue
	}

	body.Level = WalStreamReasonLevel(reason.Level)
	body.Since = reason.Since
	body.ThresholdBytes = reason.ThresholdBytes
	body.ValueBytes = reason.ValueBytes

	return body, nil
}
