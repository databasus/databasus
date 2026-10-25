// Package identity holds the types every package of the users module shares: the user as other
// modules see it, the global role, the user status and the actor of a request.
package identity

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
)

// Role is the global role of a user. An instance admin administers the instance; workspace roles
// are a separate concept of workspaces.
type Role string

const (
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

type Status string

const (
	// StatusInvited is a user who was invited and has not finished sign-up.
	StatusInvited  Status = "INVITED"
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// User is a user as other modules see it: never the password hash or the identity provider IDs.
type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	Role      Role
	Status    Status
	CreatedAt time.Time
}

func (u User) IsActive() bool {
	return u.Status == StatusActive
}

// GetAuditAuthor is the snapshot of the user an audit event keeps.
func (u User) GetAuditAuthor() *audit.Author {
	return &audit.Author{UserID: u.ID, Email: u.Email, Name: u.Name}
}
