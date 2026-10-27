//go:build testhelpers

package backuping_physical

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

// CreateTestPhysicalBackuper returns a fully wired PhysicalBackuper for
// tests. The notification sender is parameterized so tests that don't want
// to exercise the notifier stack can inject a counting / no-op stub.
// Pass nil to use notifiers_bridge.GetSender(), which hands notifications to the notifiers module.
func CreateTestPhysicalBackuper(notificationSender NotificationSender) *PhysicalBackuper {
	sender := notificationSender
	if sender == nil {
		sender = notifiers_bridge.GetSender()
	}

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
		sender,
		backups_bridge.GetCancellationRegistry(),
		encryption.GetMasterKey(),
		logger.GetLogger(),
		postgresql_executor.NewCreateFullBackupUsecase(),
		postgresql_executor.NewCreateIncrementalBackupUsecase(),
	}
}

func CreateTestPhysicalScheduler() *PhysicalBackupsScheduler {
	return NewPhysicalBackupsScheduler()
}

type WalStreamSupervisorTestSpec struct {
	NotificationSender NotificationSender

	// Zero means the production default; the streamer spec in the executor package
	// reads zero as "disabled", hence the different name.
	ChainAlertMinIntervalOverride time.Duration

	// Nil means the process-wide registry.
	TaskCancellationRegistry *background.CancellationRegistry
}

// CreateTestWalStreamSupervisor returns a fresh WAL stream supervisor wired to
// the production repos and services. A fresh instance (not a copy of the DI
// singleton) keeps each test's hasRun/running state isolated and avoids copying
// the embedded mutex.
func CreateTestWalStreamSupervisor(spec WalStreamSupervisorTestSpec) *PhysicalWalStreamSupervisor {
	chainAlertMinInterval := spec.ChainAlertMinIntervalOverride
	if chainAlertMinInterval == 0 {
		chainAlertMinInterval = defaultChainAlertMinInterval
	}

	taskCancellationRegistry := spec.TaskCancellationRegistry
	if taskCancellationRegistry == nil {
		taskCancellationRegistry = backups_bridge.GetCancellationRegistry()
	}

	return &PhysicalWalStreamSupervisor{
		databases_bridge.GetAccess(),
		databases_bridge.GetModule,
		backups_bridge.GetModule,
		storages_bridge.GetModule,
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalHistoryRepository(),
		physical_repositories.GetWalStreamerRepository(),
		physical_repositories.GetWalStreamSampleRepository(),
		spec.NotificationSender,
		taskCancellationRegistry,
		encryption.GetMasterKey(),
		logger.GetLogger(),
		chainAlertMinInterval,
		sync.Mutex{},
		make(map[uuid.UUID]*runningStreamer),
		atomicTime{},
		make(map[chainAlertKey]time.Time),
		atomic.Bool{},
		atomic.Bool{},
	}
}

// CreateTestPhysicalCleaner returns a cleaner wired to the production service +
// repos.
func CreateTestPhysicalCleaner() *PhysicalBackupCleaner {
	return NewPhysicalBackupCleaner()
}

// RegisterTestRunningStreamer makes GetLiveState answer for the database with a
// streamer supervisor that never runs, so a test reads its current values
// without streaming WAL. The registration is removed when the test ends.
func RegisterTestRunningStreamer(
	t *testing.T,
	databaseID uuid.UUID,
	streamer *postgresql_executor.WalStreamSupervisor,
) {
	t.Helper()

	supervisor := GetPhysicalWalStreamSupervisor()

	supervisor.mu.Lock()
	supervisor.running[databaseID] = &runningStreamer{
		cancel:              func() {},
		done:                make(chan struct{}),
		walStreamSupervisor: streamer,
	}
	supervisor.mu.Unlock()

	t.Cleanup(func() {
		supervisor.mu.Lock()
		delete(supervisor.running, databaseID)
		supervisor.mu.Unlock()
	})
}
