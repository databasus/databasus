package backups_config_logical

import (
	"sync"

	"databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
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
			notifiers_bridge.GetModule,
			workspaces_bridge.GetAccess(),
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
