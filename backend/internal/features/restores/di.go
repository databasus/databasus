package restores

import (
	"sync"

	"databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/disk"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/restores/usecases"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/util/logger"
)

var (
	getRestoreRepository = sync.OnceValue(func() *restores_core.RestoreRepository {
		return &restores_core.RestoreRepository{}
	})
	getRestoreService = sync.OnceValue(func() *RestoreService {
		return &RestoreService{
			backups_services.GetBackupService(),
			getRestoreRepository(),
			backups_bridge.GetModule,
			usecases.GetRestoreBackupUsecase(),
			databases_bridge.GetAccess(),
			databases_bridge.GetModule,
			logger.GetLogger(),
			workspaces_bridge.GetAccess(),
			audit_logs.GetAuditLogService(),
			config.GetEnv().ClientToolsDir,
			disk.GetDiskService(),
			tasks_cancellation.GetRequester(),
		}
	})
)

var getRestoreController = sync.OnceValue(func() *RestoreController {
	return &RestoreController{
		getRestoreService(),
	}
})

func GetRestoreController() *RestoreController {
	return getRestoreController()
}

var SetupDependencies = sync.OnceFunc(func() {
	backuping_logical.GetBackupCleaner().AddBackupRemoveListener(getRestoreService())
})
