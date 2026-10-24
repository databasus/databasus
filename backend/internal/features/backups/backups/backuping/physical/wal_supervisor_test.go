package backuping_physical

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/storage"
)

// seedStreamerRow inserts a physical_wal_streamers row with an explicit
// last_heartbeat_at and status, for ClaimIfClaimable tests. database_id must FK
// to a real databases row.
func seedStreamerRow(
	t *testing.T,
	databaseID uuid.UUID,
	heartbeatAt time.Time,
	status physical_enums.PhysicalWalStreamerStatus,
) {
	t.Helper()

	require.NoError(t, storage.GetDb().Exec(`
		INSERT INTO physical_wal_streamers (database_id, started_at, last_heartbeat_at, status)
		VALUES (?, NOW(), ?, ?)
		ON CONFLICT (database_id) DO UPDATE
		  SET last_heartbeat_at = EXCLUDED.last_heartbeat_at, status = EXCLUDED.status
	`, databaseID, heartbeatAt, status).Error)

	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(databaseID)
	})
}

func Test_ClaimIfClaimable_WhenNoRowExists_InsertsAndClaims(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(prereqs.DB.ID)
	})

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.True(t, claimed, "a missing row must be insert-claimed")

	row, err := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, physical_enums.PhysicalWalStreamerStatusRunning, row.Status)
}

func Test_ClaimIfClaimable_WhenFreshRunningRow_NotClaimable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(t, prereqs.DB.ID, time.Now().UTC(), physical_enums.PhysicalWalStreamerStatusRunning)

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.False(t, claimed, "a fresh RUNNING streamer owned by a live process must not be reclaimed")
}

func Test_ClaimIfClaimable_WhenHeartbeatStale_Claimable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(
		t,
		prereqs.DB.ID,
		time.Now().UTC().Add(-5*time.Minute),
		physical_enums.PhysicalWalStreamerStatusRunning,
	)

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.True(t, claimed, "a streamer whose heartbeat aged past staleness must be reclaimable")
}

func Test_ClaimIfClaimable_WhenFailedRow_Claimable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(t, prereqs.DB.ID, time.Now().UTC(), physical_enums.PhysicalWalStreamerStatusFailed)

	claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
	require.NoError(t, err)
	require.True(t, claimed, "a FAILED streamer must be reclaimable even with a fresh heartbeat")
}

func Test_ClaimIfClaimable_WhenConcurrentClaims_ExactlyOneWins(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(
		t,
		prereqs.DB.ID,
		time.Now().UTC().Add(-5*time.Minute),
		physical_enums.PhysicalWalStreamerStatusRunning,
	)

	const racers = 8

	type claimOutcome struct {
		claimed bool
		err     error
	}

	results := make(chan claimOutcome, racers)
	start := make(chan struct{})

	for range racers {
		go func() {
			<-start

			// Don't call require.* off the test goroutine — collect and assert below.
			claimed, err := physical_repositories.GetWalStreamerRepository().ClaimIfClaimable(prereqs.DB.ID, 90)
			results <- claimOutcome{claimed: claimed, err: err}
		}()
	}

	close(start)

	wins := 0
	for range racers {
		outcome := <-results
		require.NoError(t, outcome.err)

		if outcome.claimed {
			wins++
		}
	}

	require.Equal(t, 1, wins, "exactly one concurrent claimant may win the stale streamer")
}

func Test_PhysicalWalStreamSupervisorRun_WhenCalledTwice_Panics(t *testing.T) {
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender: notifiers.GetNotifierService(),
	})

	ctx := t.Context()

	go supervisor.Run(ctx)

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if supervisor.IsRunning() {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	require.True(t, supervisor.IsRunning(), "supervisor must report running before the double-run check")

	require.Panics(t, func() { supervisor.Run(ctx) },
		"a second Run() on the same instance must panic")
}

func Test_StopStreamer_WhenOwnedStreamerStops_MarksRowFailed(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender: notifiers.GetNotifierService(),
	})
	require.NoError(t, physical_repositories.GetWalStreamerRepository().Claim(prereqs.DB.ID))

	done := make(chan struct{})
	close(done)

	supervisor.running[prereqs.DB.ID] = &runningStreamer{
		cancel: func() {},
		done:   done,
	}

	supervisor.stopStreamer(prereqs.DB.ID, false)

	streamer, err := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	require.Equal(t, physical_enums.PhysicalWalStreamerStatusFailed, streamer.Status)
	require.Nil(t, streamer.FailingSince, "a deliberate stop is not a failure, so a deploy does not read as one")
}

func Test_RecoverStreamersOnStartup_WhenRunningRowStale_MarksFailed(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	seedStreamerRow(
		t,
		prereqs.DB.ID,
		time.Now().UTC().Add(-2*wal_health.StreamerHeartbeatStaleness),
		physical_enums.PhysicalWalStreamerStatusRunning,
	)

	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender: notifiers.GetNotifierService(),
	})

	require.NoError(t, supervisor.recoverStreamersOnStartup())

	streamer, err := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	require.Equal(t, physical_enums.PhysicalWalStreamerStatusFailed, streamer.Status)
}

func Test_RemoveWatchDirIfRequested_WhenCleanupRequested_RemovesQueue(t *testing.T) {
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender: notifiers.GetNotifierService(),
	})
	watchDir := filepath.Join(t.TempDir(), "wal-queue")
	require.NoError(t, os.MkdirAll(watchDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(watchDir, "segment"), []byte("wal"), 0o600))

	streamer := &runningStreamer{watchDir: watchDir}
	streamer.shouldRemoveWatchDir.Store(true)

	supervisor.removeWatchDirIfRequested(supervisor.logger, streamer)

	require.NoDirExists(t, watchDir)
}

func Test_NotifyChainBroken_WhenSameIncidentRepeats_SendsOneNotificationPerWindow(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	sender := &recordingNotificationSender{}
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{NotificationSender: sender})

	prereqs.Config.SendNotificationsOn = []backups_config_physical.BackupNotificationType{
		backups_config_physical.NotificationChainBroken,
	}

	// A FAILED streamer is reclaimable on the next tick, so the same incident
	// re-enters this path every cycle until the cause is fixed.
	for range 3 {
		supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, chainAlert{
			Kind:    chainAlertStreamerFailed,
			Heading: "WAL streaming stopped",
			Message: "database_id=x",
		})
	}

	require.Len(t, sender.sentNotifications, 1, "a reclaim loop must not turn into a notification loop")
	require.Equal(t, "WAL streaming stopped", sender.sentNotifications[0].Notification.Heading)
}

func Test_NotifyChainBroken_WhenConfigDoesNotOptIn_SendsNoNotification(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	sender := &recordingNotificationSender{}
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{NotificationSender: sender})

	prereqs.Config.SendNotificationsOn = []backups_config_physical.BackupNotificationType{
		backups_config_physical.NotificationBackupSuccess,
	}

	supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, chainAlert{
		Kind:    chainAlertStreamerFailed,
		Heading: "WAL streaming stopped",
		Message: "database_id=x",
	})

	require.Empty(t, sender.sentNotifications)
}

func Test_NotifyChainBroken_WhenDifferentIncidentKinds_SendsEachOnce(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	sender := &recordingNotificationSender{}
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{NotificationSender: sender})

	prereqs.Config.SendNotificationsOn = []backups_config_physical.BackupNotificationType{
		backups_config_physical.NotificationChainBroken,
	}

	supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, chainAlert{
		Kind:    chainAlertChainAtRisk,
		Heading: "chain at risk",
		Message: "database_id=x",
	})
	supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, chainAlert{
		Kind:    chainAlertSlotRebuilt,
		Heading: "chain rebuilt",
		Message: "database_id=x",
	})

	require.Len(t, sender.sentNotifications, 2,
		"a retention warning must not swallow the rebuild alert that reports the new chain anchor")
	require.Equal(t, "chain at risk", sender.sentNotifications[0].Notification.Heading)
	require.Equal(t, "chain rebuilt", sender.sentNotifications[1].Notification.Heading)
}

func Test_NotifyChainBroken_WhenThrottleWindowElapsed_SendsAgain(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	sender := &recordingNotificationSender{}
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender:            sender,
		ChainAlertMinIntervalOverride: time.Millisecond,
	})

	prereqs.Config.SendNotificationsOn = []backups_config_physical.BackupNotificationType{
		backups_config_physical.NotificationChainBroken,
	}

	streamerFailed := chainAlert{
		Kind:    chainAlertStreamerFailed,
		Heading: "WAL streaming stopped",
		Message: "database_id=x",
	}

	supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, streamerFailed)

	time.Sleep(5 * time.Millisecond)

	supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, streamerFailed)

	require.Len(t, sender.sentNotifications, 2,
		"the throttle must mute a burst, not the incident itself — an unfixed break has to keep reminding the operator")
}

func Test_StopStreamer_WhenStreamerStops_ForgetsChainAlertThrottle(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	sender := &recordingNotificationSender{}
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{NotificationSender: sender})

	prereqs.Config.SendNotificationsOn = []backups_config_physical.BackupNotificationType{
		backups_config_physical.NotificationChainBroken,
	}

	streamerFailed := chainAlert{
		Kind:    chainAlertStreamerFailed,
		Heading: "WAL streaming stopped",
		Message: "database_id=x",
	}

	supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, streamerFailed)

	done := make(chan struct{})
	close(done)

	supervisor.running[prereqs.DB.ID] = &runningStreamer{cancel: func() {}, done: done}
	supervisor.stopStreamer(prereqs.DB.ID, false)

	supervisor.notifyChainBroken(t.Context(), prereqs.DB, prereqs.Config, streamerFailed)

	require.Len(t, sender.sentNotifications, 2,
		"a reclaimed database is a fresh incident, so the throttle from the previous streamer must not mute it")
}

func Test_ChainRiskNotifier_WhenReasonsDiffer_UsesSeparateThrottleKinds(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	sender := &recordingNotificationSender{}
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{NotificationSender: sender})

	prereqs.Config.SendNotificationsOn = []backups_config_physical.BackupNotificationType{
		backups_config_physical.NotificationChainBroken,
	}

	notifyChainRisk := supervisor.chainRiskNotifier(t.Context(), prereqs.DB, prereqs.Config)

	lastArchivedWalAt := time.Now().UTC().Add(-2 * time.Hour)

	notifyChainRisk(postgresql_executor.ChainRiskReport{
		Reason:        postgresql_executor.ChainRiskReasonSlotRetention,
		SlotWalStatus: "extended",
		LagBytes:      64 * 1024 * 1024,
	})
	notifyChainRisk(postgresql_executor.ChainRiskReport{
		Reason:            postgresql_executor.ChainRiskReasonArchiveStale,
		LagBytes:          16 * 1024 * 1024,
		LastArchivedWalAt: &lastArchivedWalAt,
	})
	notifyChainRisk(postgresql_executor.ChainRiskReport{
		Reason: postgresql_executor.ChainRiskReasonRotationDenied,
	})

	require.Len(t, sender.sentNotifications, 3,
		"a slot-retention warning must not mute the alerts about a stalled recovery point or a refused rotation")

	require.Contains(t, sender.sentNotifications[1].Notification.Message, lastArchivedWalAt.Format(time.RFC3339))
	require.Contains(t, sender.sentNotifications[2].Notification.Message, "pg_switch_wal()")
}

func seedBackupPrereqsOnSharedSource(t *testing.T, backupType postgresql_physical.BackupType) *backupPrereqs {
	t.Helper()

	port, err := strconv.Atoi(config.GetEnv().TestPhysicalPostgres17Port)
	require.NoError(t, err)

	return seedBackupPrereqsOnEndpoint(t, config.GetEnv().TestLocalhost, port, backupType)
}

func seedBackupPrereqsOnEndpoint(
	t *testing.T,
	host string,
	port int,
	backupType postgresql_physical.BackupType,
) *backupPrereqs {
	t.Helper()

	return seedBackupPrereqsWithDatabase(
		t,
		func(workspaceID uuid.UUID, notifier *notifiers.Notifier) *databases.Database {
			return databases.CreateTestPhysicalPostgresDatabaseWithType(
				host,
				port,
				workspaceID,
				notifier,
				"17",
				backupType,
			)
		},
	)
}

// A supervisor that stops its streamers before the test drops their slots, so
// no receiver re-attaches in between.
func startReconcilingSupervisor(t *testing.T) (*PhysicalWalStreamSupervisor, context.Context) {
	t.Helper()

	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender: notifiers.GetNotifierService(),
	})

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(func() {
		cancel()
		supervisor.stopAllStreamers()
	})

	return supervisor, ctx
}

func dropStreamerSlotOnCleanup(t *testing.T, prereqs *backupPrereqs) {
	t.Helper()

	adminConn := postgresql_executor.OpenAdminConn(t, &postgresql_executor.PhysicalDBFixture{DB: prereqs.DB})

	t.Cleanup(func() {
		postgresql_executor.DropReplicationSlotExternally(
			t,
			adminConn,
			prereqs.DB.PostgresqlPhysical.ReplicationSlotName,
		)
	})
}

func countSamples(t *testing.T, databaseID uuid.UUID) int {
	t.Helper()

	samples, err := physical_repositories.GetWalStreamSampleRepository().FindSinceByDatabase(databaseID, time.Time{})
	require.NoError(t, err)

	return len(samples)
}

func Test_Reconcile_WhenDatabaseIsNotWalStreaming_RecordsNoSamplesForIt(t *testing.T) {
	streamingPrereqs := seedBackupPrereqsOnSharedSource(t, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	dropStreamerSlotOnCleanup(t, streamingPrereqs)

	incrementalPrereqs := seedBackupPrereqsOnSharedSource(t, postgresql_physical.BackupTypeFullAndIncremental)

	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(streamingPrereqs.DB.ID)
	})

	supervisor, ctx := startReconcilingSupervisor(t)
	supervisor.reconcile(ctx)

	require.Eventually(t, func() bool {
		return countSamples(t, streamingPrereqs.DB.ID) > 0
	}, 2*wal_health.SampleInterval+30*time.Second, time.Second,
		"the WAL streaming database records its first sample within one lag-monitor tick")

	require.Zero(t, countSamples(t, incrementalPrereqs.DB.ID),
		"a database outside WAL streaming mode is never sampled")
}

func Test_Reconcile_WhenEverySessionFailsBeforeFirstSample_KeepsFirstFailingTime(t *testing.T) {
	unreachablePrereqs := seedBackupPrereqsOnEndpoint(t, "127.0.0.1", 1,
		postgresql_physical.BackupTypeFullIncrementalAndWalStream)

	streamerRepository := physical_repositories.GetWalStreamerRepository()
	t.Cleanup(func() {
		_ = streamerRepository.DeleteByDatabaseID(unreachablePrereqs.DB.ID)
	})

	supervisor, ctx := startReconcilingSupervisor(t)

	waitForFailedSession := func(previousStartedAt time.Time) *physical_models.PhysicalWalStreamer {
		var failedStreamer *physical_models.PhysicalWalStreamer

		require.Eventually(t, func() bool {
			supervisor.mu.Lock()
			_, isRunning := supervisor.running[unreachablePrereqs.DB.ID]
			supervisor.mu.Unlock()

			streamer, err := streamerRepository.FindByDatabaseID(unreachablePrereqs.DB.ID)
			if err != nil || streamer == nil || isRunning {
				return false
			}

			failedStreamer = streamer

			return streamer.Status == physical_enums.PhysicalWalStreamerStatusFailed &&
				streamer.StartedAt.After(previousStartedAt)
		}, 60*time.Second, 200*time.Millisecond, "a session against an unreachable source fails at start")

		return failedStreamer
	}

	supervisor.reconcile(ctx)
	firstFailure := waitForFailedSession(time.Time{})
	require.NotNil(t, firstFailure.FailingSince)

	supervisor.reconcile(ctx)
	secondFailure := waitForFailedSession(firstFailure.StartedAt)

	require.NotNil(t, secondFailure.FailingSince)
	require.True(t, firstFailure.FailingSince.Equal(*secondFailure.FailingSince),
		"a crash loop keeps the time of its first failure")
	require.Zero(t, countSamples(t, unreachablePrereqs.DB.ID))
}

func Test_PhysicalWalStreamSupervisorRun_WhenContextCancelledWithRunningStreamer_UnregistersStreamer(
	t *testing.T,
) {
	prereqs := seedBackupPrereqsOnSharedSource(t, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	dropStreamerSlotOnCleanup(t, prereqs)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(prereqs.DB.ID)
	})

	registry := tasks_cancellation.NewRegistry()
	supervisor := CreateTestWalStreamSupervisor(WalStreamSupervisorTestSpec{
		NotificationSender:       notifiers.GetNotifierService(),
		TaskCancellationRegistry: registry,
	})

	supervisorCtx, cancelSupervisor := context.WithCancel(t.Context())
	supervisorStopped := make(chan struct{})

	go func() {
		defer close(supervisorStopped)

		supervisor.Run(supervisorCtx)
	}()

	t.Cleanup(func() {
		cancelSupervisor()
		<-supervisorStopped
	})

	require.Eventually(t, func() bool {
		supervisor.mu.Lock()
		defer supervisor.mu.Unlock()

		_, isRunning := supervisor.running[prereqs.DB.ID]

		return isRunning
	}, 30*time.Second, 100*time.Millisecond, "the supervisor starts a streamer for the WAL streaming database")

	cancelSupervisor()

	waitCtx, cancelWait := context.WithTimeout(t.Context(), 6*time.Second)
	defer cancelWait()

	require.NoError(t, registry.WaitUntilNoTasksRegistered(waitCtx),
		"a cancelled supervisor unregisters its streamers before the shutdown deadline")
}

func Test_Reconcile_AfterBackupsStorageChanged_StartsStreamerOnNewStorage(t *testing.T) {
	tasks_cancellation.SetupDependencies()

	prereqs := seedBackupPrereqsOnSharedSource(t, postgresql_physical.BackupTypeFullIncrementalAndWalStream)
	dropStreamerSlotOnCleanup(t, prereqs)

	backupConfigService := backups_config_physical.GetBackupConfigService()
	newStorage := storages.CreateTestStorage(prereqs.Workspace.ID)

	t.Cleanup(func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, prereqs.DB.ID)
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(prereqs.DB.ID)

		restoredConfig := *prereqs.Config
		restoredConfig.StorageID = &prereqs.Storage.ID
		restoredConfig.Storage = prereqs.Storage
		_, err := backupConfigService.SaveBackupConfig(context.Background(), &restoredConfig)
		require.NoError(t, err)

		storages.RemoveTestStorage(context.Background(), newStorage.ID)
	})

	supervisor, ctx := startReconcilingSupervisor(t)

	supervisor.reconcile(ctx)
	waitForStreamerOnStorage(t, supervisor, prereqs.DB.ID, prereqs.Storage.ID)

	movedConfig := *prereqs.Config
	movedConfig.StorageID = &newStorage.ID
	movedConfig.Storage = newStorage
	_, err := backupConfigService.SaveBackupConfig(t.Context(), &movedConfig)
	require.NoError(t, err)

	newTestCancellationListener().OnBackupsStorageChanged(t.Context(), prereqs.DB.ID)

	require.Eventually(t, func() bool {
		supervisor.mu.Lock()
		defer supervisor.mu.Unlock()

		_, isRunning := supervisor.running[prereqs.DB.ID]

		return !isRunning
	}, 30*time.Second, 100*time.Millisecond, "the storage change stops the streamer on the old storage")

	supervisor.reconcile(ctx)
	waitForStreamerOnStorage(t, supervisor, prereqs.DB.ID, newStorage.ID)
}

func waitForStreamerOnStorage(
	t *testing.T,
	supervisor *PhysicalWalStreamSupervisor,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		supervisor.mu.Lock()
		defer supervisor.mu.Unlock()

		streamer, isRunning := supervisor.running[databaseID]

		return isRunning && streamer.storageID == storageID
	}, 30*time.Second, 100*time.Millisecond, "a streamer runs for the database on the expected storage")
}
