package catalog

import (
	"time"

	"github.com/google/uuid"
)

type BackupMetadata struct {
	BackupID         uuid.UUID  `json:"backupId"`
	DatabaseID       uuid.UUID  `json:"databaseId"`
	BackupType       BackupType `json:"backupType"`
	SystemIdentifier uint64     `json:"systemIdentifier"`
	PgVersion        int        `json:"pgVersion"`
	TimelineID       int        `json:"timelineId"`

	StartLSN string `json:"startLsn"`
	StopLSN  string `json:"stopLsn"`

	UncompressedSizeBytes int64 `json:"uncompressedSizeBytes"`
	CompressedSizeBytes   int64 `json:"compressedSizeBytes"`

	RootFullBackupID          *uuid.UUID `json:"rootFullBackupId,omitempty"`
	ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId,omitempty"`

	Encryption     BackupEncryption `json:"encryption"`
	EncryptionSalt string           `json:"encryptionSalt,omitempty"`
	EncryptionIV   string           `json:"encryptionIv,omitempty"`

	Compression BackupCompression `json:"compression"`

	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt"`
}
