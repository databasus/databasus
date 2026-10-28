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
	Code             ReasonCode
	Level            ReasonLevel
	Since            *time.Time
	ValueBytes       *int64
	ThresholdBytes   *int64
	Count            *int
	EstimatedSeconds *int64
}

type Notice struct {
	Code  NoticeCode
	Level ReasonLevel
	Since *time.Time
}

type Verdict struct {
	Level   HealthLevel
	Reasons []Reason
	Notices []Notice
}

type KeepingUp struct {
	State KeepingUpState

	SourceWriteBytesPerSecond *float64
	ArchiveBytesPerSecond     *float64

	UnarchivedWalBytes                *int64
	UnarchivedWalChangeBytesPerSecond *float64

	SlotLagBytes                *int64
	SlotLagChangeBytesPerSecond *float64

	SlotRebuildLagBytes int64

	EstimatedCatchUpSeconds     *int64
	EstimatedSlotRebuildSeconds *int64
	EstimatedSlotLimitSeconds   *int64
}
