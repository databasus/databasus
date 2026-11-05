package signin

import (
	"time"

	"github.com/google/uuid"
)

// maxCodeAttempts is how many codes are checked against one pending sign-in before it is
// destroyed.
const maxCodeAttempts = 5

// PendingSignIn is a password sign-in that waits for the code emailed to the user. The row stays
// after it is spent or destroyed, because the hourly cap on codes counts it.
type PendingSignIn struct {
	ID                 uuid.UUID `gorm:"column:id"`
	UserID             uuid.UUID `gorm:"column:user_id"`
	HashedCode         string    `gorm:"column:hashed_code"`
	PasswordVersion    uuid.UUID `gorm:"column:password_version"`
	ExpiresAt          time.Time `gorm:"column:expires_at"`
	IsUsed             bool      `gorm:"column:is_used"`
	FailedAttemptCount int       `gorm:"column:failed_attempt_count"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (PendingSignIn) TableName() string {
	return "users_pending_sign_ins"
}

func (p PendingSignIn) IsUsable(now time.Time) bool {
	return !p.IsUsed && p.FailedAttemptCount < maxCodeAttempts && now.Before(p.ExpiresAt)
}
