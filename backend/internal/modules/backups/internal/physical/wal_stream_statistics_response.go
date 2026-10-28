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
	Period WalStreamStatisticsPeriod `form:"period" binding:"required"`
}

type GetWalStreamStatisticsResponse struct {
	Health WalStreamHealth `json:"health"`

	LatestRestorableAt *time.Time `json:"latestRestorableAt"`

	RestorableRanges []WalStreamTimeRange `json:"restorableRanges"`

	Streamer  WalStreamStreamer      `json:"streamer"`
	Current   *WalStreamCurrent      `json:"current"`
	KeepingUp streamhealth.KeepingUp `json:"keepingUp"`
	Series    WalStreamSeries        `json:"series"`
}

type WalStreamTimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type WalStreamHealth struct {
	Level   streamhealth.HealthLevel `json:"level"`
	Reasons []streamhealth.Reason    `json:"reasons"`
	Notices []streamhealth.Notice    `json:"notices"`

	IsFullBackupPending bool `json:"isFullBackupPending"`
}

type WalStreamStreamer struct {
	State            streamhealth.StreamerState `json:"state"`
	SessionStartedAt *time.Time                 `json:"sessionStartedAt"`
	FailingSince     *time.Time                 `json:"failingSince"`
	RestartsLastHour int                        `json:"restartsLastHour"`
}

type WalStreamCurrent struct {
	SampledAt         time.Time  `json:"sampledAt"`
	LastArchivedWalAt *time.Time `json:"lastArchivedWalAt"`
	SessionStartedAt  time.Time  `json:"sessionStartedAt"`

	SourceState            catalog.WalStreamSourceState `json:"sourceState"`
	SourceUnreachableSince *time.Time                   `json:"sourceUnreachableSince"`

	SourceCurrentLsn   *string `json:"sourceCurrentLsn"`
	ArchiveProgressLsn *string `json:"archiveProgressLsn"`
	SourceWrittenBytes *int64  `json:"sourceWrittenBytes"`

	ArchivedSegments         int    `json:"archivedSegments"`
	ArchivedWalBytes         int64  `json:"archivedWalBytes"`
	ArchivedStoredBytes      int64  `json:"archivedStoredBytes"`
	WalSegmentSizeBytes      int64  `json:"walSegmentSizeBytes"`
	HighestConcurrentUploads int    `json:"highestConcurrentUploads"`
	SegmentWriteAvgMillis    *int64 `json:"segmentWriteAvgMillis"`
	SegmentWriteMaxMillis    *int64 `json:"segmentWriteMaxMillis"`

	SlotWalStatus        *string    `json:"slotWalStatus"`
	SlotExtraWalSince    *time.Time `json:"slotExtraWalSince"`
	SlotLagBytes         *int64     `json:"slotLagBytes"`
	SlotSafeWalSizeBytes *int64     `json:"slotSafeWalSizeBytes"`

	UnarchivedWalBytes  *int64     `json:"unarchivedWalBytes"`
	ArchiveStalledSince *time.Time `json:"archiveStalledSince"`

	LocalQueueBytes      int64 `json:"localQueueBytes"`
	LocalQueuePauseBytes int64 `json:"localQueuePauseBytes"`

	ReceiverRestarts       int  `json:"receiverRestarts"`
	ReceiverPauses         int  `json:"receiverPauses"`
	IsReceiverRunning      bool `json:"isReceiverRunning"`
	IsForcedRotationDenied bool `json:"isForcedRotationDenied"`
}

type WalStreamSeries struct {
	Period        WalStreamStatisticsPeriod       `json:"period"`
	BucketSeconds int64                           `json:"bucketSeconds"`
	From          time.Time                       `json:"from"`
	To            time.Time                       `json:"to"`
	Points        []*catalog.WalStreamSeriesPoint `json:"points"`
}
