package module

import (
	"context"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/healthcheck/internal/attempts"
	"databasus-backend/internal/modules/healthcheck/internal/configs"
	healthcheckhttpapi "databasus-backend/internal/modules/healthcheck/internal/httpapi"
)

type (
	Deps struct {
		Handler  *healthcheckhttpapi.Handler
		Configs  *configs.Service
		Attempts *attempts.Service
	}
	Module struct {
		handler  *healthcheckhttpapi.Handler
		configs  *configs.Service
		attempts *attempts.Service
	}
)

func New(
	deps Deps,
) *Module {
	return &Module{configs: deps.Configs, attempts: deps.Attempts, handler: deps.Handler}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	healthcheckhttpapi.RegisterHandlersWithOptions(
		routes,
		healthcheckhttpapi.NewStrictHandlerWithOptions(
			m.handler,
			nil,
			healthcheckhttpapi.StrictGinServerOptions{
				RequestErrorHandlerFunc:  hooks.HandleRequestError,
				HandlerErrorFunc:         hooks.HandleHandlerError,
				ResponseErrorHandlerFunc: hooks.HandleResponseError,
			},
		),
		healthcheckhttpapi.GinServerOptions{
			BaseURL:      "",
			Middlewares:  []healthcheckhttpapi.MiddlewareFunc{healthcheckhttpapi.NewRequestValidator(hooks)},
			ErrorHandler: hooks.HandleParameterError,
		},
	)
}
func (m *Module) GetJobs() []background.Job { return m.attempts.GetJobs() }
func (m *Module) OnDatabaseCreated(ctx context.Context, event databases.DatabaseCreated) error {
	return m.configs.OnDatabaseCreated(ctx, event)
}

func (m *Module) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	return m.configs.OnDatabaseCopied(ctx, event)
}

func (m *Module) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	if err := m.attempts.OnDatabaseDeleting(ctx, event); err != nil {
		return err
	}

	return m.configs.OnDatabaseDeleting(ctx, event)
}
