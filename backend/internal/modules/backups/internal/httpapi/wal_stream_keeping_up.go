package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
)

func toWalStreamKeepingUp(keepingUp streamhealth.KeepingUp) (WalStreamKeepingUp, error) {
	body := WalStreamKeepingUp{}
	body.ArchiveBytesPerSecond = keepingUp.ArchiveBytesPerSecond
	body.EstimatedCatchUpSeconds = keepingUp.EstimatedCatchUpSeconds
	body.EstimatedSlotLimitSeconds = keepingUp.EstimatedSlotLimitSeconds
	body.EstimatedSlotRebuildSeconds = keepingUp.EstimatedSlotRebuildSeconds
	body.SlotLagBytes = keepingUp.SlotLagBytes
	body.SlotLagChangeBytesPerSecond = keepingUp.SlotLagChangeBytesPerSecond
	body.SlotRebuildLagBytes = keepingUp.SlotRebuildLagBytes
	body.SourceWriteBytesPerSecond = keepingUp.SourceWriteBytesPerSecond

	switch string(keepingUp.State) {
	case "UNKNOWN", "FALLING_BEHIND", "CATCHING_UP", "IDLE", "KEEPING_UP":
	default:
		return WalStreamKeepingUp{}, errUnknownPhysicalValue
	}

	body.State = WalStreamKeepingUpState(keepingUp.State)
	body.UnarchivedWalBytes = keepingUp.UnarchivedWalBytes
	body.UnarchivedWalChangeBytesPerSecond = keepingUp.UnarchivedWalChangeBytesPerSecond

	return body, nil
}
