package audit_logs

import (
	"sync"

	users_bridge "databasus-backend/internal/features/users"
	"databasus-backend/internal/util/logger"
)

var getAuditLogService = sync.OnceValue(func() *AuditLogService {
	return &AuditLogService{
		getUsersModule: users_bridge.GetModule,
		logger:         logger.GetLogger(),
	}
})

func GetAuditLogService() *AuditLogService {
	return getAuditLogService()
}
