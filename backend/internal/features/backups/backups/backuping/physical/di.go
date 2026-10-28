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
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getPhysicalBackuper = sync.OnceValue(func() *PhysicalBackuper {
	return &PhysicalBackuper{
		databases_bridge.GetAccess(),
		databases_bridge.GetModule,
		workspaces_bridge.GetAccess(),
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		physical_repositories.GetWalHistoryRepository(),
		backups_bridge.GetModule,
		storages_bridge.GetModule,
		notifiers_bridge.GetSender(),
		backups_bridge.GetCancellationRegistry(),
		encryption.GetMasterKey(),
		logger.GetLogger(),
		postgresql_executor.NewCreateFullBackupUsecase(),
		postgresql_executor.NewCreateIncrementalBackupUsecase(),
		sync.WaitGroup{},
	}
})

func GetPhysicalBackuper() *PhysicalBackuper { return getPhysicalBackuper() }

func NewPhysicalBackupsScheduler() *PhysicalBackupsScheduler {
	return &PhysicalBackupsScheduler{
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		backups_bridge.GetModule,
		chain_view.GetChainViewService(),
		backups_bridge.GetCancellationRegistry(),
		storages_bridge.GetModule,
		getPhysicalBackuper(),
		atomicTime{},
		logger.GetLogger(),
		atomic.Bool{},
		atomic.Bool{},
	}
}

func NewPhysicalBackupCleaner() *PhysicalBackupCleaner {
	return &PhysicalBackupCleaner{
		physical_service.GetPhysicalBackupService(),
		chain_view.GetChainViewService(),
		backups_bridge.GetModule,
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalStreamSampleRepository(),
		logger.GetLogger(),
		atomic.Bool{},
	}
}

func NewPhysicalWalStreamSupervisor(timings backups.WalStreamTimings) *PhysicalWalStreamSupervisor {
	return &PhysicalWalStreamSupervisor{
		databases_bridge.GetAccess(),
		databases_bridge.GetModule,
		backups_bridge.GetModule,
		storages_bridge.GetModule,
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalHistoryRepository(),
		physical_repositories.GetWalStreamerRepository(),
		physical_repositories.GetWalStreamSampleRepository(),
		notifiers_bridge.GetSender(),
		backups_bridge.GetCancellationRegistry(),
		encryption.GetMasterKey(),
		logger.GetLogger(),
		defaultChainAlertMinInterval,
		timings,
		sync.Mutex{},
		make(map[uuid.UUID]*runningStreamer),
		atomicTime{},
		make(map[chainAlertKey]time.Time),
		atomic.Bool{},
		atomic.Bool{},
	}
}

var getPhysicalSlotCleanupListener = sync.OnceValue(func() *postgresql_executor.PhysicalSlotCleanupListener {
	return postgresql_executor.NewPhysicalSlotCleanupListener(databases_bridge.GetModule, logger.GetLogger())
})

func GetPhysicalSlotCleanupListener() *postgresql_executor.PhysicalSlotCleanupListener {
	return getPhysicalSlotCleanupListener()
}

var getPhysicalBackupCanceller = sync.OnceValue(func() *PhysicalBackupCanceller {
	return NewPhysicalBackupCanceller(
		physical_repositories.GetInFlightBackupRepository(),
		backups_bridge.GetCancellationRegistry(),
		logger.GetLogger(),
	)
})

func GetPhysicalBackupCanceller() *PhysicalBackupCanceller { return getPhysicalBackupCanceller() }

var getPhysicalBackupCancellationListener = sync.OnceValue(func() *PhysicalBackupCancellationListener {
	return &PhysicalBackupCancellationListener{
		getPhysicalBackupCanceller(),
		physical_repositories.GetWalStreamerRepository(),
		backups_bridge.GetCancellationRegistry(),
		logger.GetLogger(),
	}
})

func GetPhysicalBackupCancellationListener() *PhysicalBackupCancellationListener {
	return getPhysicalBackupCancellationListener()
}
