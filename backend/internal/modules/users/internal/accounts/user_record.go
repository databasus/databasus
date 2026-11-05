// Package accounts owns the users table: the profile, the password, invitations, user
// administration and the console commands, and the rule that the first user administers the
// instance.
package accounts

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users/internal/identity"
)

// UserRecord is a row of the users table with everything the module keeps about a user. Only the
// module sees it; other modules get an identity.User.
type UserRecord struct {
	ID                   uuid.UUID       `gorm:"column:id"`
	Email                string          `gorm:"column:email"`
	Name                 string          `gorm:"column:name"`
	HashedPassword       *string         `gorm:"column:hashed_password"`
	PasswordVersion      uuid.UUID       `gorm:"column:password_version"`
	PasswordCreationTime time.Time       `gorm:"column:password_creation_time"`
	Role                 identity.Role   `gorm:"column:role"`
	Status               identity.Status `gorm:"column:status"`
	GitHubOAuthID        *string         `gorm:"column:github_oauth_id"`
	GoogleOAuthID        *string         `gorm:"column:google_oauth_id"`
	// IsBootstrapAdmin marks the user that claimed the empty instance. The column keeps its old
	// name, so the mark survives a change of the address.
	IsBootstrapAdmin bool      `gorm:"column:is_root_admin"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (UserRecord) TableName() string {
	return "users"
}

func (r UserRecord) IsActive() bool {
	return r.Status == identity.StatusActive
}

func (r UserRecord) HasPassword() bool {
	return r.HashedPassword != nil && *r.HashedPassword != ""
}

func (r UserRecord) ToUser() identity.User {
	return identity.User{
		ID:        r.ID,
		Email:     r.Email,
		Name:      r.Name,
		Role:      r.Role,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
	}
}

func (r UserRecord) ToActor() identity.Actor {
	return identity.Actor{UserID: r.ID, Role: r.Role}
}
