package catalog

import (
	"time"
)

// One interval of the WAL stream statistics series. Every value is nil when no
// sample falls into the interval, so the hole stays a hole instead of a zero.
type WalStreamSeriesPoint struct {
	StartedAt time.Time `json:"startedAt" gorm:"column:started_at"`

	SourceWriteBytesPerSecond    *float64 `json:"sourceWriteBytesPerSecond"    gorm:"column:source_write_bytes_per_second"`
	ArchiveBytesPerSecond        *float64 `json:"archiveBytesPerSecond"        gorm:"column:archive_bytes_per_second"`
	ArchivedStoredBytesPerSecond *float64 `json:"archivedStoredBytesPerSecond" gorm:"column:archived_stored_bytes_per_second"`

	MaxSlotLagBytes       *int64 `json:"maxSlotLagBytes"       gorm:"column:max_slot_lag_bytes"`
	MaxUnarchivedWalBytes *int64 `json:"maxUnarchivedWalBytes" gorm:"column:max_unarchived_wal_bytes"`
	MaxLocalQueueBytes    *int64 `json:"maxLocalQueueBytes"    gorm:"column:max_local_queue_bytes"`
	LocalQueuePauseBytes  *int64 `json:"localQueuePauseBytes"  gorm:"column:local_queue_pause_bytes"`

	Restarts      *int64 `json:"restarts"      gorm:"column:restarts"`
	Pauses        *int64 `json:"pauses"        gorm:"column:pauses"`
	SessionStarts *int64 `json:"sessionStarts" gorm:"column:session_starts"`

	HighestConcurrentUploads *int64 `json:"highestConcurrentUploads" gorm:"column:highest_concurrent_uploads"`
	SegmentWriteAvgMillis    *int64 `json:"segmentWriteAvgMillis"    gorm:"column:segment_write_avg_millis"`
	SegmentWriteMaxMillis    *int64 `json:"segmentWriteMaxMillis"    gorm:"column:segment_write_max_millis"`
}
