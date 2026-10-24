package backuping_logical

import (
	"sync"
	"sync/atomic"
	"time"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	usecases_logical "databasus-backend/internal/features/backups/backups/usecases/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var getBackupRepository = sync.OnceValue(func() *backups_core_logical.BackupRepository {
	return &backups_core_logical.BackupRepository{}
})

var (
	getTaskCancellationRegistry  = sync.OnceValue(tasks_cancellation.GetRegistry)
	getTaskCancellationRequester = sync.OnceValue(tasks_cancellation.GetRequester)
)

var getBackupCleaner = sync.OnceValue(func() *BackupCleaner {
	return &BackupCleaner{
		getBackupRepository(),
		storages.GetStorageFileStore(),
		backups_config_logical.GetBackupConfigService(),
		logger.GetLogger(),
		[]backups_core_logical.BackupRemoveListener{},
		atomic.Bool{},
	}
})

var getBackuper = sync.OnceValue(func() *Backuper {
	return &Backuper{
		databases.GetDatabaseService(),
		workspaces_services.GetWorkspaceService(),
		getBackupRepository(),
		backups_config_logical.GetBackupConfigService(),
		storages.GetStorageFileStore(),
		notifiers.GetNotifierService(),
		getTaskCancellationRegistry(),
		logger.GetLogger(),
		usecases_logical.GetCreateBackupUsecase(),
	}
})

var getBackupsScheduler = sync.OnceValue(func() *BackupsScheduler {
	return &BackupsScheduler{
		getBackupRepository(),
		backups_config_logical.GetBackupConfigService(),
		getTaskCancellationRequester(),
		databases.GetDatabaseService(),
		time.Now().UTC(),
		logger.GetLogger(),
		getBackuper(),
		[]backups_core_logical.BackupCompletionListener{},
		atomic.Bool{},
		atomic.Bool{},
	}
})

func GetBackupsScheduler() *BackupsScheduler {
	return getBackupsScheduler()
}

func GetBackuper() *Backuper {
	return getBackuper()
}

func GetBackupCleaner() *BackupCleaner {
	return getBackupCleaner()
}
