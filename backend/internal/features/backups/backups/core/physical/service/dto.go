package physical_service

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
)

type DeleteOrphanWalSegmentsSpec struct {
	DatabaseID      uuid.UUID
	TimelineID      int
	Span            chain_view.LSNRange
	WalByteBudgetMB float64
}

// PhysicalBackupListRow is one row of the merged, paginated backup list across
// the FULL, incremental and committed-WAL tables. Type-only columns are NULL for
// the other types (scanned into the pointer fields); LSNs arrive as text. It is
// the catalog-level shape the API maps to its presentation DTO.
type PhysicalBackupListRow struct {
	ID                        uuid.UUID  `gorm:"column:id"`
	Type                      string     `gorm:"column:type"`
	Status                    string     `gorm:"column:status"`
	TimelineID                int        `gorm:"column:timeline_id"`
	StartLSN                  string     `gorm:"column:start_lsn"`
	StopLSN                   string     `gorm:"column:stop_lsn"`
	RootFullBackupID          *uuid.UUID `gorm:"column:root_full_backup_id"`
	ParentIncrementalBackupID *uuid.UUID `gorm:"column:parent_incremental_backup_id"`
	WalFilename               *string    `gorm:"column:wal_filename"`
	SizeMb                    float64    `gorm:"column:size_mb"`
	FailMessage               *string    `gorm:"column:fail_message"`
	CreatedAt                 time.Time  `gorm:"column:created_at"`
	CompletedAt               *time.Time `gorm:"column:completed_at"`
}

// BackupListFilter narrows ListBackups / CountBackups. An empty slice (or nil
// BeforeDate) is "no filter on that dimension". Types and Statuses match the
// synthetic columns the UNION projects ('WAL'/'COMPLETED' for WAL rows) and
// match any of their values; BeforeDate filters on created_at.
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

// DependentsSummary counts a chain's dependents and total on-disk size without
// deleting anything. Powers "what would DeleteFull remove?" UI/audit.
type DependentsSummary struct {
	RootFullBackupID uuid.UUID
	WalSegments      int
	Incrementals     int
	HistoryFiles     int
	TotalSizeMB      float64
}

type DatabasePhysicalBackupTotals struct {
	DatabaseID            uuid.UUID `gorm:"column:database_id"`
	BackupsCount          int64     `gorm:"column:backups_count"`
	CompletedBackupsCount int64     `gorm:"column:completed_backups_count"`
	FailedBackupsCount    int64     `gorm:"column:failed_backups_count"`
	CompletedBackupSizeMb float64   `gorm:"column:completed_backup_size_mb"`
	WalSizeMb             float64   `gorm:"column:wal_size_mb"`
}

type PhysicalBackupTotals struct {
	BackupsCount          int64   `gorm:"column:backups_count"`
	CompletedBackupSizeMb float64 `gorm:"column:completed_backup_size_mb"`
	WalSizeMb             float64 `gorm:"column:wal_size_mb"`
}
