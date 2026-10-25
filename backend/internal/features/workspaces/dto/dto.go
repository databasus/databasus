package workspaces_dto

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/workspaces"
)

type AddMemberStatus string

const (
	AddStatusInvited AddMemberStatus = "INVITED"
	AddStatusAdded   AddMemberStatus = "ADDED"
)

// Workspace DTOs
type CreateWorkspaceRequestDTO struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

type WorkspaceResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`

	// User's role in this workspace (populated when fetching for specific user)
	UserRole *workspaces.Role `json:"userRole,omitempty"`
}

type ListWorkspacesResponseDTO struct {
	Workspaces []WorkspaceResponseDTO `json:"workspaces"`
}

// Membership DTOs
type AddMemberRequestDTO struct {
	Email string          `json:"email" binding:"required,email"`
	Role  workspaces.Role `json:"role"  binding:"required"`
}

type AddMemberResponseDTO struct {
	Status AddMemberStatus `json:"status"`
}

type ChangeMemberRoleRequestDTO struct {
	Role workspaces.Role `json:"role" binding:"required"`
}

type TransferOwnershipRequestDTO struct {
	NewOwnerEmail string `json:"newOwnerEmail" binding:"required"`
}

type WorkspaceMemberResponseDTO struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"userId"`
	Email     string          `json:"email"` // Populated from user join
	Name      string          `json:"name"`  // Populated from user join
	Role      workspaces.Role `json:"role"`
	CreatedAt time.Time       `json:"createdAt"`
}

type GetMembersResponseDTO struct {
	Members []WorkspaceMemberResponseDTO `json:"members"`
}
