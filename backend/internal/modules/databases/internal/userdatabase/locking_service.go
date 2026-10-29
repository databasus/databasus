package userdatabase

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
)

func (s *Service) GetDatabaseForUpdate(ctx context.Context, databaseID uuid.UUID) (Database, bool, error) {
	if err := database.RequireTransaction(ctx); err != nil {
		return Database{}, false, err
	}

	userDatabase, isFound, err := s.repository.get(ctx, databaseID, forUpdateRowLock)
	if err != nil || !isFound {
		return Database{}, false, err
	}

	attachedDatabases, err := s.loadNotifierIDs(ctx, []Database{userDatabase})
	if err != nil {
		return Database{}, false, err
	}

	return attachedDatabases[0], true, nil
}
