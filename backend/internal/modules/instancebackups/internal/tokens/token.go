// Package tokens issues the single-use credentials of downloads that a browser link carries: a
// master-key download and a Databasus backup download. Only the SHA-256 of a token is stored.
package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// Purpose tells the downloads apart, so a token of one kind never authorizes the other.
type Purpose string

const (
	PurposeMasterKeyDownload      Purpose = "MASTER_KEY_DOWNLOAD"
	PurposeInstanceBackupDownload Purpose = "INSTANCE_BACKUP_DOWNLOAD"
)

const tokenSizeBytes = 32

// Token is a stored download credential.
type Token struct {
	TokenSha256 string     `gorm:"column:token_sha256"`
	Purpose     Purpose    `gorm:"column:purpose"`
	UserID      uuid.UUID  `gorm:"column:user_id"`
	CopyID      *uuid.UUID `gorm:"column:copy_id"`
	ExpiresAt   time.Time  `gorm:"column:expires_at"`
	ConsumedAt  *time.Time `gorm:"column:consumed_at"`
}

func (Token) TableName() string {
	return "instancebackups_tokens"
}

// IssuedToken is returned once, to the operation that issued it.
type IssuedToken struct {
	Token     string
	ExpiresAt time.Time
}

// generateTokenText panics when the system's random source fails, because no weaker token may
// stand in for it.
func generateTokenText() string {
	randomBytes := make([]byte, tokenSizeBytes)

	if _, err := rand.Read(randomBytes); err != nil {
		panic("read the random bytes of a download token: " + err.Error())
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes)
}

func hashToken(tokenText string) string {
	digest := sha256.Sum256([]byte(tokenText))

	return hex.EncodeToString(digest[:])
}
