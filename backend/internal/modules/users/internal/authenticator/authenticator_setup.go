package authenticator

import (
	"time"

	"github.com/google/uuid"
)

const (
	// setupLifetime gives a user time to install an app before the setup stops being confirmable.
	setupLifetime = 30 * time.Minute

	maxSetupAttempts = 5

	addressIssuer = "Databasus"
)

// AuthenticatorAppSetup is an app that is not connected until a code from it is confirmed. A user has at most one;
// starting another replaces it.
type AuthenticatorAppSetup struct {
	ID                 uuid.UUID `gorm:"column:id"`
	UserID             uuid.UUID `gorm:"column:user_id"`
	EncryptedSecret    string    `gorm:"column:encrypted_secret"`
	FailedAttemptCount int       `gorm:"column:failed_attempt_count"`
	ExpiresAt          time.Time `gorm:"column:expires_at"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (AuthenticatorAppSetup) TableName() string {
	return "users_authenticator_setups"
}

func (s AuthenticatorAppSetup) IsConfirmable(now time.Time) bool {
	return s.FailedAttemptCount < maxSetupAttempts && now.Before(s.ExpiresAt)
}

// StartedAuthenticatorAppSetup is the answer of the operation that started a setup, the only answer that carries
// the secret.
type StartedAuthenticatorAppSetup struct {
	SetupID   uuid.UUID
	Secret    string
	Address   string
	ExpiresAt time.Time
}

// AuthenticatorAppSetupConfirmation tells a wrong code apart from a connected app without an error, so the caller's
// transaction still records the wrong attempt.
type AuthenticatorAppSetupConfirmation struct {
	IsConnected   bool
	RecoveryCodes []string
}
