package configs

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Authorization struct {
	Actor      users.Actor
	DatabaseID uuid.UUID
	Action     workspaces.Action
}

func (s *Service) Authorize(
	ctx context.Context,
	input Authorization,
) (databases.Database, error) {
	listedDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{input.DatabaseID})
	if err != nil {
		return databases.Database{}, err
	}

	if len(listedDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}

	userDatabase := listedDatabases[0]
	if err := s.workspaces.Authorize(ctx, input.Actor, userDatabase.WorkspaceID, input.Action); err != nil {
		var problem *apierror.Problem
		if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
			return databases.Database{}, ErrDatabaseNotFound
		}

		return databases.Database{}, err
	}

	return userDatabase, nil
}
