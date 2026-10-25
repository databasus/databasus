package workspaces_models

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/workspaces"
)

type WorkspaceMembership struct {
	ID          uuid.UUID       `json:"id"          gorm:"column:id"`
	UserID      uuid.UUID       `json:"userId"      gorm:"column:user_id"`
	WorkspaceID uuid.UUID       `json:"workspaceId" gorm:"column:workspace_id"`
	Role        workspaces.Role `json:"role"        gorm:"column:role"`
	CreatedAt   time.Time       `json:"createdAt"   gorm:"column:created_at"`
}

func (WorkspaceMembership) TableName() string {
	return "workspace_memberships"
}
