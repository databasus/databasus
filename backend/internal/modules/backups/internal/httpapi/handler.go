package httpapi

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/transfer"
	"databasus-backend/internal/modules/users"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Configs   *config.Service
	Transfers *transfer.Service
}

type Handler struct {
	configs   *config.Service
	transfers *transfer.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{configs: deps.Configs, transfers: deps.Transfers}
}

func getActor(ctx context.Context) (users.Actor, error) {
	actor, isFound := users.GetActor(ctx)
	if !isFound {
		return users.Actor{}, errNoActor
	}

	return actor, nil
}
