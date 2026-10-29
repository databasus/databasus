package app

import (
	"github.com/gin-gonic/gin"

	contract "databasus-backend/api"
	systemhttpapi "databasus-backend/internal/app/systemhttpapi"
	"databasus-backend/internal/features/disk"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	system_agent "databasus-backend/internal/features/system/agent"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	system_version "databasus-backend/internal/features/system/version"
	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/httpserver"
)

// registerContractRoutes mounts the servers generated from the contract on a group that answers 500
// to any route the contract does not describe. The old controllers stay on the plain group below.
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

	systemHandler := systemhttpapi.NewHandler(systemhttpapi.Deps{Version: system_version.GetAppVersion()})
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

func (a *App) registerRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")

	// Public routes: only the user sign-in routes, the health check and the agent-facing routes.
	system_healthcheck.GetHealthcheckController().RegisterRoutes(v1)
	agentController := system_agent.AgentController{BinariesDir: a.environment.AgentBinariesDir}
	agentController.RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(a.usersModule.GetSessionMiddleware())

	disk.GetDiskController().RegisterRoutes(protected)
	healthcheck_config.GetHealthcheckConfigController().RegisterRoutes(protected)
	healthcheck_attempt.GetHealthcheckAttemptController().RegisterRoutes(protected)
}
