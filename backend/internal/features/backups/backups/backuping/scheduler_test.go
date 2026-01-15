package backuping

import (
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	task_registry "databasus-backend/internal/features/tasks/registry"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	cache_utils "databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/period"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_RunPendingBackups_WhenLastBackupWasYesterday_CreatesNewBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database
	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add old backup
	backupRepository.Save(&backups_core.Backup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status: backups_core.BackupStatusCompleted,

		CreatedAt: time.Now().UTC().Add(-24 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	// Wait for backup to complete (runs in goroutine)
	WaitForBackupCompletion(t, database.ID, 1, 10*time.Second)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 2)

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenLastBackupWasRecentlyCompleted_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database
	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add recent backup (1 hour ago)
	backupRepository.Save(&backups_core.Backup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status: backups_core.BackupStatusCompleted,

		CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1) // Should still be 1 backup, no new backup created

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenLastBackupFailedAndRetriesDisabled_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database with retries disabled
	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID
	backupConfig.IsRetryIfFailed = false
	backupConfig.MaxFailedTriesCount = 0

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add failed backup
	failMessage := "backup failed"
	backupRepository.Save(&backups_core.Backup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status:      backups_core.BackupStatusFailed,
		FailMessage: &failMessage,

		CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1) // Should still be 1 backup, no retry attempted

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenLastBackupFailedAndRetriesEnabled_CreatesNewBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database with retries enabled
	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID
	backupConfig.IsRetryIfFailed = true
	backupConfig.MaxFailedTriesCount = 3

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add failed backup
	failMessage := "backup failed"
	backupRepository.Save(&backups_core.Backup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status:      backups_core.BackupStatusFailed,
		FailMessage: &failMessage,

		CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	// Wait for backup to complete (runs in goroutine)
	WaitForBackupCompletion(t, database.ID, 1, 10*time.Second)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 2) // Should have 2 backups, retry was attempted

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenFailedBackupsExceedMaxRetries_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database with retries enabled
	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID
	backupConfig.IsRetryIfFailed = true
	backupConfig.MaxFailedTriesCount = 3

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	failMessage := "backup failed"

	for range 3 {
		backupRepository.Save(&backups_core.Backup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,

			Status:      backups_core.BackupStatusFailed,
			FailMessage: &failMessage,

			CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
		})
	}

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 3) // Should have 3 backups, not more than max

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenBackupsDisabled_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = false
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add old backup that would trigger new backup if enabled
	backupRepository.Save(&backups_core.Backup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status: backups_core.BackupStatusCompleted,

		CreatedAt: time.Now().UTC().Add(-24 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_CheckDeadNodesAndFailBackups_WhenNodeDies_FailsBackupAndCleansUpRegistry(t *testing.T) {
	cache_utils.ClearAllCache()

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	var mockNodeID uuid.UUID

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)

		// Clean up mock node
		if mockNodeID != uuid.Nil {
			nodesRegistry.UnregisterNodeFromRegistry(task_registry.TaskNode{ID: mockNodeID})
		}
		cache_utils.ClearAllCache()
	}()

	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Register mock node without subscribing to backups (simulates node crash after registration)
	mockNodeID = uuid.New()
	err = CreateMockNodeInRegistry(mockNodeID, 100, time.Now().UTC())
	assert.NoError(t, err)

	// Scheduler assigns backup to mock node
	GetBackupsScheduler().StartBackup(database.ID, false)
	time.Sleep(100 * time.Millisecond)

	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, backups_core.BackupStatusInProgress, backups[0].Status)

	// Verify Valkey counter was incremented when backup was assigned
	stats, err := nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	foundStat := false
	for _, stat := range stats {
		if stat.ID == mockNodeID {
			assert.Equal(t, 1, stat.ActiveTasks)
			foundStat = true
			break
		}
	}
	assert.True(t, foundStat, "Node stats should be present")

	// Simulate node death by setting heartbeat older than 2-minute threshold
	oldHeartbeat := time.Now().UTC().Add(-3 * time.Minute)
	err = UpdateNodeHeartbeatDirectly(mockNodeID, 100, oldHeartbeat)
	assert.NoError(t, err)

	// Trigger dead node detection
	err = GetBackupsScheduler().checkDeadNodesAndFailBackups()
	assert.NoError(t, err)

	// Verify backup was failed with appropriate error message
	backups, err = backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, backups_core.BackupStatusFailed, backups[0].Status)
	assert.NotNil(t, backups[0].FailMessage)
	assert.Contains(t, *backups[0].FailMessage, "node unavailability")

	// Verify Valkey counter was decremented after backup failed
	stats, err = nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	for _, stat := range stats {
		if stat.ID == mockNodeID {
			assert.Equal(t, 0, stat.ActiveTasks)
		}
	}

	time.Sleep(200 * time.Millisecond)
}

func Test_OnBackupCompleted_WhenTaskIsNotBackup_SkipsProcessing(t *testing.T) {
	cache_utils.ClearAllCache()

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	var mockNodeID uuid.UUID

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)

		// Clean up mock node
		if mockNodeID != uuid.Nil {
			nodesRegistry.UnregisterNodeFromRegistry(task_registry.TaskNode{ID: mockNodeID})
		}
		cache_utils.ClearAllCache()
	}()

	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Register mock node
	mockNodeID = uuid.New()
	err = CreateMockNodeInRegistry(mockNodeID, 100, time.Now().UTC())
	assert.NoError(t, err)

	// Start a backup and assign it to the node
	GetBackupsScheduler().StartBackup(database.ID, false)
	time.Sleep(100 * time.Millisecond)

	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, backups_core.BackupStatusInProgress, backups[0].Status)

	// Get initial state of the registry
	initialStats, err := nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	var initialActiveTasks int
	for _, stat := range initialStats {
		if stat.ID == mockNodeID {
			initialActiveTasks = stat.ActiveTasks
			break
		}
	}
	assert.Equal(t, 1, initialActiveTasks, "Should have 1 active task")

	// Call onBackupCompleted with a random UUID (not a backup ID)
	nonBackupTaskID := uuid.New()
	GetBackupsScheduler().onBackupCompleted(mockNodeID.String(), nonBackupTaskID)

	time.Sleep(100 * time.Millisecond)

	// Verify: Active tasks counter should remain the same (not decremented)
	stats, err := nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	for _, stat := range stats {
		if stat.ID == mockNodeID {
			assert.Equal(t, initialActiveTasks, stat.ActiveTasks,
				"Active tasks should not change for non-backup task")
		}
	}

	// Verify: backup should still be in progress (not modified)
	backups, err = backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, backups_core.BackupStatusInProgress, backups[0].Status,
		"Backup status should not change for non-backup task completion")

	// Verify: backupToNodeRelations should still contain the node
	scheduler := GetBackupsScheduler()
	_, exists := scheduler.backupToNodeRelations[mockNodeID]
	assert.True(t, exists, "Node should still be in backupToNodeRelations")

	time.Sleep(200 * time.Millisecond)
}

func Test_CalculateLeastBusyNode_SelectsNodeWithBestScore(t *testing.T) {
	t.Run("Nodes with same throughput", func(t *testing.T) {
		cache_utils.ClearAllCache()

		node1ID := uuid.New()
		node2ID := uuid.New()
		node3ID := uuid.New()
		now := time.Now().UTC()

		defer func() {
			// Clean up all mock nodes
			nodesRegistry.UnregisterNodeFromRegistry(task_registry.TaskNode{ID: node1ID})
			nodesRegistry.UnregisterNodeFromRegistry(task_registry.TaskNode{ID: node2ID})
			nodesRegistry.UnregisterNodeFromRegistry(task_registry.TaskNode{ID: node3ID})
			cache_utils.ClearAllCache()
		}()

		err := CreateMockNodeInRegistry(node1ID, 100, now)
		assert.NoError(t, err)
		err = CreateMockNodeInRegistry(node2ID, 100, now)
		assert.NoError(t, err)
		err = CreateMockNodeInRegistry(node3ID, 100, now)
		assert.NoError(t, err)

		for range 5 {
			err = nodesRegistry.IncrementTasksInProgress(node1ID.String())
			assert.NoError(t, err)
		}

		for range 2 {
			err = nodesRegistry.IncrementTasksInProgress(node2ID.String())
			assert.NoError(t, err)
		}

		for range 8 {
			err = nodesRegistry.IncrementTasksInProgress(node3ID.String())
			assert.NoError(t, err)
		}

		leastBusyNodeID, err := GetBackupsScheduler().calculateLeastBusyNode()
		assert.NoError(t, err)
		assert.NotNil(t, leastBusyNodeID)
		assert.Equal(t, node2ID, *leastBusyNodeID)
	})

	t.Run("Nodes with different throughput", func(t *testing.T) {
		cache_utils.ClearAllCache()

		node100MBsID := uuid.New()
		node50MBsID := uuid.New()
		now := time.Now().UTC()

		defer func() {
			// Clean up all mock nodes
			nodesRegistry.UnregisterNodeFromRegistry(task_registry.TaskNode{ID: node100MBsID})
			nodesRegistry.UnregisterNodeFromRegistry(task_registry.TaskNode{ID: node50MBsID})
			cache_utils.ClearAllCache()
		}()

		err := CreateMockNodeInRegistry(node100MBsID, 100, now)
		assert.NoError(t, err)
		err = CreateMockNodeInRegistry(node50MBsID, 50, now)
		assert.NoError(t, err)

		for range 10 {
			err = nodesRegistry.IncrementTasksInProgress(node100MBsID.String())
			assert.NoError(t, err)
		}

		err = nodesRegistry.IncrementTasksInProgress(node50MBsID.String())
		assert.NoError(t, err)

		leastBusyNodeID, err := GetBackupsScheduler().calculateLeastBusyNode()
		assert.NoError(t, err)
		assert.NotNil(t, leastBusyNodeID)
		assert.Equal(t, node50MBsID, *leastBusyNodeID)
	})
}

func Test_FailBackupsInProgress_WhenSchedulerStarts_CancelsBackupsAndUpdatesStatus(t *testing.T) {
	cache_utils.ClearAllCache()

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)

		cache_utils.ClearAllCache()
	}()

	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Create two in-progress backups that should be failed on scheduler restart
	backup1 := &backups_core.Backup{
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusInProgress,
		BackupSizeMb: 10.5,
		CreatedAt:    time.Now().UTC().Add(-30 * time.Minute),
	}
	err = backupRepository.Save(backup1)
	assert.NoError(t, err)

	backup2 := &backups_core.Backup{
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusInProgress,
		BackupSizeMb: 5.2,
		CreatedAt:    time.Now().UTC().Add(-15 * time.Minute),
	}
	err = backupRepository.Save(backup2)
	assert.NoError(t, err)

	// Create a completed backup to verify it's not affected by failBackupsInProgress
	completedBackup := &backups_core.Backup{
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 20.0,
		CreatedAt:    time.Now().UTC().Add(-1 * time.Hour),
	}
	err = backupRepository.Save(completedBackup)
	assert.NoError(t, err)

	// Trigger the scheduler's failBackupsInProgress logic
	// This should cancel in-progress backups and mark them as failed
	err = GetBackupsScheduler().failBackupsInProgress()
	assert.NoError(t, err)

	// Verify all backups exist and were processed correctly
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 3)

	var failedCount int
	var completedCount int
	for _, backup := range backups {
		switch backup.Status {
		case backups_core.BackupStatusFailed:
			failedCount++
			// Verify fail message indicates application restart
			assert.NotNil(t, backup.FailMessage)
			assert.Equal(t, "Backup failed due to application restart", *backup.FailMessage)
			// Verify backup size was reset to 0
			assert.Equal(t, float64(0), backup.BackupSizeMb)
		case backups_core.BackupStatusCompleted:
			completedCount++
		}
	}

	// Verify correct number of backups in each state
	assert.Equal(t, 2, failedCount)
	assert.Equal(t, 1, completedCount)

	time.Sleep(200 * time.Millisecond)
}

func Test_StartBackup_WhenBackupCompletes_DecrementsActiveTaskCount(t *testing.T) {
	cache_utils.ClearAllCache()

	// Start scheduler so it can handle task completions
	schedulerCancel := StartSchedulerForTest(t)
	defer schedulerCancel()

	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Get initial active task count
	stats, err := nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	var initialActiveTasks int
	for _, stat := range stats {
		if stat.ID == backuperNode.nodeID {
			initialActiveTasks = stat.ActiveTasks
			break
		}
	}
	t.Logf("Initial active tasks: %d", initialActiveTasks)

	// Start backup
	GetBackupsScheduler().StartBackup(database.ID, false)

	// Wait for backup to complete
	WaitForBackupCompletion(t, database.ID, 0, 10*time.Second)

	// Verify backup was created and completed
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, backups_core.BackupStatusCompleted, backups[0].Status)

	// Wait for active task count to decrease
	decreased := WaitForActiveTasksDecrease(
		t,
		backuperNode.nodeID,
		initialActiveTasks+1,
		10*time.Second,
	)
	assert.True(t, decreased, "Active task count should have decreased after backup completion")

	// Verify final active task count equals initial count
	finalStats, err := nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	for _, stat := range finalStats {
		if stat.ID == backuperNode.nodeID {
			t.Logf("Final active tasks: %d", stat.ActiveTasks)
			assert.Equal(t, initialActiveTasks, stat.ActiveTasks,
				"Active task count should return to initial value after backup completion")
			break
		}
	}

	time.Sleep(200 * time.Millisecond)
}

func Test_StartBackup_WhenBackupFails_DecrementsActiveTaskCount(t *testing.T) {
	cache_utils.ClearAllCache()

	// Start scheduler so it can handle task completions
	schedulerCancel := StartSchedulerForTest(t)
	defer schedulerCancel()

	backuperNode := CreateTestBackuperNode()
	cancel := StartBackuperNodeForTest(t, backuperNode)
	defer StopBackuperNodeForTest(t, cancel, backuperNode)

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Set wrong password to cause backup failure
	// We need to bypass service layer validation which would fail on connection test
	database.Postgresql.Password = "intentionally_wrong_password"
	dbRepo := &databases.DatabaseRepository{}
	_, err := dbRepo.Save(database)
	assert.NoError(t, err)

	backupConfig, err := backups_config.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.StorePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Get initial active task count
	stats, err := nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	var initialActiveTasks int
	for _, stat := range stats {
		if stat.ID == backuperNode.nodeID {
			initialActiveTasks = stat.ActiveTasks
			break
		}
	}
	t.Logf("Initial active tasks: %d", initialActiveTasks)

	// Start backup
	GetBackupsScheduler().StartBackup(database.ID, false)

	// Wait for backup to fail
	WaitForBackupCompletion(t, database.ID, 0, 10*time.Second)

	// Verify backup was created and failed
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, backups_core.BackupStatusFailed, backups[0].Status)
	assert.NotNil(t, backups[0].FailMessage)
	if backups[0].FailMessage != nil {
		t.Logf("Backup failed with message: %s", *backups[0].FailMessage)
	}

	// Wait for active task count to decrease
	decreased := WaitForActiveTasksDecrease(
		t,
		backuperNode.nodeID,
		initialActiveTasks+1,
		10*time.Second,
	)
	assert.True(t, decreased, "Active task count should have decreased after backup failure")

	// Verify final active task count equals initial count
	finalStats, err := nodesRegistry.GetNodesStats()
	assert.NoError(t, err)
	for _, stat := range finalStats {
		if stat.ID == backuperNode.nodeID {
			t.Logf("Final active tasks: %d", stat.ActiveTasks)
			assert.Equal(t, initialActiveTasks, stat.ActiveTasks,
				"Active task count should return to initial value after backup failure")
			break
		}
	}

	time.Sleep(200 * time.Millisecond)
}
