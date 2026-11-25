package authenticator

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	recoveryCodeCount = 10
	// recoveryCodeLength base32 characters carry 50 random bits, too many to guess online and too
	// many to brute-force from the SHA-256 hash offline in useful time.
	recoveryCodeLength    = 10
	recoveryCodeGroupSize = 5
)

// RecoveryCode is one single-use replacement for an app code. Only its SHA-256 hash is stored.
type RecoveryCode struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	AuthenticatorAppID uuid.UUID  `gorm:"column:authenticator_app_id"`
	HashedCode         string     `gorm:"column:hashed_code"`
	UsedAt             *time.Time `gorm:"column:used_at"`
}

func (RecoveryCode) TableName() string {
	return "users_recovery_codes"
}

func generateRecoveryCodes(appID uuid.UUID) ([]string, []RecoveryCode, error) {
	shownCodes := make([]string, 0, recoveryCodeCount)
	storedCodes := make([]RecoveryCode, 0, recoveryCodeCount)

	for range recoveryCodeCount {
		randomBytes := make([]byte, 7)

		_, err := rand.Read(randomBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("generate a recovery code: %w", err)
		}

		code := strings.ToLower(base32.StdEncoding.EncodeToString(randomBytes))[:recoveryCodeLength]

		shownCodes = append(shownCodes, code[:recoveryCodeGroupSize]+"-"+code[recoveryCodeGroupSize:])
		storedCodes = append(storedCodes, RecoveryCode{
			ID:                 uuid.New(),
			AuthenticatorAppID: appID,
			HashedCode:         hashRecoveryCode(code),
			UsedAt:             nil,
		})
	}

	return shownCodes, storedCodes, nil
}

func normalizeRecoveryCode(input string) string {
	replacer := strings.NewReplacer("-", "", " ", "")

	return strings.ToLower(replacer.Replace(strings.TrimSpace(input)))
}

func hashRecoveryCode(normalizedCode string) string {
	digest := sha256.Sum256([]byte(normalizedCode))

	return hex.EncodeToString(digest[:])
}
