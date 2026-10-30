package app

import (
	"github.com/gin-gonic/gin"

	contract "databasus-backend/api"
	systemhttpapi "databasus-backend/internal/app/systemhttpapi"
	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/httpserver"
)

// registerContractRoutes mounts the servers generated from the contract on a group that answers 500
// to any route the contract does not describe.
func (a *App) registerContractRoutes(router *gin.Engine) error {
	contractGuard, err := httpserver.RequireContractOperation(contract.OpenAPI, a.logger)
	if err != nil {
		return err
	}

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

	agentGuard, err := httpserver.RequireContractOperation(contract.Agent, a.logger)
	if err != nil {
		return err
	}
	a.verificationModule.RegisterAgentRoutes(router.Group("/api/v1", agentGuard), errorHooks)
	return nil
}
