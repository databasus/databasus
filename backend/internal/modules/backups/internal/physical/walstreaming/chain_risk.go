package walstreaming

import "time"

const (
	ChainRiskReasonSlotRetention  = "SLOT_WAL_RETENTION"
	ChainRiskReasonArchiveStale   = "WAL_ARCHIVE_STALE"
	ChainRiskReasonRotationDenied = "WAL_ROTATION_DENIED"
)

// Payload of WalStreamSpec.OnChainAtRisk. SlotWalStatus and LagBytes describe the
// slot; LastArchivedWalAt is set only for ChainRiskReasonArchiveStale, where the
// question is how far the recovery point has fallen behind.
type ChainRiskReport struct {
	Reason            string
	SlotWalStatus     string
	LagBytes          int64
	LastArchivedWalAt *time.Time
}
