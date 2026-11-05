package catalog

import (
	"time"

	"github.com/google/uuid"
)

type WalSegmentMetadata struct {
	WalSegmentID uuid.UUID `json:"walSegmentId"`
	DatabaseID   uuid.UUID `json:"databaseId"`
	TimelineID   int       `json:"timelineId"`
	WalFilename  string    `json:"walFilename"`

	StartLSN string `json:"startLsn"`
	EndLSN   string `json:"endLsn"`

	CompressedSizeBytes int64 `json:"compressedSizeBytes"`

	Encryption     BackupEncryption `json:"encryption"`
	EncryptionSalt string           `json:"encryptionSalt,omitempty"`
	EncryptionIV   string           `json:"encryptionIv,omitempty"`

	ReceivedAt time.Time `json:"receivedAt"`
}
