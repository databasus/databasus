package physical_models

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

type PhysicalIncrementalBackup struct {
	ID         uuid.UUID `json:"id"         gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;type:uuid;not null"`

	// The FULL this chain is built on. Always set.
	RootFullBackupID uuid.UUID `json:"rootFullBackupId" gorm:"column:root_full_backup_id;type:uuid;not null"`
	// The previous INCR in the chain or NULL if the previous step is the FULL.
	ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId" gorm:"column:parent_incremental_backup_id;type:uuid"`

	TimelineID int `json:"timelineId" gorm:"column:timeline_id;type:int;not null"`

	Status      physical_enums.PhysicalBackupStatus       `json:"status"      gorm:"column:status;type:text;not null"`
	ErrorReason *physical_enums.PhysicalBackupErrorReason `json:"errorReason" gorm:"column:error_reason;type:text"`
	FailMessage *string                                   `json:"failMessage" gorm:"column:fail_message;type:text"`

	FileName *string `json:"fileName" gorm:"column:file_name;type:text"`

	StartLSN *walmath.LSN `json:"startLsn" gorm:"column:start_lsn;type:pg_lsn"`
	StopLSN  *walmath.LSN `json:"stopLsn"  gorm:"column:stop_lsn;type:pg_lsn"`

	BackupSizeMb     *float64 `json:"backupSizeMb"     gorm:"column:backup_size_mb;type:double precision"`
	RawSizeMb        *float64 `json:"rawSizeMb"        gorm:"column:raw_size_mb;type:double precision"`
	BackupDurationMs *int64   `json:"backupDurationMs" gorm:"column:backup_duration_ms;type:bigint"`

	Encryption     backups_core_enums.BackupEncryption `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`
	EncryptionSalt *string                             `json:"-"          gorm:"column:encryption_salt;type:text"`
	EncryptionIV   *string                             `json:"-"          gorm:"column:encryption_iv;type:text"`

	Compression physical_enums.PhysicalBackupCompression `json:"compression" gorm:"column:compression;type:text;not null;default:'ZSTD'"`

	ManifestFileName       *string `json:"manifestFileName" gorm:"column:manifest_file_name;type:text"`
	ManifestEncryptionSalt *string `json:"-"                gorm:"column:manifest_encryption_salt;type:text"`
	ManifestEncryptionIV   *string `json:"-"                gorm:"column:manifest_encryption_iv;type:text"`

	CreatedAt   time.Time  `json:"createdAt"   gorm:"column:created_at"`
	CompletedAt *time.Time `json:"completedAt" gorm:"column:completed_at"`
}

func (PhysicalIncrementalBackup) TableName() string {
	return "physical_incremental_backups"
}

func (b *PhysicalIncrementalBackup) ParentBackupID() uuid.UUID {
	if b.ParentIncrementalBackupID != nil {
		return *b.ParentIncrementalBackupID
	}

	return b.RootFullBackupID
}
