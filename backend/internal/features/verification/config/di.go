package verification_config

import (
	"sync"

	"databasus-backend/internal/features/audit_logs"
	databases_bridge "databasus-backend/internal/features/databases"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/util/logger"
)

var getVerificationConfigRepository = sync.OnceValue(func() *VerificationConfigRepository {
	return &VerificationConfigRepository{}
})

var getVerificationConfigService = sync.OnceValue(func() *VerificationConfigService {
	return &VerificationConfigService{
		getVerificationConfigRepository(),
		databases_bridge.GetAccess(),
		workspaces_bridge.GetAccess(),
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
