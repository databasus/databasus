//go:build testhelpers

package backuping_physical

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
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

// CreateTestPhysicalBackuper returns a fully wired PhysicalBackuper for
// tests. The notification sender is parameterized so tests that don't want
// to exercise the notifier stack can inject a counting / no-op stub.
// Pass nil to use the production notifier service.
func CreateTestPhysicalBackuper(notificationSender NotificationSender) *PhysicalBackuper {
	sender := notificationSender
	if sender == nil {
		sender = notifiers.GetNotifierService()
	}

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
		sender,
		tasks_cancellation.GetRegistry(),
		encryption_secrets.GetSecretKeyService(),
		logger.GetLogger(),
		postgresql_executor.NewCreateFullBackupUsecase(),
		postgresql_executor.NewCreateIncrementalBackupUsecase(),
	}
}

// CreateTestPhysicalScheduler returns a scheduler wired to the production repos
// and its own in-process backuper.
func CreateTestPhysicalScheduler() *PhysicalBackupsScheduler {
	return &PhysicalBackupsScheduler{
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		backups_config_physical.GetBackupConfigService(),
		chain_view.GetChainViewService(),
		tasks_cancellation.GetRequester(),
		storages.GetStorageFileStore(),
		CreateTestPhysicalBackuper(nil),
		atomicTime{},
		logger.GetLogger(),
		atomic.Bool{},
		atomic.Bool{},
	}
}

type WalStreamSupervisorTestSpec struct {
	NotificationSender NotificationSender

	// Zero means the production default; the streamer spec in the executor package
	// reads zero as "disabled", hence the different name.
	ChainAlertMinIntervalOverride time.Duration

	// Nil means the process-wide registry.
	TaskCancellationRegistry *tasks_cancellation.Registry
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
		taskCancellationRegistry = tasks_cancellation.GetRegistry()
	}

	return &PhysicalWalStreamSupervisor{
		databases.GetDatabaseService(),
		backups_config_physical.GetBackupConfigService(),
		storages.GetStorageService(),
		storages.GetStorageFileStore(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalHistoryRepository(),
		physical_repositories.GetWalStreamerRepository(),
		physical_repositories.GetWalStreamSampleRepository(),
		spec.NotificationSender,
		taskCancellationRegistry,
		encryption_secrets.GetSecretKeyService(),
		encryption.GetFieldEncryptor(),
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
}

// StartPhysicalSchedulerForTest starts a fresh scheduler's Run loop in a
// goroutine and waits until it is ready (IsRunning). The scheduler invokes its
// own in-process backuper directly, so a backup requested over HTTP (config-enable
// bootstrap for the FULL, the trigger endpoint for incrementals) is picked up on
// the next tick and run to a terminal state. Returns a cancel func the caller
// defers; it cancels and waits for the goroutine to drain.
func StartPhysicalSchedulerForTest(t *testing.T) context.CancelFunc {
	t.Helper()

	scheduler := CreateTestPhysicalScheduler()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		scheduler.Run(ctx)
		close(done)
	}()

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if scheduler.IsRunning() {
			return func() {
				cancel()

				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Log("physical scheduler stop timeout")
				}
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("physical scheduler failed to start within timeout")

	return nil
}

// StartPhysicalWalStreamSupervisorForTest starts a fresh WAL stream supervisor's
// Run loop in a goroutine and waits until it is ready (IsRunning). It mirrors the
// supervisor wiring in cmd/main.go: with the supervisor up, enabling
// WAL-stream backups over the HTTP config API causes it to claim the database and
// create its replication slot on the next reconcile tick — so tests drive the slot
// lifecycle through the API instead of starting a streamer by hand. Returns a
// cancel func the caller defers; it cancels and waits for the goroutine to drain.
func StartPhysicalWalStreamSupervisorForTest(t *testing.T) context.CancelFunc {
	t.Helper()

	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender: notifiers.GetNotifierService(),
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		supervisor.Run(ctx)
		close(done)
	}()

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if supervisor.IsRunning() {
			return func() {
				cancel()

				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Log("physical wal stream supervisor stop timeout")
				}
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("physical wal stream supervisor failed to start within timeout")

	return nil
}

// RunOrphanWalCleanupForTest sweeps one database, not every enabled one: the test
// suite shares a metadata database across parallel packages, so a global pass
// would reclaim WAL belonging to another package's fixtures. Run cannot serve
// either — it is an infinite ticker that panics when called a second time in a
// process, and waiting out its tick would tie the assertion to timing rather than
// to sequence.
func RunOrphanWalCleanupForTest(t *testing.T, databaseID uuid.UUID) {
	t.Helper()

	CreateTestPhysicalCleaner().cleanOrphanWalForDatabase(t.Context(), logger.GetLogger(), databaseID)
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
