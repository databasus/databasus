package databases

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	"databasus-backend/internal/features/notifiers"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getDatabaseRepository = sync.OnceValue(func() *DatabaseRepository {
	return &DatabaseRepository{}
})

var getDatabaseService = sync.OnceValue(func() *DatabaseService {
	return &DatabaseService{
		getDatabaseRepository(),
		notifiers.GetNotifierService(),
		logger.GetLogger(),
		[]DatabaseCreationListener{},
		[]DatabaseRemoveListener{},
		[]DatabaseCopyListener{},
		[]DatabaseBackupTypeChangeListener{},
		workspaces_services.GetWorkspaceService(),
		audit_logs.GetAuditLogService(),
		encryption.GetFieldEncryptor(),
		physical_core_service.GetPhysicalBackupService(),
	}
})

var getDatabaseController = sync.OnceValue(func() *DatabaseController {
	return &DatabaseController{
		getDatabaseService(),
		users_services.GetUserService(),
		workspaces_services.GetWorkspaceService(),
	}
})

func GetDatabaseService() *DatabaseService {
	return getDatabaseService()
}

func GetDatabaseController() *DatabaseController {
	return getDatabaseController()
}

var SetupDependencies = sync.OnceFunc(func() {
	workspaces_services.GetWorkspaceService().AddWorkspaceDeletionListener(getDatabaseService())
	notifiers.GetNotifierService().SetNotifierDatabaseCounter(getDatabaseService())
})
