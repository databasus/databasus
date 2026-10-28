package catalog

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type FullBackup struct {
	ID         uuid.UUID `json:"id"         gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;type:uuid;not null"`

	TimelineID int `json:"timelineId" gorm:"column:timeline_id;type:int;not null"`

	Status      BackupStatus       `json:"status"      gorm:"column:status;type:text;not null"`
	ErrorReason *BackupErrorReason `json:"errorReason" gorm:"column:error_reason;type:text"`
	FailMessage *string            `json:"failMessage" gorm:"column:fail_message;type:text"`

	FileName *string `json:"fileName" gorm:"column:file_name;type:text"`

	StartLSN *walmath.LSN `json:"startLsn" gorm:"column:start_lsn;type:pg_lsn"`
	StopLSN  *walmath.LSN `json:"stopLsn"  gorm:"column:stop_lsn;type:pg_lsn"`

	BackupSizeMb     *float64 `json:"backupSizeMb"     gorm:"column:backup_size_mb;type:double precision"`
	RawSizeMb        *float64 `json:"rawSizeMb"        gorm:"column:raw_size_mb;type:double precision"`
	BackupDurationMs *int64   `json:"backupDurationMs" gorm:"column:backup_duration_ms;type:bigint"`

	Encryption     BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`
	EncryptionSalt *string          `json:"-"          gorm:"column:encryption_salt;type:text"`
	EncryptionIV   *string          `json:"-"          gorm:"column:encryption_iv;type:text"`

	Compression BackupCompression `json:"compression" gorm:"column:compression;type:text;not null;default:'ZSTD'"`

	ManifestFileName       *string `json:"manifestFileName" gorm:"column:manifest_file_name;type:text"`
	ManifestEncryptionSalt *string `json:"-"                gorm:"column:manifest_encryption_salt;type:text"`
	ManifestEncryptionIV   *string `json:"-"                gorm:"column:manifest_encryption_iv;type:text"`

	CreatedAt   time.Time  `json:"createdAt"   gorm:"column:created_at"`
	CompletedAt *time.Time `json:"completedAt" gorm:"column:completed_at"`
}

func (FullBackup) TableName() string {
	return "backups_physical_full"
}
