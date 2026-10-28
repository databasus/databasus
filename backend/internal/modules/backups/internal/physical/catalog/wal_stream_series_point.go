package catalog

import (
	"time"
)

// One interval of the WAL stream statistics series. Every value is nil when no
// sample falls into the interval, so the hole stays a hole instead of a zero.
type WalStreamSeriesPoint struct {
	StartedAt time.Time `gorm:"column:started_at"`

	SourceWriteBytesPerSecond    *float64 `gorm:"column:source_write_bytes_per_second"`
	ArchiveBytesPerSecond        *float64 `gorm:"column:archive_bytes_per_second"`
	ArchivedStoredBytesPerSecond *float64 `gorm:"column:archived_stored_bytes_per_second"`

	MaxSlotLagBytes       *int64 `gorm:"column:max_slot_lag_bytes"`
	MaxUnarchivedWalBytes *int64 `gorm:"column:max_unarchived_wal_bytes"`
	MaxLocalQueueBytes    *int64 `gorm:"column:max_local_queue_bytes"`
	LocalQueuePauseBytes  *int64 `gorm:"column:local_queue_pause_bytes"`

	Restarts      *int64 `gorm:"column:restarts"`
	Pauses        *int64 `gorm:"column:pauses"`
	SessionStarts *int64 `gorm:"column:session_starts"`

	HighestConcurrentUploads *int64 `gorm:"column:highest_concurrent_uploads"`
	SegmentWriteAvgMillis    *int64 `gorm:"column:segment_write_avg_millis"`
	SegmentWriteMaxMillis    *int64 `gorm:"column:segment_write_max_millis"`
}
