package healthcheck_attempt

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
)

type HealthcheckAttemptService struct {
	healthcheckAttemptRepository *HealthcheckAttemptRepository
	databaseService              *databases.DatabaseService
	workspaceService             *workspaces_services.WorkspaceService
}

func (s *HealthcheckAttemptService) GetAttemptsByDatabase(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	afterDate time.Time,
) ([]*HealthcheckAttempt, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot access healthcheck attempts for databases without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("forbidden")
	}

	return s.healthcheckAttemptRepository.FindByDatabaseIdOrderByCreatedAtDesc(
		databaseID,
		afterDate,
	)
}
