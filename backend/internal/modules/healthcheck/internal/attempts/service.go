package attempts

import (
	"context"
	"log/slog"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/healthcheck/internal/configs"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	Logger       *slog.Logger
	Configs      *configs.Service
	Databases    databases.Module
	Notifiers    notifiers.Module
}
type Service struct {
	repository *attemptRepository
	transactor *database.Transactor
	clock      clock.Source
	logger     *slog.Logger
	configs    *configs.Service
	databases  databases.Module
	notifiers  notifiers.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		repository: deps.Repositories.attempts,
		transactor: deps.Transactor,
		clock:      deps.Clock,
		logger:     deps.Logger,
		configs:    deps.Configs,
		databases:  deps.Databases,
		notifiers:  deps.Notifiers,
	}
}

func (s *Service) ListAttempts(ctx context.Context, actor users.Actor, input ListInput) (Page, error) {
	if _, err := s.configs.Authorize(
		ctx,
		configs.Authorization{Actor: actor, DatabaseID: input.DatabaseID, Action: workspaces.ActionView},
	); err != nil {
		return Page{}, err
	}

	var cursor *database.TimeCursor
	if input.After != nil {
		decoded, err := database.DecodeTimeCursor(*input.After)
		if err != nil {
			return Page{}, apierror.NewInvalidFieldsProblem(
				[]apierror.FieldError{{Field: "after", Detail: "invalid cursor"}},
			)
		}

		isFound, err := s.repository.hasCursor(ctx, input, decoded)
		if err != nil {
			return Page{}, err
		}

		if !isFound {
			return Page{}, apierror.NewInvalidFieldsProblem(
				[]apierror.FieldError{{Field: "after", Detail: "invalid cursor"}},
			)
		}

		cursor = &decoded
	}

	limit := input.Limit
	input.Limit++
	attempts, err := s.repository.list(ctx, input, cursor)
	if err != nil {
		return Page{}, err
	}

	page := Page{Items: attempts, NextCursor: nil}
	if len(attempts) > limit {
		page.Items = attempts[:limit]
		lastAttempt := page.Items[limit-1]
		encoded := (database.TimeCursor{Time: lastAttempt.CreatedAt, ID: lastAttempt.ID}).Encode()
		page.NextCursor = &encoded
	}

	return page, nil
}

func (s *Service) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return s.repository.delete(ctx, event.DatabaseID)
}
