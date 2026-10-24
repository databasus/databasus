package healthcheck_config

import (
	"sync"

	"databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/databases"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var (
	getHealthcheckConfigRepository = sync.OnceValue(func() *HealthcheckConfigRepository {
		return &HealthcheckConfigRepository{}
	})
	getHealthcheckConfigService = sync.OnceValue(func() *HealthcheckConfigService {
		return &HealthcheckConfigService{
			databases.GetDatabaseService(),
			getHealthcheckConfigRepository(),
			workspaces_services.GetWorkspaceService(),
			audit_logs.GetAuditLogService(),
			logger.GetLogger(),
		}
	})
)

var getHealthcheckConfigController = sync.OnceValue(func() *HealthcheckConfigController {
	return &HealthcheckConfigController{
		getHealthcheckConfigService(),
	}
})

func GetHealthcheckConfigService() *HealthcheckConfigService {
	return getHealthcheckConfigService()
}

func GetHealthcheckConfigController() *HealthcheckConfigController {
	return getHealthcheckConfigController()
}

var SetupDependencies = sync.OnceFunc(func() {
	databases.
		GetDatabaseService().
		AddDbCreationListener(getHealthcheckConfigService())
})
