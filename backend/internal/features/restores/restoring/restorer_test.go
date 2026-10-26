package restoring

import (
	"bytes"
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
	"databasus-backend/internal/features/caching"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	restores_core "databasus-backend/internal/features/restores/core"
	storages_bridge "databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
)

func Test_GetRestoreDatabaseCache_WhenProviderFails_UsesSuppliedMetadata(t *testing.T) {
	databaseConfiguration := &restores_core.PostgresqlTargetDatabase{Host: "database.internal"}
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
	caching.GetStore().Clear(t.Context())

	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user)
	storage := storages_bridge.CreateTestStorage(workspace.ID)
	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, storage.ID)

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

		databases_bridge.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers_bridge.RemoveTestNotifier(notifier)
		storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

		caching.GetStore().Clear(t.Context())
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
	caching.GetStore().Clear(t.Context())

	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user)
	storage := storages_bridge.CreateTestStorage(workspace.ID)
	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, storage.ID)

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

		databases_bridge.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers_bridge.RemoveTestNotifier(notifier)
		storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)

		caching.GetStore().Clear(t.Context())
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
		PostgresqlLogicalDatabase: &restores_core.PostgresqlTargetDatabase{
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
	restore       *restores_core.Restore
	databaseCache *RestoreDatabaseCache
	backupContent []byte
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
	assert.Equal(t, fixture.backupContent, storageCapturingUsecase.ReadBackupContent)
}

func createRestoreForOutcomeTest(t *testing.T) (*restores_core.Restore, *RestoreDatabaseCache) {
	t.Helper()

	fixture := createRestoreWithConfigStorage(t, configNamesBackupStorage)

	return fixture.restore, fixture.databaseCache
}

func createRestoreWithConfigStorage(t *testing.T, configStorage configStorageChoice) restoreTestFixture {
	t.Helper()

	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user)
	backupStorage := storages_bridge.CreateTestStorage(workspace.ID)
	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)
	database := databases_bridge.CreateTestDatabase(workspace.ID, notifier)

	var otherStorage storages.Storage

	switch configStorage {
	case configNamesBackupStorage:
		backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, backupStorage.ID)
	case configNamesOtherStorage:
		otherStorage = storages_bridge.CreateTestStorage(workspace.ID)
		backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, otherStorage.ID)
	case configNamesNoStorage:
	}

	backup := backups_controllers_logical.CreateTestBackup(database.ID, backupStorage.ID)
	backup.FileName = backup.ID.String()
	require.NoError(t, (&backups_core_logical.BackupRepository{}).Save(backup))

	backupContent := []byte("content of backup " + backup.ID.String())
	_, err := storages_bridge.GetModule().WriteFile(t.Context(),
		storages.StoredFileReference{StorageID: backupStorage.ID, FileName: backup.FileName},
		bytes.NewReader(backupContent))
	require.NoError(t, err)

	restore := &restores_core.Restore{
		BackupID: backup.ID,
		Status:   restores_core.RestoreStatusInProgress,
	}
	require.NoError(t, getRestoreRepository().Save(restore))

	t.Cleanup(func() {
		_ = getRestoreRepository().DeleteByID(restore.ID)
		_ = (&backups_core_logical.BackupRepository{}).DeleteByID(backup.ID)
		databases_bridge.RemoveTestDatabase(t.Context(), database)
		notifiers_bridge.RemoveTestNotifier(notifier)
		// RemoveTestStorage would leave the backup file this fixture wrote in the data folder.
		require.NoError(t, storages_bridge.DeleteTestFile(t.Context(),
			storages.StoredFileReference{StorageID: backupStorage.ID, FileName: backup.FileName}))
		storages_bridge.RemoveTestStorage(t.Context(), backupStorage.ID)
		if otherStorage.ID != uuid.Nil {
			storages_bridge.RemoveTestStorage(t.Context(), otherStorage.ID)
		}
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	})

	return restoreTestFixture{
		restore:       restore,
		databaseCache: &RestoreDatabaseCache{PostgresqlLogicalDatabase: restores_core.GetTestPostgresTarget()},
		backupContent: backupContent,
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
