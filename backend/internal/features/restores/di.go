package restores

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/disk"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/restores/usecases"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var (
	restoreRepository = &restores_core.RestoreRepository{}
	restoreService    = &RestoreService{
		backups_services.GetBackupService(),
		restoreRepository,
		storages.GetStorageService(),
		backups_config_logical.GetBackupConfigService(),
		usecases.GetRestoreBackupUsecase(),
		databases.GetDatabaseService(),
		logger.GetLogger(),
		workspaces_services.GetWorkspaceService(),
		audit_logs.GetAuditLogService(),
		encryption.GetFieldEncryptor(),
		disk.GetDiskService(),
		tasks_cancellation.GetRequester(),
	}
)

var restoreController = &RestoreController{
	restoreService,
}

func GetRestoreController() *RestoreController {
	return restoreController
}

var SetupDependencies = sync.OnceFunc(func() {
	backuping_logical.GetBackupCleaner().AddBackupRemoveListener(restoreService)
})
