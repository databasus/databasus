package streamstatistics_test

import "time"

type statisticsResponse struct {
	Health streamHealth `json:"health"`

	LatestRestorableAt *time.Time `json:"latestRestorableAt"`

	RestorableRanges []restorableTimeRange `json:"restorableRanges"`

	Streamer  streamerReading `json:"streamer"`
	Current   *currentSample  `json:"current"`
	KeepingUp keepingUp       `json:"keepingUp"`
	Series    streamSeries    `json:"series"`
}

type restorableTimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type streamHealth struct {
	Level   string         `json:"level"`
	Reasons []healthReason `json:"reasons"`
	Notices []healthNotice `json:"notices"`

	IsFullBackupPending bool `json:"isFullBackupPending"`
}

type streamerReading struct {
	State            string     `json:"state"`
	SessionStartedAt *time.Time `json:"sessionStartedAt"`
	FailingSince     *time.Time `json:"failingSince"`
	RestartsLastHour int        `json:"restartsLastHour"`
}

type currentSample struct {
	SampledAt         time.Time  `json:"sampledAt"`
	LastArchivedWalAt *time.Time `json:"lastArchivedWalAt"`
	SessionStartedAt  time.Time  `json:"sessionStartedAt"`

	SourceState            string     `json:"sourceState"`
	SourceUnreachableSince *time.Time `json:"sourceUnreachableSince"`

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

type streamSeries struct {
	Period        string         `json:"period"`
	BucketSeconds int64          `json:"bucketSeconds"`
	From          time.Time      `json:"from"`
	To            time.Time      `json:"to"`
	Points        []*seriesPoint `json:"points"`
}

type healthReason struct {
	Code             string     `json:"code"`
	Level            string     `json:"level"`
	Since            *time.Time `json:"since,omitempty"`
	ValueBytes       *int64     `json:"valueBytes,omitempty"`
	ThresholdBytes   *int64     `json:"thresholdBytes,omitempty"`
	Count            *int       `json:"count,omitempty"`
	EstimatedSeconds *int64     `json:"estimatedSeconds,omitempty"`
}

type healthNotice struct {
	Code  string     `json:"code"`
	Level string     `json:"level"`
	Since *time.Time `json:"since,omitempty"`
}

type keepingUp struct {
	State string `json:"state"`

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

type seriesPoint struct {
	StartedAt time.Time `json:"startedAt"`

	SourceWriteBytesPerSecond    *float64 `json:"sourceWriteBytesPerSecond"`
	ArchiveBytesPerSecond        *float64 `json:"archiveBytesPerSecond"`
	ArchivedStoredBytesPerSecond *float64 `json:"archivedStoredBytesPerSecond"`

	MaxSlotLagBytes       *int64 `json:"maxSlotLagBytes"`
	MaxUnarchivedWalBytes *int64 `json:"maxUnarchivedWalBytes"`
	MaxLocalQueueBytes    *int64 `json:"maxLocalQueueBytes"`
	LocalQueuePauseBytes  *int64 `json:"localQueuePauseBytes"`

	Restarts      *int64 `json:"restarts"`
	Pauses        *int64 `json:"pauses"`
	SessionStarts *int64 `json:"sessionStarts"`

	HighestConcurrentUploads *int64 `json:"highestConcurrentUploads"`
	SegmentWriteAvgMillis    *int64 `json:"segmentWriteAvgMillis"`
	SegmentWriteMaxMillis    *int64 `json:"segmentWriteMaxMillis"`
}
