package databases

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	"databasus-backend/internal/features/notifiers"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
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
		workspaces_bridge.GetAccess(),
		audit_logs.GetAuditLogService(),
		encryption.GetFieldEncryptor(),
		physical_core_service.GetPhysicalBackupService(),
	}
})

var getDatabaseController = sync.OnceValue(func() *DatabaseController {
	return &DatabaseController{
		getDatabaseService(),
		workspaces_bridge.GetAccess(),
	}
})

func GetDatabaseService() *DatabaseService {
	return getDatabaseService()
}

func GetDatabaseController() *DatabaseController {
	return getDatabaseController()
}

var SetupDependencies = sync.OnceFunc(func() {
	notifiers.GetNotifierService().SetNotifierDatabaseCounter(getDatabaseService())
})
