package verification_config

import (
	"sync"

	"databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/databases"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var getVerificationConfigRepository = sync.OnceValue(func() *VerificationConfigRepository {
	return &VerificationConfigRepository{}
})

var getVerificationConfigService = sync.OnceValue(func() *VerificationConfigService {
	return &VerificationConfigService{
		getVerificationConfigRepository(),
		databases.GetDatabaseService(),
		workspaces_services.GetWorkspaceService(),
		audit_logs.GetAuditLogService(),
		logger.GetLogger(),
	}
})

var getVerificationConfigController = sync.OnceValue(func() *VerificationConfigController {
	return &VerificationConfigController{
		getVerificationConfigService(),
	}
})

func GetVerificationConfigService() *VerificationConfigService {
	return getVerificationConfigService()
}

func GetVerificationConfigController() *VerificationConfigController {
	return getVerificationConfigController()
}

var SetupDependencies = sync.OnceFunc(func() {
	databases.GetDatabaseService().AddDbCopyListener(getVerificationConfigService())
})
