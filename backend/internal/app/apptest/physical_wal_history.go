package apptest

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type PhysicalWalHistoryFileSeed struct {
	ID         uuid.UUID `json:"id"         gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;type:uuid;not null"`

	TimelineID      int    `json:"timelineId"      gorm:"column:timeline_id;type:int;not null"`
	FileName        string `json:"fileName"        gorm:"column:file_name;type:text;not null"`
	HistoryFilename string `json:"historyFilename" gorm:"column:history_filename;type:text;not null"`

	CompressedSizeMb float64 `json:"compressedSizeMb" gorm:"column:compressed_size_mb;type:double precision;not null;default:0"`

	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (a *App) InsertPhysicalWalHistoryFile(tb testing.TB, seed PhysicalWalHistoryFileSeed) PhysicalWalHistoryFileSeed {
	tb.Helper()
	if seed.ID == uuid.Nil {
		seed.ID = uuid.New()
	}
	if seed.TimelineID == 0 {
		seed.TimelineID = 1
	}
	if seed.CreatedAt.IsZero() {
		seed.CreatedAt = time.Now().UTC()
	}
	a.insertPhysicalCatalogRow(tb, "backups_physical_wal_history_files", &seed)
	return seed
}
