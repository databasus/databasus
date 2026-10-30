package module

import (
	"context"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/verification/internal/agentapi"
	"databasus-backend/internal/modules/verification/internal/agents"
	"databasus-backend/internal/modules/verification/internal/configs"
	verificationhttpapi "databasus-backend/internal/modules/verification/internal/httpapi"
	"databasus-backend/internal/modules/verification/internal/verifications"
)

type Deps struct {
	Agents        *agents.Service
	Configs       *configs.Service
	Verifications *verifications.Service
	Handler       *verificationhttpapi.Handler
	AgentHandler  *agentapi.Handler
}
type Module struct {
	agents        *agents.Service
	configs       *configs.Service
	verifications *verifications.Service
	handler       *verificationhttpapi.Handler
	agentHandler  *agentapi.Handler
}

func New(deps Deps) *Module {
	return &Module{
		agents:        deps.Agents,
		configs:       deps.Configs,
		verifications: deps.Verifications,
		handler:       deps.Handler,
		agentHandler:  deps.AgentHandler,
	}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	verificationhttpapi.RegisterHandlersWithOptions(
		routes,
		verificationhttpapi.NewStrictHandlerWithOptions(
			m.handler,
			nil,
			verificationhttpapi.StrictGinServerOptions{
				RequestErrorHandlerFunc:  hooks.HandleRequestError,
				HandlerErrorFunc:         hooks.HandleHandlerError,
				ResponseErrorHandlerFunc: hooks.HandleResponseError,
			},
		),
		verificationhttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: hooks.HandleParameterError},
	)
}
func (m *Module) GetJobs() []background.Job { return m.verifications.GetJobs() }
func (m *Module) OnLogicalBackupCompleted(ctx context.Context, event backups.LogicalBackupCompleted) error {
	return m.verifications.OnLogicalBackupCompleted(ctx, event)
}

func (m *Module) OnLogicalBackupsDeleting(ctx context.Context, event backups.LogicalBackupsDeleting) error {
	return m.verifications.OnLogicalBackupsDeleting(ctx, event)
}

func (m *Module) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	return m.configs.OnDatabaseCopied(ctx, event)
}

func (m *Module) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return m.configs.OnDatabaseDeleting(ctx, event)
}

func (m *Module) ListStaleAgents(ctx context.Context) ([]agents.Agent, error) {
	return m.agents.ListStaleAgents(ctx)
}

func (m *Module) ListAgents(ctx context.Context) ([]agents.Agent, error) {
	return m.agents.ListAgents(ctx)
}

func (m *Module) ListEnabledConfigs(ctx context.Context) ([]configs.Config, error) {
	return m.configs.ListEnabledConfigs(ctx)
}

func (m *Module) RegisterAgentRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	group := routes.Group(
		"",
		m.agentHandler.GetAuthenticationMiddleware(hooks),
		agentapi.GetHeartbeatValidationMiddleware(hooks),
	)
	agentapi.RegisterHandlersWithOptions(
		group,
		agentapi.NewStrictHandlerWithOptions(m.agentHandler, nil, agentapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  hooks.HandleRequestError,
			HandlerErrorFunc:         hooks.HandleHandlerError,
			ResponseErrorHandlerFunc: hooks.HandleResponseError,
		}),
		agentapi.GinServerOptions{ErrorHandler: hooks.HandleParameterError},
	)
}
