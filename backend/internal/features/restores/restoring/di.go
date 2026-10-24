package restoring

import (
	"sync/atomic"
	"time"

	"databasus-backend/internal/config"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/restores/usecases"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var restoreRepository = &restores_core.RestoreRepository{}

var restoreDatabaseCache = cache.NewJSONStore[RestoreDatabaseCache](cache.GetStore(), "restore_db")

var taskCancellationRegistry = tasks_cancellation.GetRegistry()

var restorer = &Restorer{
	databases.GetDatabaseService(),
	backups_services.GetBackupService(),
	encryption.GetFieldEncryptor(),
	restoreRepository,
	backups_config_logical.GetBackupConfigService(),
	storages.GetStorageService(),
	logger.GetLogger(),
	usecases.GetRestoreBackupUsecase(),
	restoreDatabaseCache,
	taskCancellationRegistry,
	config.GetShutdownSignal(),
}

var restoresScheduler = &RestoresScheduler{
	restoreRepository,
	time.Now().UTC(),
	logger.GetLogger(),
	restorer,
	restoreDatabaseCache,
	atomic.Bool{},
}

func GetRestoresScheduler() *RestoresScheduler {
	return restoresScheduler
}

func GetRestorer() *Restorer {
	return restorer
}
