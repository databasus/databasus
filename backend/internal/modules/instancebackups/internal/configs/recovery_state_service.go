package configs

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RecoveryTargetSettings is what the Databasus backup settings of a startup recovery target hold.
type RecoveryTargetSettings struct {
	// IsPresent is false on a database that has no settings row yet.
	IsPresent bool
	// IsDefault is true when the row still holds the values a migration creates it with.
	IsDefault bool
}

// RecoveryStateService reads and writes the settings row of a database other than the running
// application's: the target a startup recovery would replace and the staging database it pauses.
type RecoveryStateService struct {
	repositories Repositories
}

func NewRecoveryStateService(repositories Repositories) *RecoveryStateService {
	return &RecoveryStateService{repositories: repositories}
}

// InspectTarget reads a database that may be fresh or of an older schema. A row this release
// cannot read counts as not default, so an unknown state is never taken for an automatic one.
func (s *RecoveryStateService) InspectTarget(ctx context.Context) (RecoveryTargetSettings, error) {
	rowCount, err := s.repositories.configs.countRows(ctx)
	if err != nil {
		return RecoveryTargetSettings{}, err
	}

	if rowCount == 0 {
		return RecoveryTargetSettings{IsPresent: false, IsDefault: false}, nil
	}

	config, err := s.repositories.configs.get(ctx)
	if err != nil {
		return RecoveryTargetSettings{}, err
	}

	return RecoveryTargetSettings{IsPresent: true, IsDefault: rowCount == 1 && config.IsDefault()}, nil
}

// Pause marks a recovered database paused by its recovery operation.
func (s *RecoveryStateService) Pause(ctx context.Context, operationID uuid.UUID, pausedAt time.Time) error {
	return s.repositories.configs.updateRecoveryPause(ctx, &operationID, &pausedAt)
}

// GetPause returns the operation that paused the database, or nil for an active one.
func (s *RecoveryStateService) GetPause(ctx context.Context) (*uuid.UUID, error) {
	config, err := s.repositories.configs.get(ctx)
	if err != nil {
		return nil, err
	}

	return config.RecoveryPausedOperationID, nil
}
