package audit_logs

import (
	"sync"

	users_bridge "databasus-backend/internal/features/users"
	"databasus-backend/internal/util/logger"
)

var (
	getAuditLogRepository = sync.OnceValue(func() *AuditLogRepository {
		return &AuditLogRepository{}
	})
	getAuditLogService = sync.OnceValue(func() *AuditLogService {
		return &AuditLogService{
			auditLogRepository: getAuditLogRepository(),
			getUsersModule:     users_bridge.GetModule,
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
