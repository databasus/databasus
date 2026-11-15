// Package identityproof confirms that root admin is present before a sensitive operation: their
// current password or a single-use code emailed to their account. A proof is bound to the account,
// the session, a purpose and the operation's subject, so a proof for one operation cannot authorize
// another.
package identityproof

import (
	"time"

	"github.com/google/uuid"
)

// Purpose names the operation a proof authorizes.
type Purpose string

const (
	PurposeMasterKeyExport  Purpose = "MASTER_KEY_EXPORT"
	PurposeInstanceRecovery Purpose = "INSTANCE_RECOVERY"
)

// Method is a way root admin can confirm identity.
type Method string

const (
	MethodPassword  Method = "PASSWORD"
	MethodEmailCode Method = "EMAIL_CODE"
)

const (
	codeLifetime    = 5 * time.Minute
	maxCodeAttempts = 5
)

// Binding is what a proof authorizes: a purpose and the subject of that operation, such as a
// recovery ID with its archive fingerprint. A master-key export has an empty subject.
type Binding struct {
	Purpose Purpose
	Subject string
}

// Proof carries exactly one of the current password and an emailed code.
type Proof struct {
	Password *string
	Code     *string
}

// IdentityCode is an emailed code. Its recipient address is kept, so a code sent before the
// account's address changed stops working.
type IdentityCode struct {
	ID                 uuid.UUID `gorm:"column:id"`
	UserID             uuid.UUID `gorm:"column:user_id"`
	SessionSha256      string    `gorm:"column:session_sha256"`
	Purpose            Purpose   `gorm:"column:purpose"`
	Subject            string    `gorm:"column:subject"`
	RecipientEmail     string    `gorm:"column:recipient_email"`
	HashedCode         string    `gorm:"column:hashed_code"`
	ExpiresAt          time.Time `gorm:"column:expires_at"`
	IsUsed             bool      `gorm:"column:is_used"`
	FailedAttemptCount int       `gorm:"column:failed_attempt_count"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (IdentityCode) TableName() string {
	return "users_identity_codes"
}

func (c IdentityCode) IsUsable(now time.Time) bool {
	return !c.IsUsed && c.FailedAttemptCount < maxCodeAttempts && now.Before(c.ExpiresAt)
}
