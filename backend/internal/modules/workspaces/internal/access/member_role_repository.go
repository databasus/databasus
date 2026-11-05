package access

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// authorizer never holds it.
type Repositories struct {
	memberRoles *memberRoleRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{memberRoles: &memberRoleRepository{db: db}}
}

type memberRoleRepository struct {
	db *gorm.DB
}

func (r *memberRoleRepository) getRole(ctx context.Context, workspaceID, userID uuid.UUID) (Role, bool, error) {
	var roles []Role

	err := database.Conn(ctx, r.db).
		Raw("SELECT role FROM workspaces_members WHERE workspace_id = ? AND user_id = ?", workspaceID, userID).
		Scan(&roles).Error
	if err != nil {
		return "", false, fmt.Errorf("get the workspace role of the user: %w", err)
	}

	if len(roles) == 0 {
		return "", false, nil
	}

	return roles[0], true, nil
}
