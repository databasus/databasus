package connections

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
)

// A session authenticates with the settings it was dialed with, so the key carries a fingerprint of
// those settings: once they change, the next checkout lands under a new key and never reuses a
// session opened with the old ones.
type Key struct {
	StorageID   uuid.UUID
	Fingerprint string
}

func Fingerprint(parts ...any) string {
	encodedParts, err := json.Marshal(parts)
	if err != nil {
		panic("storage fingerprint parts must be JSON-encodable: " + err.Error())
	}

	digest := sha256.Sum256(encodedParts)

	return hex.EncodeToString(digest[:])
}
