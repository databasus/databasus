package httpapi

import (
	"context"

	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/healthcheck/internal/attempts"
	"databasus-backend/internal/modules/healthcheck/internal/configs"
	"databasus-backend/internal/modules/users"
)

var _ StrictServerInterface = (*Handler)(nil)

type (
	Deps struct {
		Configs  *configs.Service
		Attempts *attempts.Service
	}
	Handler struct {
		configs  *configs.Service
		attempts *attempts.Service
	}
)

func NewHandler(deps Deps) *Handler { return &Handler{configs: deps.Configs, attempts: deps.Attempts} }
func getActor(ctx context.Context) (users.Actor, error) {
	actor, isFound := users.GetActor(httpserver.GetRequestContext(ctx))
	if !isFound {
		return users.Actor{}, errNoActor
	}

	return actor, nil
}
