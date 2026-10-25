package audit_logs

import (
	"sync"

	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/util/logger"
)

var (
	getAuditLogRepository = sync.OnceValue(func() *AuditLogRepository {
		return &AuditLogRepository{}
	})
	getAuditLogService = sync.OnceValue(func() *AuditLogService {
		return &AuditLogService{
			auditLogRepository: getAuditLogRepository(),
			userRepository:     users_repositories.GetUserRepository(),
			logger:             logger.GetLogger(),
		}
	})
)

var getAuditLogController = sync.OnceValue(func() *AuditLogController {
	return &AuditLogController{
		getAuditLogService(),
	}
})

func GetAuditLogService() *AuditLogService {
	return getAuditLogService()
}

func GetAuditLogController() *AuditLogController {
	return getAuditLogController()
}

var SetupDependencies = sync.OnceFunc(func() {
	users_services.GetUserService().SetAuditLogWriter(getAuditLogService())
	users_services.GetSettingsService().SetAuditLogWriter(getAuditLogService())
	users_services.GetManagementService().SetAuditLogWriter(getAuditLogService())
})
