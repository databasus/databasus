package healthcheck_attempt

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

type HealthcheckAttemptService struct {
	healthcheckAttemptRepository *HealthcheckAttemptRepository
	databaseService              *databases.DatabaseService
	workspaceService             *workspaces_services.WorkspaceService
}

func (s *HealthcheckAttemptService) GetAttemptsByDatabase(
	ctx context.Context,
	user users_models.User,
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

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, &user)
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

func (s *HealthcheckAttemptService) GetRecentAttemptsByDatabaseIDs(
	databaseIDs []uuid.UUID,
	limitPerDatabase int,
) (map[uuid.UUID][]*HealthcheckAttempt, error) {
	attempts, err := s.healthcheckAttemptRepository.FindRecentByDatabaseIDs(databaseIDs, limitPerDatabase)
	if err != nil {
		return nil, err
	}

	attemptsByDatabaseID := make(map[uuid.UUID][]*HealthcheckAttempt, len(databaseIDs))
	for _, attempt := range attempts {
		attemptsByDatabaseID[attempt.DatabaseID] = append(attemptsByDatabaseID[attempt.DatabaseID], attempt)
	}

	return attemptsByDatabaseID, nil
}
