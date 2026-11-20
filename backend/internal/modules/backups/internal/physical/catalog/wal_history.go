package catalog

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type WalHistoryFile struct {
	ID         uuid.UUID `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `gorm:"column:storage_id;type:uuid;not null"`

	TimelineID      int    `gorm:"column:timeline_id;type:int;not null"`
	FileName        string `gorm:"column:file_name;type:text;not null"`
	HistoryFilename string `gorm:"column:history_filename;type:text;not null"`

	// ParentTimelineID and ForkLSN are both empty on rows published before fork points were recorded.
	// Such a row's fork point is unknown, so no line crosses it.
	ParentTimelineID *int         `gorm:"column:parent_timeline_id;type:int"`
	ForkLSN          *walmath.LSN `gorm:"column:fork_lsn;type:pg_lsn"`
	IsConflicting    bool         `gorm:"column:is_conflicting;type:boolean;not null;default:false"`

	CompressedSizeMb float64 `gorm:"column:compressed_size_mb;type:double precision;not null;default:0"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (WalHistoryFile) TableName() string {
	return "backups_physical_wal_history_files"
}

// HasUsableFork reports whether a line may cross this row's fork point: it is recorded and no
// conflicting history was seen for the timeline.
func (f *WalHistoryFile) HasUsableFork() bool {
	return f.ParentTimelineID != nil && f.ForkLSN != nil && !f.IsConflicting
}
