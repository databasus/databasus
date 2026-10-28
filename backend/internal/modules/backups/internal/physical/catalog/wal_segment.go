package catalog

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// file_name = NULL means the row is claimed but bytes are still uploading.
type WalSegment struct {
	ID         uuid.UUID `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `gorm:"column:storage_id;type:uuid;not null"`

	TimelineID  int     `gorm:"column:timeline_id;type:int;not null"`
	FileName    *string `gorm:"column:file_name;type:text"`
	WalFilename string  `gorm:"column:wal_filename;type:text;not null"`

	StartLSN walmath.LSN `gorm:"column:start_lsn;type:pg_lsn;not null"`
	EndLSN   walmath.LSN `gorm:"column:end_lsn;type:pg_lsn;not null"`

	CompressedSizeMb float64 `gorm:"column:compressed_size_mb;type:double precision;not null;default:0"`

	ReceivedAt time.Time `gorm:"column:received_at"`
	ClaimedAt  time.Time `gorm:"column:claimed_at"`

	Encryption     BackupEncryption `gorm:"column:encryption;type:text;not null;default:'NONE'"`
	EncryptionSalt *string          `gorm:"column:encryption_salt;type:text"`
	EncryptionIV   *string          `gorm:"column:encryption_iv;type:text"`
}

func (WalSegment) TableName() string {
	return "backups_physical_wal_segments"
}
