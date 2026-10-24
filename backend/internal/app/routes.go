package app

import (
	"github.com/gin-gonic/gin"

	contract "databasus-backend/api"
	systemhttpapi "databasus-backend/internal/app/systemhttpapi"
	"databasus-backend/internal/features/audit_logs"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/disk"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/restores"
	"databasus-backend/internal/features/storages"
	system_agent "databasus-backend/internal/features/system/agent"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	system_version "databasus-backend/internal/features/system/version"
	users_controllers "databasus-backend/internal/features/users/controllers"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
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

	contractRoutes := router.Group("/api/v1", contractGuard)
	errorHooks := apierror.NewHooks(apierror.Deps{Logger: a.logger})

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

func registerRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")

	// Public routes: only the user sign-in routes, the health check and the agent-facing routes.
	userController := users_controllers.GetUserController()
	userController.RegisterRoutes(v1)
	system_healthcheck.GetHealthcheckController().RegisterRoutes(v1)
	system_agent.GetAgentController().RegisterRoutes(v1)
	backups_controllers_logical.GetBackupController().RegisterPublicRoutes(v1)
	backups_controllers_physical.GetPhysicalBackupController().RegisterPublicRoutes(v1)
	databases.GetDatabaseController().RegisterPublicRoutes(v1)
	verification_agents.GetAgentFacingController().RegisterRoutes(v1)
	verification_runs.GetVerificationAgentController().RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	userController.RegisterProtectedRoutes(protected)
	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected)
	workspaces_controllers.GetMembershipController().RegisterRoutes(protected)
	disk.GetDiskController().RegisterRoutes(protected)
	notifiers.GetNotifierController().RegisterRoutes(protected)
	storages.GetStorageController().RegisterRoutes(protected)
	databases.GetDatabaseController().RegisterRoutes(protected)
	backups_controllers_logical.GetBackupController().RegisterRoutes(protected)
	backups_controllers_physical.GetPhysicalBackupController().RegisterRoutes(protected)
	restores.GetRestoreController().RegisterRoutes(protected)
	healthcheck_config.GetHealthcheckConfigController().RegisterRoutes(protected)
	healthcheck_attempt.GetHealthcheckAttemptController().RegisterRoutes(protected)
	backups_config_logical.GetBackupConfigController().RegisterRoutes(protected)
	backups_config_physical.GetBackupConfigController().RegisterRoutes(protected)
	audit_logs.GetAuditLogController().RegisterRoutes(protected)
	users_controllers.GetManagementController().RegisterRoutes(protected)
	users_controllers.GetSettingsController().RegisterRoutes(protected)
	verification_agents.GetAgentController().RegisterRoutes(protected)
	verification_config.GetVerificationConfigController().RegisterRoutes(protected)
	verification_runs.GetVerificationController().RegisterRoutes(protected)
}
