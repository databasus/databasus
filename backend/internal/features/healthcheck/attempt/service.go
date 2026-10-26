package healthcheck_attempt

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	databases_bridge "databasus-backend/internal/features/databases"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type HealthcheckAttemptService struct {
	healthcheckAttemptRepository *HealthcheckAttemptRepository
	databaseAccess               *databases_bridge.Access
	workspaceService             *workspaces_bridge.Access
}

func (s *HealthcheckAttemptService) GetAttemptsByDatabase(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	afterDate time.Time,
) ([]*HealthcheckAttempt, error) {
	database, err := s.databaseAccess.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("forbidden")
	}

	return s.healthcheckAttemptRepository.FindByDatabaseIdOrderByCreatedAtDesc(
		databaseID,
		afterDate,
	)
}
