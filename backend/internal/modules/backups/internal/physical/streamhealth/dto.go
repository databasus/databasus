package streamhealth

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type VerdictInput struct {
	// The newest sample of the database, whatever its age; nil when none exists.
	NewestSample *catalog.WalStreamSample

	// The samples of the last hour, oldest first. The newest sample is the last
	// one whenever it is younger than an hour.
	RecentSamples []*catalog.WalStreamSample

	// Nil when no streamer has ever run for the database.
	Streamer *catalog.WalStreamer

	SessionStartsLastHour int

	SlotRebuildLagBytes int64

	IsStreamingExpected    bool
	HasCompletedFullBackup bool

	// The WAL gaps of the newest chain that the latest restore cannot cross.
	// A gap a completed incremental bridged is already left out.
	RestoreBlockingGaps []catalog.LSNRange

	Now time.Time
}

type Reason struct {
	Code             ReasonCode  `json:"code"`
	Level            ReasonLevel `json:"level"`
	Since            *time.Time  `json:"since,omitempty"`
	ValueBytes       *int64      `json:"valueBytes,omitempty"`
	ThresholdBytes   *int64      `json:"thresholdBytes,omitempty"`
	Count            *int        `json:"count,omitempty"`
	EstimatedSeconds *int64      `json:"estimatedSeconds,omitempty"`
}

type Notice struct {
	Code  NoticeCode  `json:"code"`
	Level ReasonLevel `json:"level"`
	Since *time.Time  `json:"since,omitempty"`
}

type Verdict struct {
	Level   HealthLevel `json:"level"`
	Reasons []Reason    `json:"reasons"`
	Notices []Notice    `json:"notices"`
}

type KeepingUp struct {
	State KeepingUpState `json:"state"`

	SourceWriteBytesPerSecond *float64 `json:"sourceWriteBytesPerSecond"`
	ArchiveBytesPerSecond     *float64 `json:"archiveBytesPerSecond"`

	UnarchivedWalBytes                *int64   `json:"unarchivedWalBytes"`
	UnarchivedWalChangeBytesPerSecond *float64 `json:"unarchivedWalChangeBytesPerSecond"`

	SlotLagBytes                *int64   `json:"slotLagBytes"`
	SlotLagChangeBytesPerSecond *float64 `json:"slotLagChangeBytesPerSecond"`

	SlotRebuildLagBytes int64 `json:"slotRebuildLagBytes"`

	EstimatedCatchUpSeconds     *int64 `json:"estimatedCatchUpSeconds"`
	EstimatedSlotRebuildSeconds *int64 `json:"estimatedSlotRebuildSeconds"`
	EstimatedSlotLimitSeconds   *int64 `json:"estimatedSlotLimitSeconds"`
}
