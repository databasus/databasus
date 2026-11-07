package httpapi

import (
	"databasus-backend/internal/modules/verification/internal/agents"
	"databasus-backend/internal/modules/verification/internal/configs"
	"databasus-backend/internal/modules/verification/internal/verifications"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Agents        *agents.Service
	Configs       *configs.Service
	Verifications *verifications.Service
}
type Handler struct {
	agents        *agents.Service
	configs       *configs.Service
	verifications *verifications.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{agents: deps.Agents, configs: deps.Configs, verifications: deps.Verifications}
}
