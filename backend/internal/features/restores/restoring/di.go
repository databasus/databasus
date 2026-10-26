package restoring

import (
	"sync"
	"sync/atomic"
	"time"

	"databasus-backend/internal/config"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/caching"
	databases_bridge "databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/restores/usecases"
	storages_bridge "databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/util/logger"
)

var getRestoreRepository = sync.OnceValue(func() *restores_core.RestoreRepository {
	return &restores_core.RestoreRepository{}
})

var getRestoreDatabaseCache = sync.OnceValue(func() *cache.JSONStore[RestoreDatabaseCache] {
	return cache.NewJSONStore[RestoreDatabaseCache](caching.GetStore(), "restore_db")
})

var getTaskCancellationRegistry = sync.OnceValue(tasks_cancellation.GetRegistry)

var getRestorer = sync.OnceValue(func() *Restorer {
	return &Restorer{
		databases_bridge.GetAccess(),
		databases_bridge.GetModule,
		backups_services.GetBackupService(),
		getRestoreRepository(),
		backups_config_logical.GetBackupConfigService(),
		storages_bridge.GetModule,
		logger.GetLogger(),
		usecases.GetRestoreBackupUsecase(),
		getRestoreDatabaseCache(),
		getTaskCancellationRegistry(),
		config.GetShutdownSignal(),
		config.GetEnv().ClientToolsDir,
	}
})

var getRestoresScheduler = sync.OnceValue(func() *RestoresScheduler {
	return &RestoresScheduler{
		getRestoreRepository(),
		time.Now().UTC(),
		logger.GetLogger(),
		getRestorer(),
		getRestoreDatabaseCache(),
		atomic.Bool{},
	}
})

func GetRestoresScheduler() *RestoresScheduler {
	return getRestoresScheduler()
}

func GetRestorer() *Restorer {
	return getRestorer()
}
