package apptest

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type PhysicalWalSegmentSeed struct {
	ID         uuid.UUID `json:"id"         gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;type:uuid;not null"`

	TimelineID  int     `json:"timelineId"  gorm:"column:timeline_id;type:int;not null"`
	FileName    *string `json:"fileName"    gorm:"column:file_name;type:text"`
	WalFilename string  `json:"walFilename" gorm:"column:wal_filename;type:text;not null"`

	StartLSN string `json:"startLsn" gorm:"column:start_lsn;type:pg_lsn;not null"`
	EndLSN   string `json:"endLsn"   gorm:"column:end_lsn;type:pg_lsn;not null"`

	CompressedSizeMb float64 `json:"compressedSizeMb" gorm:"column:compressed_size_mb;type:double precision;not null;default:0"`

	ReceivedAt time.Time `json:"receivedAt" gorm:"column:received_at"`
	ClaimedAt  time.Time `json:"claimedAt"  gorm:"column:claimed_at"`

	Encryption     string  `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`
	EncryptionSalt *string `json:"-"          gorm:"column:encryption_salt;type:text"`
	EncryptionIV   *string `json:"-"          gorm:"column:encryption_iv;type:text"`
}

func (a *App) InsertPhysicalWalSegment(tb testing.TB, seed PhysicalWalSegmentSeed) PhysicalWalSegmentSeed {
	tb.Helper()
	if seed.ID == uuid.Nil {
		seed.ID = uuid.New()
	}
	if seed.TimelineID == 0 {
		seed.TimelineID = 1
	}
	if seed.ReceivedAt.IsZero() {
		seed.ReceivedAt = time.Now().UTC()
	}
	if seed.ClaimedAt.IsZero() {
		seed.ClaimedAt = seed.ReceivedAt
	}
	a.insertPhysicalCatalogRow(tb, "backups_physical_wal_segments", &seed)
	return seed
}
