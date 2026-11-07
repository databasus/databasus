package httpapi

import (
	"databasus-backend/internal/modules/healthcheck/internal/attempts"
	"databasus-backend/internal/modules/healthcheck/internal/configs"
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
