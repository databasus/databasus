// Package members adds, invites, lists, changes and removes the members of a workspace and
// transfers its ownership.
package members

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/workspaces/internal/access"
)

// Member is one user in one workspace.
type Member struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        access.Role
	CreatedAt   time.Time
}

// ListedMember is a member with the user it names.
type ListedMember struct {
	Member
	Email string
	Name  string
	// IsInvited is true while the user has not finished sign-up.
	IsInvited bool
}

type AdditionOutcome string

const (
	AdditionOutcomeAdded   AdditionOutcome = "ADDED"
	AdditionOutcomeInvited AdditionOutcome = "INVITED"
)

type Addition struct {
	WorkspaceID uuid.UUID
	Email       string
	Role        access.Role
}

type AddedMember struct {
	ListedMember
	Outcome AdditionOutcome
}
