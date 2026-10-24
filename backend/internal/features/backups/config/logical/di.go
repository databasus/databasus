package backups_config_logical

import (
	"sync"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

var (
	getBackupConfigRepository = sync.OnceValue(func() *BackupConfigRepository {
		return &BackupConfigRepository{}
	})
	getBackupConfigService = sync.OnceValue(func() *BackupConfigService {
		return &BackupConfigService{
			getBackupConfigRepository(),
			databases.GetDatabaseService(),
			storages.GetStorageService(),
			notifiers.GetNotifierService(),
			workspaces_services.GetWorkspaceService(),
			nil,
		}
	})
)

var getBackupConfigController = sync.OnceValue(func() *BackupConfigController {
	return &BackupConfigController{
		getBackupConfigService(),
	}
})

func GetBackupConfigController() *BackupConfigController {
	return getBackupConfigController()
}

func GetBackupConfigService() *BackupConfigService {
	return getBackupConfigService()
}

var SetupDependencies = sync.OnceFunc(func() {
	storages.GetStorageService().AddStorageDatabaseCounter(getBackupConfigService())
})
