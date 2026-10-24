//go:build testhelpers

package restoring

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/restores/usecases"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

type failingRestoreMetadataStore struct {
	err error
}

func NewFailingRestoreMetadataStore(providerError error) cache.Store {
	return failingRestoreMetadataStore{err: providerError}
}

func (s failingRestoreMetadataStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingRestoreMetadataStore) Set(context.Context, cache.Entry) error {
	return s.err
}

func (s failingRestoreMetadataStore) CreateIfAbsent(context.Context, cache.Entry) (bool, error) {
	return false, s.err
}

func (s failingRestoreMetadataStore) ReadAndDelete(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingRestoreMetadataStore) Delete(context.Context, string) error {
	return s.err
}

func (s failingRestoreMetadataStore) Clear(context.Context) error {
	return s.err
}

func CreateTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config_logical.GetBackupConfigController(),
	)

	return router
}

func CreateTestRestorer() *Restorer {
	return CreateTestRestorerWithUsecase(usecases.GetRestoreBackupUsecase())
}

func CreateTestRestorerWithUsecase(usecase restores_core.RestoreBackupUsecase) *Restorer {
	return createTestRestorerWithShutdownSignal(usecase, make(chan struct{}))
}

func createTestRestorerWithShutdownSignal(
	usecase restores_core.RestoreBackupUsecase,
	shutdownSignal <-chan struct{},
) *Restorer {
	return &Restorer{
		databases.GetDatabaseService(),
		backups_services.GetBackupService(),
		encryption.GetFieldEncryptor(),
		restoreRepository,
		backups_config_logical.GetBackupConfigService(),
		storages.GetStorageService(),
		logger.GetLogger(),
		usecase,
		restoreDatabaseCache,
		tasks_cancellation.GetRegistry(),
		shutdownSignal,
	}
}

func CreateTestRestoresScheduler() *RestoresScheduler {
	return &RestoresScheduler{
		restoreRepository,
		time.Now().UTC(),
		logger.GetLogger(),
		restorer,
		restoreDatabaseCache,
		atomic.Bool{},
	}
}

// WaitForRestoreCompletion waits for a restore to be completed (or failed)
func WaitForRestoreCompletion(
	t *testing.T,
	restoreID uuid.UUID,
	timeout time.Duration,
) {
	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		restore, err := restoreRepository.FindByID(restoreID)
		if err != nil {
			t.Logf("WaitForRestoreCompletion: error finding restore: %v", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}

		t.Logf("WaitForRestoreCompletion: restore status: %s", restore.Status)

		if restore.Status == restores_core.RestoreStatusCompleted ||
			restore.Status == restores_core.RestoreStatusFailed {
			t.Logf(
				"WaitForRestoreCompletion: restore finished with status %s",
				restore.Status,
			)
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("WaitForRestoreCompletion: timeout waiting for restore to complete")
}

// StartRestorerForTest installs the given restorer as the DI scheduler's
// in-process executor for the duration of a test, so a test-injected usecase
// (e.g. a blocking or capturing mock) actually runs when StartRestore dispatches.
// Returns a cancel function that restores the original executor.
func StartRestorerForTest(_ *testing.T, restorerForTest *Restorer) context.CancelFunc {
	previous := restoresScheduler.restorer
	restoresScheduler.restorer = restorerForTest

	return func() {
		restoresScheduler.restorer = previous
	}
}

// StartSchedulerForTest starts the RestoresScheduler in a goroutine for testing.
// Returns a context cancel function that should be deferred to stop the scheduler.
func StartSchedulerForTest(t *testing.T, scheduler *RestoresScheduler) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		scheduler.Run(ctx)
		close(done)
	}()

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if scheduler.IsSchedulerRunning() {
			t.Log("RestoresScheduler started")

			return func() {
				cancel()
				select {
				case <-done:
					t.Log("RestoresScheduler stopped gracefully")
				case <-time.After(2 * time.Second):
					t.Log("RestoresScheduler stop timeout")
				}
			}
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatal("RestoresScheduler failed to start within timeout")

	return nil
}

// StopRestorerForTest restores the DI scheduler's original executor.
func StopRestorerForTest(_ *testing.T, cancel context.CancelFunc, _ *Restorer) {
	cancel()
}

// CreateTestRestore creates a test restore with the given backup and status
func CreateTestRestore(
	t *testing.T,
	backup *backups_core_logical.LogicalBackup,
	status restores_core.RestoreStatus,
) *restores_core.Restore {
	restore := &restores_core.Restore{
		BackupID: backup.ID,
		Status:   status,
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Host:     config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "test",
			Password: "test",
			Database: func() *string { s := "testdb"; return &s }(),
			Version:  "16",
		},
	}

	err := restoreRepository.Save(restore)
	if err != nil {
		t.Fatalf("Failed to create test restore: %v", err)
	}

	return restore
}
