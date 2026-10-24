package notifiers

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var (
	getNotifierRepository = sync.OnceValue(func() *NotifierRepository {
		return &NotifierRepository{}
	})
	getNotifierService = sync.OnceValue(func() *NotifierService {
		return &NotifierService{
			getNotifierRepository(),
			logger.GetLogger(),
			workspaces_services.GetWorkspaceService(),
			audit_logs.GetAuditLogService(),
			encryption.GetFieldEncryptor(),
			nil,
		}
	})
)

var getNotifierController = sync.OnceValue(func() *NotifierController {
	return &NotifierController{
		getNotifierService(),
		workspaces_services.GetWorkspaceService(),
	}
})

func GetNotifierController() *NotifierController {
	return getNotifierController()
}

func GetNotifierService() *NotifierService {
	return getNotifierService()
}

func GetNotifierRepository() *NotifierRepository {
	return getNotifierRepository()
}

var SetupDependencies = sync.OnceFunc(func() {
	workspaces_services.GetWorkspaceService().AddWorkspaceDeletionListener(getNotifierService())
})
