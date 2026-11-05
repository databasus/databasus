package catalog

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type BackupKind string

const (
	BackupKindFull        BackupKind = "FULL"
	BackupKindIncremental BackupKind = "INCREMENTAL"
)

type Backup struct {
	StorageGeneration int64      `gorm:"column:storage_generation"`
	Kind              BackupKind `gorm:"column:kind;type:text;not null"`
	ParentBackupID    *uuid.UUID `gorm:"column:parent_backup_id;type:uuid"`
	RootFullBackupID  *uuid.UUID `gorm:"column:root_full_backup_id;type:uuid"`
	ID                uuid.UUID  `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID        uuid.UUID  `gorm:"column:database_id;type:uuid;not null"`
	StorageID         uuid.UUID  `gorm:"column:storage_id;type:uuid;not null"`

	TimelineID int `gorm:"column:timeline_id;type:int;not null"`

	Status      BackupStatus       `gorm:"column:status;type:text;not null"`
	ErrorReason *BackupErrorReason `gorm:"column:error_reason;type:text"`
	FailMessage *string            `gorm:"column:fail_message;type:text"`

	FileName *string `gorm:"column:file_name;type:text"`

	StartLSN *walmath.LSN `gorm:"column:start_lsn;type:pg_lsn"`
	StopLSN  *walmath.LSN `gorm:"column:stop_lsn;type:pg_lsn"`

	BackupSizeMb     *float64 `gorm:"column:backup_size_mb;type:double precision"`
	RawSizeMb        *float64 `gorm:"column:raw_size_mb;type:double precision"`
	BackupDurationMs *int64   `gorm:"column:backup_duration_ms;type:bigint"`

	Encryption     BackupEncryption `gorm:"column:encryption;type:text;not null;default:'NONE'"`
	EncryptionSalt *string          `gorm:"column:encryption_salt;type:text"`
	EncryptionIV   *string          `gorm:"column:encryption_iv;type:text"`

	Compression BackupCompression `gorm:"column:compression;type:text;not null;default:'ZSTD'"`

	ManifestFileName       *string `gorm:"column:manifest_file_name;type:text"`
	ManifestEncryptionSalt *string `gorm:"column:manifest_encryption_salt;type:text"`
	ManifestEncryptionIV   *string `gorm:"column:manifest_encryption_iv;type:text"`

	CreatedAt   time.Time  `gorm:"column:created_at"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
}

func (Backup) TableName() string {
	return "backups_physical"
}

func (b *Backup) GetManifestFileName() string {
	if b.ManifestFileName != nil && *b.ManifestFileName != "" {
		return *b.ManifestFileName
	}
	if b.FileName != nil && *b.FileName != "" {
		return *b.FileName + ManifestSuffix
	}
	return ""
}
