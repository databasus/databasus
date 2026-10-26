package backups_config_physical

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/util/logger"
)

var (
	getBackupConfigRepository = sync.OnceValue(func() *BackupConfigRepository {
		return &BackupConfigRepository{getDatabases: databases_bridge.GetModule}
	})
	getBackupConfigService = sync.OnceValue(func() *BackupConfigService {
		return &BackupConfigService{
			getBackupConfigRepository(),
			databases_bridge.GetAccess(),
			databases_bridge.GetModule,
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
