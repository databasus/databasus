package catalog

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// One sample of a WAL-streaming database's archiving health, taken by the
// streamer's lag monitor. Source-side values are nil whenever SourceState is not
// REACHABLE; on a reachable sample a nil SlotSafeWalSizeBytes means the source
// sets no limit on the WAL it keeps for the slot.
type WalStreamSample struct {
	ID         uuid.UUID `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `gorm:"column:database_id;type:uuid;not null"`

	SampledAt        time.Time `gorm:"column:sampled_at;not null"`
	SessionStartedAt time.Time `gorm:"column:session_started_at;not null"`

	SourceState            WalStreamSourceState `gorm:"column:source_state;type:text;not null"`
	SourceUnreachableSince *time.Time           `gorm:"column:source_unreachable_since"`

	SourceCurrentLSN   *walmath.LSN `gorm:"column:source_current_lsn;type:pg_lsn"`
	ArchiveProgressLSN *walmath.LSN `gorm:"column:archive_progress_lsn;type:pg_lsn"`
	SourceWrittenBytes *int64       `gorm:"column:source_written_bytes"`

	ArchivedSegments         int    `gorm:"column:archived_segments;not null"`
	ArchivedWalBytes         int64  `gorm:"column:archived_wal_bytes;not null"`
	ArchivedStoredBytes      int64  `gorm:"column:archived_stored_bytes;not null"`
	WalSegmentSizeBytes      int64  `gorm:"column:wal_segment_size_bytes;not null"`
	HighestConcurrentUploads int    `gorm:"column:highest_concurrent_uploads;not null"`
	SegmentWriteAvgMillis    *int64 `gorm:"column:segment_write_avg_millis"`
	SegmentWriteMaxMillis    *int64 `gorm:"column:segment_write_max_millis"`

	SlotWalStatus        *string    `gorm:"column:slot_wal_status;type:text"`
	SlotExtraWalSince    *time.Time `gorm:"column:slot_extra_wal_since"`
	SlotLagBytes         *int64     `gorm:"column:slot_lag_bytes"`
	SlotSafeWalSizeBytes *int64     `gorm:"column:slot_safe_wal_size_bytes"`

	UnarchivedWalBytes  *int64     `gorm:"column:unarchived_wal_bytes"`
	ArchiveStalledSince *time.Time `gorm:"column:archive_stalled_since"`

	LocalQueueBytes      int64 `gorm:"column:local_queue_bytes;not null"`
	LocalQueuePauseBytes int64 `gorm:"column:local_queue_pause_bytes;not null"`

	ReceiverRestarts       int  `gorm:"column:receiver_restarts;not null"`
	ReceiverPauses         int  `gorm:"column:receiver_pauses;not null"`
	IsReceiverRunning      bool `gorm:"column:is_receiver_running;not null"`
	IsForcedRotationDenied bool `gorm:"column:is_forced_rotation_denied;not null"`
}

func (WalStreamSample) TableName() string {
	return "backups_physical_wal_stream_samples"
}

func (s *WalStreamSample) IsSourceReachable() bool {
	return s.SourceState == WalStreamSourceStateReachable
}
