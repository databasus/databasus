package download

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

// tokenLifetime is how long a download token stays usable after it was issued.
const tokenLifetime = 5 * time.Minute

// tokenSizeBytes makes a token of 256 random bits.
const tokenSizeBytes = 32

// DownloadToken lets one request without a session download one backup file once.
type DownloadToken struct {
	ID        uuid.UUID
	Token     string
	BackupID  uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	IsUsed    bool
	CreatedAt time.Time
}

// IssuedToken is what the user gets for a download: the token, the name of the downloaded file and
// when the token expires.
type IssuedToken struct {
	Token     string
	FileName  string
	ExpiresAt time.Time
}

// generateTokenText panics when the system's random source fails, because no weaker token may stand
// in for it.
func generateTokenText() string {
	randomBytes := make([]byte, tokenSizeBytes)

	_, err := rand.Read(randomBytes)
	if err != nil {
		panic("read the random bytes of a download token: " + err.Error())
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes)
}
