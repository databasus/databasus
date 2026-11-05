package apptest

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type PhysicalIncrementalBackupSeed struct {
	ID         uuid.UUID `json:"id"         gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	StorageID  uuid.UUID `json:"storageId"  gorm:"column:storage_id;type:uuid;not null"`

	RootFullBackupID          uuid.UUID  `json:"rootFullBackupId"          gorm:"column:root_full_backup_id;type:uuid;not null"`
	ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId" gorm:"-"`

	TimelineID int `json:"timelineId" gorm:"column:timeline_id;type:int;not null"`

	Status      string  `json:"status"      gorm:"column:status;type:text;not null"`
	ErrorReason *string `json:"errorReason" gorm:"column:error_reason;type:text"`
	FailMessage *string `json:"failMessage" gorm:"column:fail_message;type:text"`

	FileName *string `json:"fileName" gorm:"column:file_name;type:text"`

	StartLSN *string `json:"startLsn" gorm:"column:start_lsn;type:pg_lsn"`
	StopLSN  *string `json:"stopLsn"  gorm:"column:stop_lsn;type:pg_lsn"`

	BackupSizeMb     *float64 `json:"backupSizeMb"     gorm:"column:backup_size_mb;type:double precision"`
	RawSizeMb        *float64 `json:"rawSizeMb"        gorm:"column:raw_size_mb;type:double precision"`
	BackupDurationMs *int64   `json:"backupDurationMs" gorm:"column:backup_duration_ms;type:bigint"`

	Encryption     string  `json:"encryption" gorm:"column:encryption;type:text;not null;default:'NONE'"`
	EncryptionSalt *string `json:"-"          gorm:"column:encryption_salt;type:text"`
	EncryptionIV   *string `json:"-"          gorm:"column:encryption_iv;type:text"`

	Compression string `json:"compression" gorm:"column:compression;type:text;not null;default:'ZSTD'"`

	ManifestFileName       *string `json:"manifestFileName" gorm:"column:manifest_file_name;type:text"`
	ManifestEncryptionSalt *string `json:"-"                gorm:"column:manifest_encryption_salt;type:text"`
	ManifestEncryptionIV   *string `json:"-"                gorm:"column:manifest_encryption_iv;type:text"`

	CreatedAt   time.Time  `json:"createdAt"   gorm:"column:created_at"`
	CompletedAt *time.Time `json:"completedAt" gorm:"column:completed_at"`
}

func (a *App) InsertPhysicalIncrementalBackup(
	tb testing.TB,
	seed PhysicalIncrementalBackupSeed,
) PhysicalIncrementalBackupSeed {
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
	if seed.Status == "" {
		seed.Status = "COMPLETED"
	}
	if seed.Status == "COMPLETED" && seed.FileName == nil {
		seed.FileName = new("seed-" + seed.ID.String())
	}
	if seed.Status == "COMPLETED" && seed.CompletedAt == nil {
		seed.CompletedAt = new(seed.CreatedAt)
	}

	parentID := seed.ParentIncrementalBackupID
	if parentID == nil {
		parentID = &seed.RootFullBackupID
	}

	row := struct {
		PhysicalIncrementalBackupSeed `gorm:"embedded"`
		Kind                          string     `gorm:"column:kind"`
		ParentBackupID                *uuid.UUID `gorm:"column:parent_backup_id"`
	}{PhysicalIncrementalBackupSeed: seed, Kind: "INCREMENTAL", ParentBackupID: parentID}

	a.insertPhysicalCatalogRow(tb, "backups_physical", &row)
	seed = row.PhysicalIncrementalBackupSeed
	return seed
}
