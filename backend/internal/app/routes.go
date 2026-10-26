package app

import (
	"github.com/gin-gonic/gin"

	contract "databasus-backend/api"
	systemhttpapi "databasus-backend/internal/app/systemhttpapi"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/disk"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/restores"
	system_agent "databasus-backend/internal/features/system/agent"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	system_version "databasus-backend/internal/features/system/version"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
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

	return nil
}

func (a *App) registerRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")

	// Public routes: only the user sign-in routes, the health check and the agent-facing routes.
	system_healthcheck.GetHealthcheckController().RegisterRoutes(v1)
	system_agent.GetAgentController().RegisterRoutes(v1)
	backups_controllers_logical.GetBackupController().RegisterPublicRoutes(v1)
	backups_controllers_physical.GetPhysicalBackupController().RegisterPublicRoutes(v1)
	databases.GetDatabaseController().RegisterPublicRoutes(v1)
	verification_agents.GetAgentFacingController().RegisterRoutes(v1)
	verification_runs.GetVerificationAgentController().RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(a.usersModule.GetSessionMiddleware())

	disk.GetDiskController().RegisterRoutes(protected)
	databases.GetDatabaseController().RegisterRoutes(protected)
	backups_controllers_logical.GetBackupController().RegisterRoutes(protected)
	backups_controllers_physical.GetPhysicalBackupController().RegisterRoutes(protected)
	restores.GetRestoreController().RegisterRoutes(protected)
	healthcheck_config.GetHealthcheckConfigController().RegisterRoutes(protected)
	healthcheck_attempt.GetHealthcheckAttemptController().RegisterRoutes(protected)
	backups_config_logical.GetBackupConfigController().RegisterRoutes(protected)
	backups_config_physical.GetBackupConfigController().RegisterRoutes(protected)
	verification_agents.GetAgentController().RegisterRoutes(protected)
	verification_config.GetVerificationConfigController().RegisterRoutes(protected)
	verification_runs.GetVerificationController().RegisterRoutes(protected)
}
