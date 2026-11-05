package app

import (
	"sync"

	"github.com/getkin/kin-openapi/routers"
	"github.com/gin-gonic/gin"

	contract "databasus-backend/api"
	systemhttpapi "databasus-backend/internal/app/systemhttpapi"
	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/httpserver"
)

//nolint:gochecknoglobals // The immutable contract router is parsed once per process and shared by applications; it holds no application state.
var loadUserContractRouter = sync.OnceValues(func() (routers.Router, error) {
	return httpserver.NewAccessCheckedContractRouter(contract.OpenAPI)
})

//nolint:gochecknoglobals // The immutable contract router is parsed once per process and shared by applications; it holds no application state.
var loadAgentContractRouter = sync.OnceValues(func() (routers.Router, error) {
	return httpserver.NewAccessCheckedContractRouter(contract.Agent)
})

// registerContractRoutes mounts the servers generated from the contract on a group that answers 500
// to any route the contract does not describe.
func (a *App) registerContractRoutes(router *gin.Engine) error {
	userContractRouter, err := loadUserContractRouter()
	if err != nil {
		return err
	}

	agentContractRouter, err := loadAgentContractRouter()
	if err != nil {
		return err
	}

	contractGuard := httpserver.RequireContractOperation(userContractRouter, a.logger)
	contractRoutes := router.Group("/api/v1", contractGuard, a.usersModule.GetAccessMiddleware())
	errorHooks := apierror.NewHooks(apierror.Deps{Logger: a.logger})

	a.usersModule.RegisterRoutes(contractRoutes, errorHooks)
	a.workspacesModule.RegisterRoutes(contractRoutes, errorHooks)
	a.notifiersModule.RegisterRoutes(contractRoutes, errorHooks)
	a.storagesModule.RegisterRoutes(contractRoutes, errorHooks)
	a.databasesModule.RegisterRoutes(contractRoutes, errorHooks)
	a.backupsModule.RegisterRoutes(contractRoutes, errorHooks)
	a.verificationModule.RegisterRoutes(contractRoutes, errorHooks)
	a.healthcheckModule.RegisterRoutes(contractRoutes, errorHooks)

	systemHandler := systemhttpapi.NewHandler(systemhttpapi.Deps{
		Version:         a.environment.Config.AppVersion,
		Clock:           a.environment.Clock,
		Logger:          a.logger,
		DB:              a.db,
		Verification:    a.verificationModule,
		ClientTools:     a.clientToolsChecker,
		DataDirectory:   a.environment.Config.DataDirectory,
		BinaryDirectory: a.environment.AgentBinariesDir,
		InspectDisk:     a.inspectDisk,
		GetJobs:         a.getJobHealth,
	})
	systemhttpapi.RegisterHandlersWithOptions(
		contractRoutes,
		systemhttpapi.NewStrictHandlerWithOptions(systemHandler, nil, systemhttpapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  errorHooks.HandleRequestError,
			HandlerErrorFunc:         errorHooks.HandleHandlerError,
			ResponseErrorHandlerFunc: errorHooks.HandleResponseError,
		}),
		systemhttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: errorHooks.HandleParameterError},
	)

	agentGuard := httpserver.RequireContractOperation(agentContractRouter, a.logger)
	a.verificationModule.RegisterAgentRoutes(router.Group("/api/v1", agentGuard), errorHooks)
	return nil
}
