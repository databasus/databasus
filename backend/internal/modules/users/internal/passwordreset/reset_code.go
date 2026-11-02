// Package passwordreset sends a single-use code to a user's address and replaces the password of a
// user who returns with it.
package passwordreset

import (
	"time"

	"github.com/google/uuid"
)

// maxCodeAttempts is how many values are checked against one code before it stops working.
const maxCodeAttempts = 5

// ResetCode is a code mailed to a user. Only its bcrypt hash is stored.
type ResetCode struct {
	ID                 uuid.UUID `gorm:"column:id"`
	UserID             uuid.UUID `gorm:"column:user_id"`
	HashedCode         string    `gorm:"column:hashed_code"`
	ExpiresAt          time.Time `gorm:"column:expires_at"`
	IsUsed             bool      `gorm:"column:is_used"`
	FailedAttemptCount int       `gorm:"column:failed_attempt_count"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (ResetCode) TableName() string {
	return "users_password_reset_codes"
}
