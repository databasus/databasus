package physical

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type GetWalStreamLiveResponse struct {
	IsStreamerRunning bool
	Current           *WalStreamLive
}

type WalStreamLive struct {
	MeasuredAt time.Time

	SourceProbedAt       *time.Time
	SourceState          *catalog.WalStreamSourceState
	SourceCurrentLsn     *string
	SlotWalStatus        *string
	SlotLagBytes         *int64
	SlotSafeWalSizeBytes *int64

	ArchiveProgressLsn *string
	LastArchivedWalAt  *time.Time
	UnarchivedWalBytes *int64

	LocalQueueBytes      int64
	LocalQueuePauseBytes int64
	IsReceiverRunning    bool
}
