package signin

import (
	"time"

	"github.com/google/uuid"
)

// maxCodeAttempts is how many codes are checked against one pending sign-in before it is
// destroyed.
const maxCodeAttempts = 5

type PendingSignInKind string

const (
	PendingSignInKindEmailCode PendingSignInKind = "EMAIL_CODE"
	// PendingSignInKindAuthenticatorCode waits for a code of the app connected when it was created,
	// or a recovery code of that app.
	PendingSignInKindAuthenticatorCode PendingSignInKind = "AUTHENTICATOR_CODE"
	// PendingSignInKindAuthenticatorSetup waits for a user without an app to connect the one the
	// instance requires.
	PendingSignInKindAuthenticatorSetup PendingSignInKind = "AUTHENTICATOR_SETUP"
)

// PendingSignIn is a password sign-in that waits for the second factor. Exactly one of the hashed
// emailed code, the app and the setup is set, as the kind says; deleting the app or the setup
// deletes the pending sign-in with it. The row stays after it is spent or destroyed, because the
// hourly cap on emailed codes counts it.
type PendingSignIn struct {
	ID                   uuid.UUID         `gorm:"column:id"`
	UserID               uuid.UUID         `gorm:"column:user_id"`
	Kind                 PendingSignInKind `gorm:"column:kind"`
	HashedCode           *string           `gorm:"column:hashed_code"`
	AuthenticatorAppID   *uuid.UUID        `gorm:"column:authenticator_app_id"`
	AuthenticatorSetupID *uuid.UUID        `gorm:"column:authenticator_setup_id"`
	PasswordVersion      uuid.UUID         `gorm:"column:password_version"`
	ExpiresAt            time.Time         `gorm:"column:expires_at"`
	IsUsed               bool              `gorm:"column:is_used"`
	FailedAttemptCount   int               `gorm:"column:failed_attempt_count"`
	CreatedAt            time.Time         `gorm:"column:created_at"`
}

func (PendingSignIn) TableName() string {
	return "users_pending_sign_ins"
}

func (p PendingSignIn) IsUsable(now time.Time) bool {
	return !p.IsUsed && p.FailedAttemptCount < maxCodeAttempts && now.Before(p.ExpiresAt)
}
