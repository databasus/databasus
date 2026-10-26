package healthcheck_config

import (
	"sync"

	"databasus-backend/internal/features/audit_logs"
	databases_bridge "databasus-backend/internal/features/databases"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/util/logger"
)

var (
	getHealthcheckConfigRepository = sync.OnceValue(func() *HealthcheckConfigRepository {
		return &HealthcheckConfigRepository{}
	})
	getHealthcheckConfigService = sync.OnceValue(func() *HealthcheckConfigService {
		return &HealthcheckConfigService{
			databases_bridge.GetAccess(),
			databases_bridge.GetModule,
			getHealthcheckConfigRepository(),
			workspaces_bridge.GetAccess(),
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
