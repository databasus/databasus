package physical

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
)

type WalStreamStatisticsPeriod string

const (
	WalStreamStatisticsPeriodHour WalStreamStatisticsPeriod = "HOUR"
	WalStreamStatisticsPeriodDay  WalStreamStatisticsPeriod = "DAY"
	WalStreamStatisticsPeriodWeek WalStreamStatisticsPeriod = "WEEK"
)

func (p WalStreamStatisticsPeriod) GetDurations() (period, bucket time.Duration, isKnown bool) {
	switch p {
	case WalStreamStatisticsPeriodHour:
		return time.Hour, time.Minute, true
	case WalStreamStatisticsPeriodDay:
		return 24 * time.Hour, 4 * time.Minute, true
	case WalStreamStatisticsPeriodWeek:
		return 7 * 24 * time.Hour, 28 * time.Minute, true
	default:
		return 0, 0, false
	}
}

type GetWalStreamStatisticsRequest struct {
	Period WalStreamStatisticsPeriod
}

type GetWalStreamStatisticsResponse struct {
	Health WalStreamHealth

	LatestRestorableAt *time.Time

	RestorableRanges []WalStreamTimeRange

	Streamer  WalStreamStreamer
	Current   *WalStreamCurrent
	KeepingUp streamhealth.KeepingUp
	Series    WalStreamSeries
}

type WalStreamTimeRange struct {
	From time.Time
	To   time.Time
}

type WalStreamHealth struct {
	Level   streamhealth.HealthLevel
	Reasons []streamhealth.Reason
	Notices []streamhealth.Notice

	IsFullBackupPending bool
}

type WalStreamStreamer struct {
	State            streamhealth.StreamerState
	SessionStartedAt *time.Time
	FailingSince     *time.Time
	RestartsLastHour int
}

type WalStreamCurrent struct {
	SampledAt         time.Time
	LastArchivedWalAt *time.Time
	SessionStartedAt  time.Time

	SourceState            catalog.WalStreamSourceState
	SourceUnreachableSince *time.Time

	SourceCurrentLsn   *string
	ArchiveProgressLsn *string
	SourceWrittenBytes *int64

	ArchivedSegments         int
	ArchivedWalBytes         int64
	ArchivedStoredBytes      int64
	WalSegmentSizeBytes      int64
	HighestConcurrentUploads int
	SegmentWriteAvgMillis    *int64
	SegmentWriteMaxMillis    *int64

	SlotWalStatus        *string
	SlotExtraWalSince    *time.Time
	SlotLagBytes         *int64
	SlotSafeWalSizeBytes *int64

	UnarchivedWalBytes  *int64
	ArchiveStalledSince *time.Time

	LocalQueueBytes      int64
	LocalQueuePauseBytes int64

	ReceiverRestarts       int
	ReceiverPauses         int
	IsReceiverRunning      bool
	IsForcedRotationDenied bool
}

type WalStreamSeries struct {
	Period        WalStreamStatisticsPeriod
	BucketSeconds int64
	From          time.Time
	To            time.Time
	Points        []*catalog.WalStreamSeriesPoint
}
