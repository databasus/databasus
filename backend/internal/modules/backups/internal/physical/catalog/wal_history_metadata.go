package catalog

import (
	"time"

	"github.com/google/uuid"
)

type WalHistoryMetadata struct {
	WalHistoryFileID uuid.UUID `json:"walHistoryFileId"`
	DatabaseID       uuid.UUID `json:"databaseId"`
	TimelineID       int       `json:"timelineId"`
	HistoryFilename  string    `json:"historyFilename"`

	CompressedSizeBytes int64 `json:"compressedSizeBytes"`

	Encryption     BackupEncryption `json:"encryption"`
	EncryptionSalt string           `json:"encryptionSalt,omitempty"`
	EncryptionIV   string           `json:"encryptionIv,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}
