package audit_logs

import (
	"sync"

	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/util/logger"
)

var (
	getAuditLogRepository = sync.OnceValue(func() *AuditLogRepository {
		return &AuditLogRepository{}
	})
	getAuditLogService = sync.OnceValue(func() *AuditLogService {
		return &AuditLogService{
			getAuditLogRepository(),
			logger.GetLogger(),
			logger.GetAuditLogger(),
		}
	})
)

var getAuditLogController = sync.OnceValue(func() *AuditLogController {
	return &AuditLogController{
		getAuditLogService(),
	}
})

var getAuditLogBackgroundService = sync.OnceValue(func() *AuditLogBackgroundService {
	return &AuditLogBackgroundService{
		auditLogService: getAuditLogService(),
		logger:          logger.GetLogger(),
	}
})

func GetAuditLogService() *AuditLogService {
	return getAuditLogService()
}

func GetAuditLogController() *AuditLogController {
	return getAuditLogController()
}

func GetAuditLogBackgroundService() *AuditLogBackgroundService {
	return getAuditLogBackgroundService()
}

var SetupDependencies = sync.OnceFunc(func() {
	users_services.GetUserService().SetAuditLogWriter(getAuditLogService())
	users_services.GetSettingsService().SetAuditLogWriter(getAuditLogService())
	users_services.GetManagementService().SetAuditLogWriter(getAuditLogService())
})
