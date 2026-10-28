package physical

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type GetWalStreamLiveResponse struct {
	IsStreamerRunning bool           `json:"isStreamerRunning"`
	Current           *WalStreamLive `json:"current"`
}

type WalStreamLive struct {
	MeasuredAt time.Time `json:"measuredAt"`

	SourceProbedAt       *time.Time                    `json:"sourceProbedAt"`
	SourceState          *catalog.WalStreamSourceState `json:"sourceState"`
	SourceCurrentLsn     *string                       `json:"sourceCurrentLsn"`
	SlotWalStatus        *string                       `json:"slotWalStatus"`
	SlotLagBytes         *int64                        `json:"slotLagBytes"`
	SlotSafeWalSizeBytes *int64                        `json:"slotSafeWalSizeBytes"`

	ArchiveProgressLsn *string    `json:"archiveProgressLsn"`
	LastArchivedWalAt  *time.Time `json:"lastArchivedWalAt"`
	UnarchivedWalBytes *int64     `json:"unarchivedWalBytes"`

	LocalQueueBytes      int64 `json:"localQueueBytes"`
	LocalQueuePauseBytes int64 `json:"localQueuePauseBytes"`
	IsReceiverRunning    bool  `json:"isReceiverRunning"`
}
