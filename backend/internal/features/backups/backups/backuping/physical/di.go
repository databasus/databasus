package backuping_physical

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getPhysicalBackuper = sync.OnceValue(func() *PhysicalBackuper {
	return &PhysicalBackuper{
		databases.GetDatabaseService(),
		encryption.GetFieldEncryptor(),
		workspaces_services.GetWorkspaceService(),
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		physical_repositories.GetWalHistoryRepository(),
		backups_config_physical.GetBackupConfigService(),
		storages.GetStorageService(),
		storages.GetStorageFileStore(),
		notifiers.GetNotifierService(),
		tasks_cancellation.GetRegistry(),
		encryption_secrets.GetSecretKeyService(),
		logger.GetLogger(),
		postgresql_executor.NewCreateFullBackupUsecase(),
		postgresql_executor.NewCreateIncrementalBackupUsecase(),
	}
})

func GetPhysicalBackuper() *PhysicalBackuper { return getPhysicalBackuper() }

var getPhysicalBackupsScheduler = sync.OnceValue(func() *PhysicalBackupsScheduler {
	return &PhysicalBackupsScheduler{
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		backups_config_physical.GetBackupConfigService(),
		chain_view.GetChainViewService(),
		tasks_cancellation.GetRequester(),
		storages.GetStorageFileStore(),
		getPhysicalBackuper(),
		atomicTime{},
		logger.GetLogger(),
		atomic.Bool{},
		atomic.Bool{},
	}
})

func GetPhysicalBackupsScheduler() *PhysicalBackupsScheduler { return getPhysicalBackupsScheduler() }

var getPhysicalBackupCleaner = sync.OnceValue(func() *PhysicalBackupCleaner {
	return &PhysicalBackupCleaner{
		physical_service.GetPhysicalBackupService(),
		chain_view.GetChainViewService(),
		backups_config_physical.GetBackupConfigService(),
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalStreamSampleRepository(),
		logger.GetLogger(),
		atomic.Bool{},
	}
})

func GetPhysicalBackupCleaner() *PhysicalBackupCleaner { return getPhysicalBackupCleaner() }

var getPhysicalWalStreamSupervisor = sync.OnceValue(func() *PhysicalWalStreamSupervisor {
	return &PhysicalWalStreamSupervisor{
		databases.GetDatabaseService(),
		backups_config_physical.GetBackupConfigService(),
		storages.GetStorageService(),
		storages.GetStorageFileStore(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalHistoryRepository(),
		physical_repositories.GetWalStreamerRepository(),
		physical_repositories.GetWalStreamSampleRepository(),
		notifiers.GetNotifierService(),
		tasks_cancellation.GetRegistry(),
		encryption_secrets.GetSecretKeyService(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		defaultChainAlertMinInterval,
		sync.Mutex{},
		make(map[uuid.UUID]*runningStreamer),
		atomicTime{},
		make(map[chainAlertKey]time.Time),
		atomic.Bool{},
		atomic.Bool{},
	}
})

func GetPhysicalWalStreamSupervisor() *PhysicalWalStreamSupervisor {
	return getPhysicalWalStreamSupervisor()
}

var getPhysicalSlotCleanupListener = sync.OnceValue(func() *postgresql_executor.PhysicalSlotCleanupListener {
	return postgresql_executor.NewPhysicalSlotCleanupListener(
		databases.GetDatabaseService(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
	)
})

var getPhysicalBackupCanceller = sync.OnceValue(func() *PhysicalBackupCanceller {
	return NewPhysicalBackupCanceller(
		physical_repositories.GetInFlightBackupRepository(),
		tasks_cancellation.GetRequester(),
		logger.GetLogger(),
	)
})

func GetPhysicalBackupCanceller() *PhysicalBackupCanceller { return getPhysicalBackupCanceller() }

var getPhysicalBackupCancellationListener = sync.OnceValue(func() *PhysicalBackupCancellationListener {
	return &PhysicalBackupCancellationListener{
		getPhysicalBackupCanceller(),
		physical_repositories.GetWalStreamerRepository(),
		tasks_cancellation.GetRequester(),
		logger.GetLogger(),
	}
})

var SetupDependencies = sync.OnceFunc(func() {
	// Order matters: the cancellation listener stops the local pg_receivewal and
	// deletes the streamer row first, so the slot cleanup listener that runs next
	// can drop the (now detaching) WAL slot instead of refusing it as active and
	// leaving it to pin WAL forever.
	databases.GetDatabaseService().AddDbRemoveListener(getPhysicalBackupCancellationListener())
	// The cascade that would take the object names away runs only after every
	// listener has returned.
	databases.GetDatabaseService().AddDbRemoveListener(physical_service.GetPhysicalBackupService())
	storages.GetStorageService().AddStorageBackupCounter(physical_service.GetPhysicalBackupService())
	databases.GetDatabaseService().AddDbRemoveListener(getPhysicalSlotCleanupListener())
	backups_config_physical.GetBackupConfigService().
		SetBackupCancellationListener(getPhysicalBackupCancellationListener())
})
