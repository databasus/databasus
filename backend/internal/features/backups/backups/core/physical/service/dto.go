package physical_service

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
)

type DeleteOrphanWalSegmentsSpec struct {
	DatabaseID      uuid.UUID
	TimelineID      int
	Span            chain_view.LSNRange
	WalByteBudgetMB float64
}

// PhysicalBackupListRow is one row of the merged, paginated backup list: a FULL,
// an incremental, or a WAL range that collapses consecutive committed segments.
// Type-only columns are NULL for the other types (scanned into the pointer
// fields); LSNs arrive as text. A WAL range takes the ID of its lowest-LSN
// segment, so its identity survives growth at the newest end. SortTime is the
// row's position in the newest-first list: created_at for a backup, the newest
// receive time for a WAL range.
type PhysicalBackupListRow struct {
	ID                        uuid.UUID  `gorm:"column:id"`
	Type                      string     `gorm:"column:type"`
	Status                    string     `gorm:"column:status"`
	TimelineID                int        `gorm:"column:timeline_id"`
	StartLSN                  string     `gorm:"column:start_lsn"`
	StopLSN                   string     `gorm:"column:stop_lsn"`
	RootFullBackupID          *uuid.UUID `gorm:"column:root_full_backup_id"`
	ParentIncrementalBackupID *uuid.UUID `gorm:"column:parent_incremental_backup_id"`
	FromWalFilename           *string    `gorm:"column:from_wal_filename"`
	ToWalFilename             *string    `gorm:"column:to_wal_filename"`
	WalSegmentsCount          *int       `gorm:"column:wal_segments_count"`
	HasGapBefore              bool       `gorm:"column:has_gap_before"`
	SizeMb                    float64    `gorm:"column:size_mb"`
	FailMessage               *string    `gorm:"column:fail_message"`
	CreatedAt                 time.Time  `gorm:"column:created_at"`
	CompletedAt               *time.Time `gorm:"column:completed_at"`
	SortTime                  time.Time  `gorm:"column:sort_time"`
}

// BackupListPosition is the keyset position ListBackups pages after: the sort
// time and ID of the last row the caller already holds.
type BackupListPosition struct {
	SortTime time.Time
	ID       uuid.UUID
}

// BackupListFilter narrows ListBackups. An empty slice (or nil BeforeDate) is
// "no filter on that dimension". Types and Statuses match the synthetic columns
// the UNION projects ('WAL'/'COMPLETED' for WAL segments) and match any of their
// values; BeforeDate filters on created_at. The filter selects segments before
// they are grouped into WAL ranges.
type BackupListFilter struct {
	Types      []string
	Statuses   []string
	BeforeDate *time.Time
}

// buildClause renders the filter as an outer WHERE applied to the merged
// subquery. Returns an empty string (and no args) when nothing is set. All
// values are bound, never interpolated - GORM expands a slice arg bound to
// `IN ?` into the right number of placeholders.
func (f BackupListFilter) buildClause() (string, []any) {
	var conditions []string
	var args []any

	if len(f.Types) > 0 {
		conditions = append(conditions, "type IN ?")
		args = append(args, f.Types)
	}

	if len(f.Statuses) > 0 {
		conditions = append(conditions, "status IN ?")
		args = append(args, f.Statuses)
	}

	if f.BeforeDate != nil {
		conditions = append(conditions, "created_at < ?")
		args = append(args, *f.BeforeDate)
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// DeletedSummary reports what a single DeleteFull / DeleteChainDependentsKeepFull
// call removed. The cleaner logs it per tick. ChainFullyDeleted is false when
// the per-tick WAL byte budget capped the call before the FULL row was reached
// (or when the call intentionally keeps the FULL) — the caller resumes on the
// next tick.
type DeletedSummary struct {
	RootFullBackupID  uuid.UUID
	WalSegments       int
	Incrementals      int
	HistoryFiles      int
	BytesDeletedMB    float64
	ChainFullyDeleted bool
}

type WalStreamStatisticsQuery struct {
	DatabaseID     uuid.UUID
	Period         time.Duration
	BucketDuration time.Duration

	SlotRebuildLagBytes    int64
	IsStreamingExpected    bool
	HasCompletedFullBackup bool
	RestoreBlockingGaps    []chain_view.LSNRange

	Now time.Time
}

// Everything the WAL stream statistics need from the catalog, decided at one
// moment. NewestSample, Streamer and LastArchivedWalAt are nil when they do not
// exist yet.
type WalStreamStatistics struct {
	Verdict          wal_health.Verdict
	KeepingUp        wal_health.KeepingUp
	StreamerState    wal_health.StreamerState
	RestartsLastHour int

	Streamer          *physical_models.PhysicalWalStreamer
	NewestSample      *physical_models.PhysicalWalStreamSample
	LastArchivedWalAt *time.Time

	SeriesFrom   time.Time
	SeriesTo     time.Time
	SeriesPoints []*physical_dto.PhysicalWalStreamSeriesPoint
}
