package backups_config_physical

import (
	"sync"

	"databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/util/logger"
)

var (
	getBackupConfigRepository = sync.OnceValue(func() *BackupConfigRepository {
		return &BackupConfigRepository{}
	})
	getBackupConfigService = sync.OnceValue(func() *BackupConfigService {
		return &BackupConfigService{
			getBackupConfigRepository(),
			databases.GetDatabaseService(),
			storages_bridge.GetModule,
			notifiers_bridge.GetModule,
			workspaces_bridge.GetAccess(),
			logger.GetLogger(),
			nil,
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
	databases.GetDatabaseService().AddDbBackupTypeChangeListener(getBackupConfigService())
})
