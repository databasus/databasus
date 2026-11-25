// Package authenticator owns the authenticator apps of the users: setting one up, checking its codes
// and recovery codes once each under a per-account limit on wrong codes, replacing, disconnecting
// and resetting it.
package authenticator

import (
	"time"

	"github.com/google/uuid"
)

// AuthenticatorApp is a connected authenticator app. Its secret is encrypted with the master key, because codes
// are computed from it and it cannot be hashed. LastUsedStep is the 30-second period of the latest
// accepted code, so no code of that period or an earlier one is accepted again.
type AuthenticatorApp struct {
	ID              uuid.UUID `gorm:"column:id"`
	UserID          uuid.UUID `gorm:"column:user_id"`
	EncryptedSecret string    `gorm:"column:encrypted_secret"`
	LastUsedStep    int64     `gorm:"column:last_used_step"`
	ConnectedAt     time.Time `gorm:"column:connected_at"`
}

func (AuthenticatorApp) TableName() string {
	return "users_authenticator_apps"
}

type AuthenticatorAppState struct {
	ConnectedAt             time.Time
	UnusedRecoveryCodeCount int
}
