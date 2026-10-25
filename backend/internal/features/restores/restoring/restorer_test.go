package restoring

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	"databasus-backend/internal/features/notifiers"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/cache"
)

func Test_GetRestoreDatabaseCache_WhenProviderFails_UsesSuppliedMetadata(t *testing.T) {
	databaseConfiguration := &postgresql_logical.PostgresqlLogicalDatabase{Host: "database.internal"}
	databaseCacheFallback := &RestoreDatabaseCache{
		PostgresqlLogicalDatabase: databaseConfiguration,
	}
	restorer := &Restorer{
		restoreDatabaseCache: cache.NewJSONStore[RestoreDatabaseCache](
			NewFailingRestoreMetadataStore(errors.New("provider failed")),
			"restore_db",
		),
	}
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	databaseCache := restorer.getRestoreDatabaseCache(
		t.Context(),
		restoreMetadataLookup{
			restoreID:             uuid.New(),
			fallbackDatabaseCache: databaseCacheFallback,
		},
		testLogger,
	)

	assert.Same(t, databaseConfiguration, databaseCache.PostgresqlLogicalDatabase)
}

func Test_MakeRestore_WhenCacheMissed_RestoreFails(t *testing.T) {
	cache.GetStore().Clear(t.Context())

	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, storage)

	defer func() {
		backupRepo := backups_core_logical.BackupRepository{}
		backupsList, _ := backupRepo.FindByDatabaseID(database.ID)
		for _, backup := range backupsList {
			backupRepo.DeleteByID(backup.ID)
		}

		restoreRepo := restores_core.RestoreRepository{}
		restoresInProgress, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusInProgress)
		for _, restore := range restoresInProgress {
			restoreRepo.DeleteByID(restore.ID)
		}
		restoresFailed, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusFailed)
		for _, restore := range restoresFailed {
			restoreRepo.DeleteByID(restore.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

		cache.GetStore().Clear(t.Context())
	}()

	backup := backups_controllers_logical.CreateTestBackup(database.ID, storage.ID)

	// Create restore but DON'T cache DB credentials
	// Also don't set embedded DB fields to avoid schema issues
	restore := &restores_core.Restore{
		BackupID: backup.ID,
		Status:   restores_core.RestoreStatusInProgress,
	}
	err := getRestoreRepository().Save(restore)
	assert.NoError(t, err)

	// Create restorer and execute restore (should fail due to cache miss)
	restorer := CreateTestRestorer()
	restorer.MakeRestore(t.Context(), restore.ID, nil)

	// Verify restore failed with appropriate error message
	updatedRestore, err := getRestoreRepository().FindByID(restore.ID)
	assert.NoError(t, err)
	assert.Equal(t, restores_core.RestoreStatusFailed, updatedRestore.Status)
	assert.NotNil(t, updatedRestore.FailMessage)
	assert.Contains(
		t,
		*updatedRestore.FailMessage,
		"Database credentials expired or missing from cache",
	)
}

func Test_MakeRestore_WhenTaskStarts_CacheDeletedImmediately(t *testing.T) {
	cache.GetStore().Clear(t.Context())

	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, storage)

	defer func() {
		backupRepo := backups_core_logical.BackupRepository{}
		backupsList, _ := backupRepo.FindByDatabaseID(database.ID)
		for _, backup := range backupsList {
			backupRepo.DeleteByID(backup.ID)
		}

		restoreRepo := restores_core.RestoreRepository{}
		restoresInProgress, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusInProgress)
		for _, restore := range restoresInProgress {
			restoreRepo.DeleteByID(restore.ID)
		}
		restoresFailed, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusFailed)
		for _, restore := range restoresFailed {
			restoreRepo.DeleteByID(restore.ID)
		}
		restoresCompleted, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusCompleted)
		for _, restore := range restoresCompleted {
			restoreRepo.DeleteByID(restore.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

		cache.GetStore().Clear(t.Context())
	}()

	backup := backups_controllers_logical.CreateTestBackup(database.ID, storage.ID)

	// Create restore with cached DB credentials
	// Don't set embedded DB fields in the restore model itself
	restore := &restores_core.Restore{
		BackupID: backup.ID,
		Status:   restores_core.RestoreStatusInProgress,
	}
	err := getRestoreRepository().Save(restore)
	assert.NoError(t, err)

	// Cache DB credentials separately
	dbCache := &RestoreDatabaseCache{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Host:     config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "test",
			Password: "test",
			Database: new("testdb"),
			Version:  "16",
		},
	}
	require.NoError(t, getRestoreDatabaseCache().SetWithLifetime(
		t.Context(),
		cache.ExpiringValue[RestoreDatabaseCache]{
			Key:      restore.ID.String(),
			Value:    *dbCache,
			Lifetime: time.Hour,
		},
	))

	// Verify cache exists before restore starts
	cachedDB, err := getRestoreDatabaseCache().Get(t.Context(), restore.ID.String())
	require.NoError(t, err)
	assert.NotNil(t, cachedDB, "Cache should exist before restore starts")

	// Start restore (this will call GetAndDelete)
	restorer := CreateTestRestorer()
	restorer.MakeRestore(t.Context(), restore.ID, nil)

	// Verify cache was deleted immediately
	cachedDBAfter, err := getRestoreDatabaseCache().Get(t.Context(), restore.ID.String())
	require.NoError(t, err)
	assert.Nil(t, cachedDBAfter, "Cache should be deleted immediately when task starts")
}

func Test_MakeRestore_WhenShutdownSignalCloses_RestoreFailsWithShutdownMessage(t *testing.T) {
	restore, databaseCache := createRestoreForOutcomeTest(t)
	blockingUsecase := &MockBlockingRestoreUsecase{StartedChan: make(chan bool, 1)}
	shutdownSignal := make(chan struct{})
	restorer := createTestRestorerWithShutdownSignal(blockingUsecase, shutdownSignal)

	restoreFinished := startRestoreInBackground(t, restorer, restore.ID, databaseCache)
	waitForRestoreUsecaseStart(t, blockingUsecase)

	close(shutdownSignal)
	waitForRestoreToReturn(t, restoreFinished)

	persistedRestore, err := getRestoreRepository().FindByID(restore.ID)
	require.NoError(t, err)
	assert.Equal(t, restores_core.RestoreStatusFailed, persistedRestore.Status)
	require.NotNil(t, persistedRestore.FailMessage)
	assert.Contains(t, *persistedRestore.FailMessage, "restore interrupted by application shutdown")
}

func Test_MakeRestore_WhenCancellationRequested_RestoreCancelled(t *testing.T) {
	restore, databaseCache := createRestoreForOutcomeTest(t)
	blockingUsecase := &MockBlockingRestoreUsecase{StartedChan: make(chan bool, 1)}
	restorer := CreateTestRestorerWithUsecase(blockingUsecase)

	restoreFinished := startRestoreInBackground(t, restorer, restore.ID, databaseCache)
	waitForRestoreUsecaseStart(t, blockingUsecase)

	tasks_cancellation.GetRegistry().CancelRegisteredTask(restore.ID)
	waitForRestoreToReturn(t, restoreFinished)

	persistedRestore, err := getRestoreRepository().FindByID(restore.ID)
	require.NoError(t, err)
	assert.Equal(t, restores_core.RestoreStatusCanceled, persistedRestore.Status)
}

func Test_MakeRestore_WhenStorageErrorWrapsCanceled_RestoreFails(t *testing.T) {
	restore, databaseCache := createRestoreForOutcomeTest(t)
	storageErr := fmt.Errorf("read backup from storage: %w", context.Canceled)
	restorer := CreateTestRestorerWithUsecase(&MockFailedRestoreUsecase{RestoreErr: storageErr})

	restorer.MakeRestore(t.Context(), restore.ID, databaseCache)

	persistedRestore, err := getRestoreRepository().FindByID(restore.ID)
	require.NoError(t, err)
	assert.Equal(t, restores_core.RestoreStatusFailed, persistedRestore.Status)
	require.NotNil(t, persistedRestore.FailMessage)
	assert.Equal(t, storageErr.Error(), *persistedRestore.FailMessage)
}

type configStorageChoice int

const (
	configNamesBackupStorage configStorageChoice = iota
	configNamesOtherStorage
	configNamesNoStorage
)

type restoreTestFixture struct {
	restore         *restores_core.Restore
	databaseCache   *RestoreDatabaseCache
	backupStorageID uuid.UUID
}

func Test_MakeRestore_WhenConfigNamesAnotherStorage_ReadsFromBackupStorage(t *testing.T) {
	fixture := createRestoreWithConfigStorage(t, configNamesOtherStorage)
	storageCapturingUsecase := &MockCaptureStorageRestoreUsecase{}

	CreateTestRestorerWithUsecase(storageCapturingUsecase).
		MakeRestore(t.Context(), fixture.restore.ID, fixture.databaseCache)

	assertRestoreCompletedFromStorage(t, fixture, storageCapturingUsecase)
}

func Test_MakeRestore_WhenConfigNamesNoStorage_ReadsFromBackupStorage(t *testing.T) {
	fixture := createRestoreWithConfigStorage(t, configNamesNoStorage)
	storageCapturingUsecase := &MockCaptureStorageRestoreUsecase{}

	CreateTestRestorerWithUsecase(storageCapturingUsecase).
		MakeRestore(t.Context(), fixture.restore.ID, fixture.databaseCache)

	assertRestoreCompletedFromStorage(t, fixture, storageCapturingUsecase)
}

func assertRestoreCompletedFromStorage(
	t *testing.T,
	fixture restoreTestFixture,
	storageCapturingUsecase *MockCaptureStorageRestoreUsecase,
) {
	t.Helper()

	persistedRestore, err := getRestoreRepository().FindByID(fixture.restore.ID)
	require.NoError(t, err)
	assert.Equal(t, restores_core.RestoreStatusCompleted, persistedRestore.Status)
	require.NotNil(t, storageCapturingUsecase.ReceivedStorage)
	assert.Equal(t, fixture.backupStorageID, storageCapturingUsecase.ReceivedStorage.ID)
}

func createRestoreForOutcomeTest(t *testing.T) (*restores_core.Restore, *RestoreDatabaseCache) {
	t.Helper()

	fixture := createRestoreWithConfigStorage(t, configNamesBackupStorage)

	return fixture.restore, fixture.databaseCache
}

func createRestoreWithConfigStorage(t *testing.T, configStorage configStorageChoice) restoreTestFixture {
	t.Helper()

	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)
	backupStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, backupStorage, notifier)

	var otherStorage *storages.Storage

	switch configStorage {
	case configNamesBackupStorage:
		backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, backupStorage)
	case configNamesOtherStorage:
		otherStorage = storages.CreateTestStorage(workspace.ID)
		backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, otherStorage)
	case configNamesNoStorage:
	}

	backup := backups_controllers_logical.CreateTestBackup(database.ID, backupStorage.ID)

	restore := &restores_core.Restore{
		BackupID: backup.ID,
		Status:   restores_core.RestoreStatusInProgress,
	}
	require.NoError(t, getRestoreRepository().Save(restore))

	t.Cleanup(func() {
		_ = getRestoreRepository().DeleteByID(restore.ID)
		_ = (&backups_core_logical.BackupRepository{}).DeleteByID(backup.ID)
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), backupStorage.ID)
		if otherStorage != nil {
			storages.RemoveTestStorage(t.Context(), otherStorage.ID)
		}
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	})

	return restoreTestFixture{
		restore:         restore,
		databaseCache:   &RestoreDatabaseCache{PostgresqlLogicalDatabase: databases.GetTestPostgresConfig()},
		backupStorageID: backupStorage.ID,
	}
}

func startRestoreInBackground(
	t *testing.T,
	restorer *Restorer,
	restoreID uuid.UUID,
	databaseCache *RestoreDatabaseCache,
) <-chan struct{} {
	t.Helper()

	restoreFinished := make(chan struct{})

	go func() {
		defer close(restoreFinished)

		restorer.MakeRestore(context.Background(), restoreID, databaseCache)
	}()

	return restoreFinished
}

func waitForRestoreUsecaseStart(t *testing.T, blockingUsecase *MockBlockingRestoreUsecase) {
	t.Helper()

	select {
	case <-blockingUsecase.StartedChan:
	case <-time.After(30 * time.Second):
		t.Fatal("restore use case did not start")
	}
}

func waitForRestoreToReturn(t *testing.T, restoreFinished <-chan struct{}) {
	t.Helper()

	select {
	case <-restoreFinished:
	case <-time.After(10 * time.Second):
		t.Fatal("restore did not return")
	}
}
