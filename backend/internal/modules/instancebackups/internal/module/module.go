// Package module implements the public contract of the instancebackups module.
package module

import (
	"context"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/instancebackups/internal/catalog"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	instancebackupshttpapi "databasus-backend/internal/modules/instancebackups/internal/httpapi"
	"databasus-backend/internal/modules/storages"
)

type (
	Deps struct {
		Handler *instancebackupshttpapi.Handler
		Configs *configs.Service
		Catalog *catalog.Service
		Jobs    []background.Job
	}
	Module struct {
		handler *instancebackupshttpapi.Handler
		configs *configs.Service
		catalog *catalog.Service
		jobs    []background.Job
	}
)

func New(deps Deps) *Module {
	return &Module{handler: deps.Handler, configs: deps.Configs, catalog: deps.Catalog, jobs: deps.Jobs}
}

// OnStorageDeleting refuses deleting a current destination first; a storage that is only in history
// may go, and its copies stay listed as unavailable.
func (m *Module) OnStorageDeleting(ctx context.Context, event storages.StorageDeleting) error {
	if err := m.configs.OnStorageDeleting(ctx, event.StorageIDs); err != nil {
		return err
	}

	return m.catalog.OnStorageDeleting(ctx, event.StorageIDs)
}

func (m *Module) Start(ctx context.Context) error {
	return m.catalog.Start(ctx)
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	instancebackupshttpapi.RegisterHandlersWithOptions(
		routes,
		instancebackupshttpapi.NewStrictHandlerWithOptions(
			m.handler,
			nil,
			instancebackupshttpapi.StrictGinServerOptions{
				RequestErrorHandlerFunc:  hooks.HandleRequestError,
				HandlerErrorFunc:         hooks.HandleHandlerError,
				ResponseErrorHandlerFunc: hooks.HandleResponseError,
			},
		),
		instancebackupshttpapi.GinServerOptions{
			BaseURL:      "",
			Middlewares:  []instancebackupshttpapi.MiddlewareFunc{instancebackupshttpapi.NewRequestValidator(hooks)},
			ErrorHandler: hooks.HandleParameterError,
		},
	)
}

func (m *Module) GetJobs() []background.Job { return m.jobs }
