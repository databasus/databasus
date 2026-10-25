package backuping_physical

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/lib/crypto"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

// Ownership and restart recovery run through the physical_wal_streamers
// heartbeat table: every tick CAS-claims the databases that are unclaimed /
// FAILED / stale, so exactly one process streams a given database at a time.
type PhysicalWalStreamSupervisor struct {
	databaseService          *databases.DatabaseService
	backupConfigService      *backups_config_physical.BackupConfigService
	storageService           *storages.StorageService
	fileStore                *storage_files.Store
	walSegmentRepo           *physical_repositories.PhysicalWalSegmentRepository
	historyRepo              *physical_repositories.PhysicalWalHistoryRepository
	walStreamerRepo          *physical_repositories.PhysicalWalStreamerRepository
	walStreamSampleRepo      *physical_repositories.PhysicalWalStreamSampleRepository
	notificationSender       NotificationSender
	taskCancellationRegistry *tasks_cancellation.Registry
	masterKey                *crypto.MasterKey
	fieldEncryptor           util_encryption.FieldEncryptor
	logger                   *slog.Logger

	chainAlertMinInterval time.Duration

	mu      sync.Mutex
	running map[uuid.UUID]*runningStreamer

	lastTickTime atomicTime

	// A FAILED streamer is reclaimable on the next tick, so an unfixable break
	// re-enters startStreamer every ~45 s. Without this the operator would get one
	// notification per cycle instead of one per incident.
	lastChainAlertAt map[chainAlertKey]time.Time

	hasRun  atomic.Bool
	isReady atomic.Bool
}

type runningStreamer struct {
	cancel               context.CancelFunc
	done                 chan struct{}
	watchDir             string
	storageID            uuid.UUID
	shouldRemoveWatchDir atomic.Bool

	// Read by GetLiveState; the HTTP handlers and the streamers share one
	// process, so the running streamer's memory is the live source.
	walStreamSupervisor *postgresql_executor.WalStreamSupervisor
}

// GetLiveState returns the current values of the database's running WAL
// stream, or false while no streamer runs for it in this process.
func (s *PhysicalWalStreamSupervisor) GetLiveState(
	databaseID uuid.UUID,
	now time.Time,
) (*postgresql_executor.WalStreamLiveState, bool) {
	s.mu.Lock()
	streamer, isRunning := s.running[databaseID]
	s.mu.Unlock()

	if !isRunning || streamer.walStreamSupervisor == nil {
		return nil, false
	}

	liveState := streamer.walStreamSupervisor.GetLiveState(now)

	return &liveState, true
}

func (s *PhysicalWalStreamSupervisor) IsRunning() bool {
	return s.isReady.Load()
}

func (s *PhysicalWalStreamSupervisor) IsSupervisorHealthy() bool {
	return s.lastTickTime.Load().After(time.Now().UTC().Add(-schedulerHealthcheckThreshold))
}

func (s *PhysicalWalStreamSupervisor) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	s.logger = s.logger.With("job_id", uuid.New(), "job_name", walStreamSupervisorJobName)

	s.lastTickTime.Store(time.Now().UTC())

	if err := s.recoverStreamersOnStartup(); err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to recover wal streamers on startup",
			"error",
			err,
		)

		panic(err)
	}

	s.isReady.Store(true)

	defer func() {
		s.isReady.Store(false)
		s.stopAllStreamers()
	}()

	ticker := time.NewTicker(walStreamSupervisorTickInterval)
	defer ticker.Stop()

	s.reconcile(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.reconcile(ctx)

			s.lastTickTime.Store(time.Now().UTC())
		}
	}
}

func (s *PhysicalWalStreamSupervisor) recoverStreamersOnStartup() error {
	failed, err := s.walStreamerRepo.MarkStaleRunningFailed(int(wal_health.StreamerHeartbeatStaleness.Seconds()))
	if err != nil {
		return err
	}

	if failed > 0 {
		s.logger.Warn(
			fmt.Sprintf("recovered %d stale wal streamers on startup", failed),
		)
	}

	return nil
}

func (s *PhysicalWalStreamSupervisor) reconcile(ctx context.Context) {
	configs, err := s.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to load enabled configs",
			"error",
			err,
		)

		return
	}

	candidates := make(map[uuid.UUID]bool)

	for _, backupConfig := range configs {
		if !backupConfig.IsWalStreamingExpected() {
			continue
		}

		logger := s.logger.With("database_id", backupConfig.DatabaseID)

		candidates[backupConfig.DatabaseID] = true

		s.ensureStreamerRunning(ctx, logger, backupConfig)
	}

	s.logger.DebugContext(ctx, fmt.Sprintf("reconciled wal streamers: %d candidates of %d enabled configs",
		len(candidates), len(configs)))

	s.stopNonCandidates(candidates)
	s.heartbeatOwnedStreamers()
}

func (s *PhysicalWalStreamSupervisor) ensureStreamerRunning(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) {
	s.mu.Lock()
	_, alreadyRunning := s.running[backupConfig.DatabaseID]
	s.mu.Unlock()

	if alreadyRunning {
		return
	}

	claimed, err := s.walStreamerRepo.ClaimIfClaimable(
		backupConfig.DatabaseID, int(wal_health.StreamerHeartbeatStaleness.Seconds()),
	)
	if err != nil {
		logger.ErrorContext(ctx, "claim failed", "error", err)

		return
	}

	if !claimed {
		return
	}

	s.startStreamer(ctx, logger, backupConfig)
}

func (s *PhysicalWalStreamSupervisor) startStreamer(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) {
	db, err := s.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil || db.PostgresqlPhysical == nil {
		logger.ErrorContext(ctx, "failed to load database for streamer", "error", err)
		s.releaseClaim(logger, backupConfig.DatabaseID)

		return
	}

	storage, err := s.storageService.GetStorageByID(ctx, *backupConfig.StorageID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load storage for streamer", "error", err)
		s.releaseClaim(logger, backupConfig.DatabaseID)

		return
	}

	masterKey, ok := s.resolveMasterKey(logger, backupConfig)
	if !ok {
		s.releaseClaim(logger, backupConfig.DatabaseID)

		return
	}

	// One SSH session for the whole streamer: pg_receivewal plus the slot-LSN, lag and rotation
	// loops would otherwise handshake with the bastion several times a minute, forever.
	tunnelCtx, cancelTunnelOpen := context.WithTimeout(context.Background(), bastionOpenTimeout)
	defer cancelTunnelOpen()

	tunneledDatabase, err := postgresql_physical.OpenTunnel(tunnelCtx, postgresql_physical.OpenTunnelSpec{
		Database:  db.PostgresqlPhysical,
		Logger:    logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to open ssh tunnel to the source cluster", "error", err)
		s.releaseClaim(logger, backupConfig.DatabaseID)

		return
	}

	// Derive from the supervisor's run ctx so a process shutdown cancels every
	// streamer; the per-DB cancel (registered with the task cancellation registry and stored on
	// runningStreamer) handles targeted teardown on disable / demote / db-remove.
	streamerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	streamer := &runningStreamer{
		cancel:    cancel,
		done:      done,
		watchDir:  filepath.Join(config.GetEnv().DataFolder, "wal-queue", db.ID.String()),
		storageID: storage.ID,
	}

	s.taskCancellationRegistry.RegisterTask(db.ID, func() {
		streamer.shouldRemoveWatchDir.Store(true)
		cancel()
	})

	supervisor := postgresql_executor.NewWalStreamSupervisor(postgresql_executor.WalStreamSpec{
		DatabaseID:                db.ID,
		SourceDB:                  tunneledDatabase.GetDatabaseThroughTunnel(),
		IsBastionReachable:        tunneledDatabase.IsBastionReachable,
		StorageID:                 storage.ID,
		FileStore:                 s.fileStore,
		Encryption:                backupConfig.Encryption,
		MasterKey:                 masterKey,
		FieldEncryptor:            s.fieldEncryptor,
		WalSegmentRepo:            s.walSegmentRepo,
		HistoryRepo:               s.historyRepo,
		WalStreamSampleRepo:       s.walStreamSampleRepo,
		WalStreamerRepo:           s.walStreamerRepo,
		WatchDirRoot:              config.GetEnv().DataFolder,
		WalLagThresholdBytes:      backupConfig.WalLagThresholdBytes,
		ForcedRotationInterval:    postgresql_executor.DefaultForcedRotationInterval,
		ArchiveStalenessThreshold: wal_health.DefaultArchiveStalenessThreshold,
		OnGapDetected:             s.gapNotifier(ctx, db, backupConfig),
		OnSlotRebuilt:             s.slotRebuildFullRequester(ctx, logger, db, backupConfig),
		OnChainAtRisk:             s.chainRiskNotifier(ctx, db, backupConfig),
		Logger:                    s.logger,
	})

	streamer.walStreamSupervisor = supervisor

	s.mu.Lock()
	s.running[db.ID] = streamer
	s.mu.Unlock()

	logger.InfoContext(ctx, "wal stream supervisor started for database", "storage_id", streamer.storageID)

	go func() {
		defer close(done)
		defer tunneledDatabase.Close()
		defer s.taskCancellationRegistry.UnregisterTask(db.ID)
		defer s.removeWatchDirIfRequested(logger, streamer)

		if err := supervisor.Run(streamerCtx); err != nil {
			logger.ErrorContext(ctx, "wal stream supervisor exited with error", "error", err)

			s.notifyChainBroken(ctx, db, backupConfig, chainAlert{
				Kind:    chainAlertStreamerFailed,
				Heading: fmt.Sprintf("WAL streaming stopped for %q", db.Name),
				Message: fmt.Sprintf("database_id=%s error=%s", db.ID, err),
			})

			// Mark FAILED so a later tick can reclaim it. A clean
			// ctx-cancel (shutdown / lifecycle stop) does not reach here with an
			// error, so the row is left for the cancelling path to handle.
			if markErr := s.walStreamerRepo.MarkFailed(db.ID); markErr != nil {
				logger.ErrorContext(ctx, "failed to mark streamer failed", "error", markErr)
			}
		}

		s.mu.Lock()
		delete(s.running, db.ID)
		s.mu.Unlock()
	}()
}

func (s *PhysicalWalStreamSupervisor) resolveMasterKey(
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) (string, bool) {
	if backupConfig.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		return "", true
	}

	key := s.masterKey.GetSecret()

	return key, true
}

func (s *PhysicalWalStreamSupervisor) gapNotifier(
	ctx context.Context,
	db *databases.Database,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) func(gapStart, gapEnd walmath.LSN) {
	return func(gapStart, gapEnd walmath.LSN) {
		if !slices.Contains(backupConfig.SendNotificationsOn, backups_config_physical.NotificationWalGap) {
			return
		}

		notification := notifier_models.Notification{
			Type:    notifier_models.NotificationTypeBackupFailed,
			Heading: fmt.Sprintf("Physical WAL gap detected for %q", db.Name),
			Message: fmt.Sprintf("database_id=%s gap=[%s, %s)", db.ID, gapStart.String(), gapEnd.String()),
		}

		for _, notifier := range db.Notifiers {
			s.notificationSender.SendNotification(ctx, &notifier, notification)
		}
	}
}

func (s *PhysicalWalStreamSupervisor) slotRebuildFullRequester(
	ctx context.Context,
	logger *slog.Logger,
	db *databases.Database,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) func(context.Context, string) error {
	return func(_ context.Context, reason string) error {
		if err := s.backupConfigService.RequestFullBackupNow(backupConfig.DatabaseID); err != nil {
			return err
		}

		logger.WarnContext(ctx, "requested out-of-cadence full backup after wal slot rebuild", "reason", reason)

		// Dropping and recreating the slot always leaves a WAL gap, so the chain
		// this notifies about is already broken by the time we get here.
		s.notifyChainBroken(ctx, db, backupConfig, chainAlert{
			Kind:    chainAlertSlotRebuilt,
			Heading: fmt.Sprintf("Physical WAL chain rebuilt for %q", db.Name),
			Message: fmt.Sprintf("database_id=%s reason=%s; a fresh full backup was requested to anchor the new chain",
				db.ID, reason),
		})

		return nil
	}
}

func (s *PhysicalWalStreamSupervisor) chainRiskNotifier(
	ctx context.Context,
	db *databases.Database,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) func(postgresql_executor.ChainRiskReport) {
	return func(report postgresql_executor.ChainRiskReport) {
		s.notifyChainBroken(ctx, db, backupConfig, buildChainRiskAlert(db, report))
	}
}

// Each risk gets its own kind: the throttle key is (database, kind), so folding
// them together would let a slot-retention warning mute the alert that says the
// recovery point has stopped advancing.
func buildChainRiskAlert(db *databases.Database, report postgresql_executor.ChainRiskReport) chainAlert {
	switch report.Reason {
	case postgresql_executor.ChainRiskReasonArchiveStale:
		lastArchivedAt := "never"
		if report.LastArchivedWalAt != nil {
			lastArchivedAt = report.LastArchivedWalAt.Format(time.RFC3339)
		}

		return chainAlert{
			Kind:    chainAlertArchiveStale,
			Heading: fmt.Sprintf("Physical WAL archiving fell behind for %q", db.Name),
			Message: fmt.Sprintf(
				"database_id=%s reason=%s last_archived_wal_at=%s lag_bytes=%d; "+
					"the recovery point stopped advancing while the source kept writing",
				db.ID, report.Reason, lastArchivedAt, report.LagBytes,
			),
		}

	case postgresql_executor.ChainRiskReasonRotationDenied:
		return chainAlert{
			Kind:    chainAlertRotationDenied,
			Heading: fmt.Sprintf("Cannot force WAL rotation for %q", db.Name),
			Message: fmt.Sprintf(
				"database_id=%s reason=%s; a rarely-written database keeps its newest WAL local until the segment "+
					"fills. Either GRANT EXECUTE ON FUNCTION pg_switch_wal() TO the backup role, "+
					"or set archive_timeout on the source",
				db.ID, report.Reason,
			),
		}

	default:
		return chainAlert{
			Kind:    chainAlertChainAtRisk,
			Heading: fmt.Sprintf("Physical WAL chain at risk for %q", db.Name),
			Message: fmt.Sprintf("database_id=%s reason=%s slot_wal_status=%s lag_bytes=%d",
				db.ID, report.Reason, report.SlotWalStatus, report.LagBytes),
		}
	}
}

func (s *PhysicalWalStreamSupervisor) notifyChainBroken(
	ctx context.Context,
	db *databases.Database,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
	alert chainAlert,
) {
	if !slices.Contains(backupConfig.SendNotificationsOn, backups_config_physical.NotificationChainBroken) {
		return
	}

	if !s.recordChainAlertIfDue(chainAlertKey{DatabaseID: db.ID, Kind: alert.Kind}) {
		return
	}

	notification := notifier_models.Notification{
		Type:    notifier_models.NotificationTypeBackupFailed,
		Heading: alert.Heading,
		Message: alert.Message,
	}

	for _, notifier := range db.Notifiers {
		s.notificationSender.SendNotification(ctx, &notifier, notification)
	}
}

func (s *PhysicalWalStreamSupervisor) recordChainAlertIfDue(key chainAlertKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	if sentAt, wasNotified := s.lastChainAlertAt[key]; wasNotified &&
		now.Sub(sentAt) < s.chainAlertMinInterval {
		return false
	}

	s.lastChainAlertAt[key] = now

	return true
}

func (s *PhysicalWalStreamSupervisor) stopNonCandidates(candidates map[uuid.UUID]bool) {
	s.mu.Lock()
	var toStop []uuid.UUID

	for databaseID := range s.running {
		if !candidates[databaseID] {
			toStop = append(toStop, databaseID)
		}
	}
	s.mu.Unlock()

	for _, databaseID := range toStop {
		s.stopStreamer(databaseID, true)
	}
}

func (s *PhysicalWalStreamSupervisor) stopStreamer(databaseID uuid.UUID, shouldRemoveWatchDir bool) {
	s.mu.Lock()
	streamer := s.running[databaseID]
	s.mu.Unlock()

	if streamer == nil {
		return
	}

	if shouldRemoveWatchDir {
		streamer.shouldRemoveWatchDir.Store(true)
	}

	streamer.cancel()

	select {
	case <-streamer.done:
		s.logger.Info("wal stream supervisor stopped for database", "database_id", databaseID)
	case <-time.After(streamerStopTimeout):
		s.logger.Warn("streamer stop timed out", "database_id", databaseID)
	}

	s.mu.Lock()
	delete(s.running, databaseID)

	maps.DeleteFunc(s.lastChainAlertAt, func(key chainAlertKey, _ time.Time) bool {
		return key.DatabaseID == databaseID
	})

	s.mu.Unlock()

	if err := s.walStreamerRepo.MarkStopped(databaseID); err != nil {
		s.logger.Error(
			"failed to mark stopped streamer released",
			"database_id",
			databaseID,
			"error",
			err,
		)
	}
}

func (s *PhysicalWalStreamSupervisor) heartbeatOwnedStreamers() {
	s.mu.Lock()
	owned := make([]uuid.UUID, 0, len(s.running))
	for databaseID := range s.running {
		owned = append(owned, databaseID)
	}
	s.mu.Unlock()

	for _, databaseID := range owned {
		if err := s.walStreamerRepo.Heartbeat(databaseID); err != nil {
			s.logger.Error("heartbeat failed", "database_id", databaseID, "error", err)
		}
	}
}

func (s *PhysicalWalStreamSupervisor) stopAllStreamers() {
	s.mu.Lock()
	owned := make([]uuid.UUID, 0, len(s.running))
	for databaseID := range s.running {
		owned = append(owned, databaseID)
	}
	s.mu.Unlock()

	for _, databaseID := range owned {
		s.stopStreamer(databaseID, false)
	}
}

func (s *PhysicalWalStreamSupervisor) removeWatchDirIfRequested(logger *slog.Logger, streamer *runningStreamer) {
	if !streamer.shouldRemoveWatchDir.Load() {
		return
	}

	if err := os.RemoveAll(streamer.watchDir); err != nil {
		logger.Warn("failed to remove wal queue directory", "watch_dir", streamer.watchDir, "error", err)
	}
}

// Marking FAILED makes the slot immediately reclaimable rather than looking
// alive until the heartbeat goes stale.
func (s *PhysicalWalStreamSupervisor) releaseClaim(logger *slog.Logger, databaseID uuid.UUID) {
	if err := s.walStreamerRepo.MarkFailed(databaseID); err != nil {
		logger.Error("failed to release streamer claim", "error", err)
	}
}
