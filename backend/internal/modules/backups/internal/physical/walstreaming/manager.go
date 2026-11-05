package walstreaming

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
)

type Manager struct {
	databases                databases.Module
	configs                  *config.Service
	storages                 storages.Module
	walSegmentRepo           *catalog.WalSegmentRepository
	historyRepo              *catalog.WalHistoryRepository
	walStreamerRepo          *catalog.WalStreamerRepository
	walStreamSampleRepo      *catalog.WalStreamSampleRepository
	notificationSender       notifiers.Module
	taskCancellationRegistry *background.CancellationRegistry
	masterKey                *crypto.MasterKey
	logger                   *slog.Logger
	clock                    clock.Source
	transactor               *database.Transactor
	runs                     *background.Group
	appCtx                   context.Context
	clientToolsDir           string
	walQueueDir              string
	chainAlertMinInterval    time.Duration
	timings                  Timings
	mu                       sync.Mutex
	running                  map[uuid.UUID]*runningStreamer
	lastChainAlertAt         chainAlertThrottle
}
type runningStreamer struct {
	claimID              uuid.UUID
	storageGeneration    int64
	cancel               context.CancelFunc
	done                 chan struct{}
	watchDir             string
	storageID            uuid.UUID
	shouldRemoveWatchDir atomic.Bool

	walStreamer *Streamer
}

func (s *Manager) GetLiveState(ctx context.Context,
	databaseID uuid.UUID,
	now time.Time,
) (*WalStreamLiveState, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	streamer, isRunning := s.running[databaseID]
	s.mu.Unlock()

	if !isRunning || streamer.walStreamer == nil {
		return nil, false
	}

	liveState := streamer.walStreamer.GetLiveState(ctx, now)

	return &liveState, true
}

func (s *Manager) GetStreamerCancellation(databaseID uuid.UUID) context.CancelFunc {
	s.mu.Lock()
	defer s.mu.Unlock()
	streamer := s.running[databaseID]
	if streamer == nil {
		return nil
	}
	return streamer.cancel
}

func (s *Manager) Start(ctx context.Context) {
	s.appCtx = ctx
	s.mu.Lock()
	s.lastChainAlertAt = make(chainAlertThrottle)
	s.mu.Unlock()
}

func (s *Manager) GetJob() background.Job {
	return background.Job{
		Name:     "backups_physical_wal_stream_manager",
		Interval: 15 * time.Second,
		Run:      func(ctx context.Context, _ time.Time) error { s.reconcile(ctx); return nil },
	}
}

func (s *Manager) FailStaleStreamers(ctx context.Context) error {
	_, err := s.walStreamerRepo.MarkStaleRunningFailed(
		ctx,
		int(streamhealth.StreamerHeartbeatStaleness.Seconds()),
		s.clock.Now(),
	)
	return err
}

func (s *Manager) reconcile(ctx context.Context) {
	configs, err := s.configs.ListEnabledPhysicalBackupConfigs(ctx)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to load enabled configs", slog.Any("error",
				err),
		)

		return
	}

	candidates := make(map[uuid.UUID]bool)

	for _, backupConfig := range configs {
		if !backupConfig.IsWalStreamingExpected() {
			continue
		}

		logger := s.logger.With(slog.Any("database_id", backupConfig.DatabaseID))

		candidates[backupConfig.DatabaseID] = true

		s.ensureStreamerRunning(ctx, logger, backupConfig)
	}

	s.logger.DebugContext(
		ctx,
		"reconciled wal streamers",
		slog.Any("candidate_count", len(candidates)),
		slog.Any("enabled_config_count", len(configs)),
	)

	s.stopNonCandidates(ctx, candidates)
	s.heartbeatOwnedStreamers(ctx)
}

func (s *Manager) ensureStreamerRunning(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
) {
	s.mu.Lock()
	running, alreadyRunning := s.running[backupConfig.DatabaseID]
	s.mu.Unlock()

	if alreadyRunning {
		if running.storageGeneration != backupConfig.StorageGeneration || running.storageID != *backupConfig.StorageID {
			running.shouldRemoveWatchDir.Store(true)
			running.cancel()
		}
		return
	}

	claimID := uuid.New()
	var claimed bool
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.configs.CheckStoragePublication(
			ctx,
			config.StoragePublication{
				DatabaseID: backupConfig.DatabaseID,
				StorageID:  *backupConfig.StorageID,
				Generation: backupConfig.StorageGeneration,
			},
		); err != nil {
			return err
		}
		var err error
		claimed, err = s.walStreamerRepo.ClaimIfClaimable(
			ctx,
			catalog.StreamerClaim{
				DatabaseID:       backupConfig.DatabaseID,
				ClaimID:          claimID,
				StalenessSeconds: int(streamhealth.StreamerHeartbeatStaleness.Seconds()),
				Now:              s.clock.Now(),
			},
		)
		return err
	})
	if err != nil {
		logger.ErrorContext(ctx, "claim failed", slog.Any("error", err))

		return
	}

	if !claimed {
		return
	}

	s.startStreamer(ctx, logger, streamerStart{Config: backupConfig, ClaimID: claimID})
}

type streamerStart struct {
	Config  config.PhysicalBackupConfig
	ClaimID uuid.UUID
}

func (s *Manager) startStreamer(
	ctx context.Context,
	logger *slog.Logger,
	start streamerStart,
) {
	backupConfig := start.Config
	claimID := start.ClaimID
	db, err := s.getDatabase(ctx, backupConfig.DatabaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load database for streamer", slog.Any("error", err))
		s.releaseClaim(ctx, logger, claimID)

		return
	}

	connectionSettings, err := s.databases.GetPostgresqlPhysicalConnectionSettings(ctx, db.ID)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to read the connection settings of the physical database",
			slog.Any("error", err),
		)
		s.releaseClaim(ctx, logger, claimID)

		return
	}

	storage, err := getStorage(ctx, s.storages, *backupConfig.StorageID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load storage for streamer", slog.Any("error", err))
		s.releaseClaim(ctx, logger, claimID)

		return
	}

	masterKey, ok := s.resolveMasterKey(logger, backupConfig)
	if !ok {
		s.releaseClaim(ctx, logger, claimID)

		return
	}

	tunnelCtx, cancelTunnelOpen := context.WithTimeout(ctx, bastionOpenTimeout)
	defer cancelTunnelOpen()

	tunnel, err := pgphysical.NewSource(connectionSettings.Engine).OpenTunnel(tunnelCtx, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to open ssh tunnel to the source cluster", slog.Any("error", err))
		s.releaseClaim(ctx, logger, claimID)

		return
	}

	sourceThroughTunnel := connectionSettings
	sourceThroughTunnel.Engine = tunnel.GetSource().GetSettings()

	streamerCtx, cancel, finish := runcontext.New(s.appCtx, runcontext.Limits{Shutdown: runcontext.ErrBackupShutdown})
	streamerCtx = logging.With(streamerCtx, slog.Any("database_id", db.ID))
	done := make(chan struct{})
	streamer := &runningStreamer{
		claimID: claimID, storageGeneration: backupConfig.StorageGeneration,
		cancel:    cancel,
		done:      done,
		watchDir:  filepath.Join(s.walQueueDir, db.ID.String()),
		storageID: storage.ID,
	}

	s.taskCancellationRegistry.Register(db.ID, func() {
		streamer.shouldRemoveWatchDir.Store(true)
		cancel()
	})

	supervisor := NewStreamer(WalStreamSpec{
		ClaimID: claimID,
		Publication: config.PublicationFence{
			Configs: s.configs,
			Publication: config.StoragePublication{
				DatabaseID: backupConfig.DatabaseID,
				StorageID:  *backupConfig.StorageID,
				Generation: backupConfig.StorageGeneration,
			},
		},
		Clock:                     s.clock,
		Transactor:                s.transactor,
		ClientTools:               pgshared.ClientTools{Root: s.clientToolsDir},
		DatabaseID:                db.ID,
		SourceDB:                  sourceThroughTunnel,
		IsBastionReachable:        tunnel.IsBastionReachable,
		StorageID:                 storage.ID,
		FileStore:                 s.storages,
		Encryption:                catalog.BackupEncryption(backupConfig.Encryption),
		MasterKey:                 masterKey,
		BackupStatus:              s.databases,
		Databases:                 s.databases,
		WalSegmentRepo:            s.walSegmentRepo,
		HistoryRepo:               s.historyRepo,
		WalStreamSampleRepo:       s.walStreamSampleRepo,
		WalStreamerRepo:           s.walStreamerRepo,
		WatchDirRoot:              s.walQueueDir,
		WalLagThresholdBytes:      backupConfig.WalLagThresholdBytes,
		ForcedRotationInterval:    s.timings.ForcedRotationInterval,
		ArchiveStalenessThreshold: s.timings.ArchiveStalenessThreshold,
		LocalQueuePauseBytes:      s.timings.LocalQueuePauseBytes,
		OnGapDetected:             s.gapNotifier(ctx, db, backupConfig),
		OnSlotRebuilt:             s.slotRebuildFullRequester(ctx, logger, db, backupConfig),
		OnChainAtRisk:             s.chainRiskNotifier(ctx, db, backupConfig),
		Logger:                    s.logger,
	})

	streamer.walStreamer = supervisor

	s.mu.Lock()
	s.running[db.ID] = streamer
	s.mu.Unlock()

	logger.InfoContext(ctx, "wal streamer started for database", slog.Any("storage_id", streamer.storageID))

	s.runs.Go(streamerCtx, "physical_wal_streamer", func(ctx context.Context) error {
		defer finish()
		defer func() {
			s.mu.Lock()
			if s.running[db.ID] == streamer {
				s.taskCancellationRegistry.Unregister(db.ID)
				delete(s.running, db.ID)
			}
			s.mu.Unlock()
			close(done)
		}()
		defer tunnel.Close()
		defer s.removeWatchDirIfRequested(ctx, logger, streamer)

		runErr := background.RunRecovering(ctx, supervisor.Run)
		if err := runErr; err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "wal streamer exited with error", slog.Any("error", err))

			s.notifyChainBroken(ctx, db, backupConfig, chainAlert{
				Kind:    chainAlertStreamerFailed,
				Heading: fmt.Sprintf("WAL streaming stopped for %q", db.Name),
				Message: fmt.Sprintf("database_id=%s error=%s", db.ID, err),
			})

			if markErr := s.walStreamerRepo.MarkFailed(
				context.WithoutCancel(ctx),
				claimID,
				s.clock.Now(),
			); markErr != nil {
				logger.ErrorContext(ctx, "failed to mark streamer failed", slog.Any("error", markErr))
			}
		}

		if runErr == nil || ctx.Err() != nil {
			if err := s.walStreamerRepo.MarkStopped(context.WithoutCancel(ctx), claimID); err != nil {
				logger.ErrorContext(ctx, "failed to record stopped wal streamer", slog.Any("error", err))
			}
		}
		return nil
	})
}

func (s *Manager) resolveMasterKey(
	_ *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
) (*crypto.MasterKey, bool) {
	if backupConfig.Encryption != config.EncryptionEncrypted {
		return nil, true
	}

	return s.masterKey, true
}

func (s *Manager) gapNotifier(
	ctx context.Context,
	db databases.Database,
	backupConfig config.PhysicalBackupConfig,
) func(gapStart, gapEnd walmath.LSN) {
	return func(gapStart, gapEnd walmath.LSN) {
		if !slices.Contains(backupConfig.NotificationTriggers, config.NotificationTriggerWalGap) {
			return
		}

		notification := notifiers.Notification{
			Category: notifiers.CategoryBackupFailed,
			Heading:  fmt.Sprintf("Physical WAL gap detected for %q", db.Name),
			Message:  fmt.Sprintf("database_id=%s gap=[%s, %s)", db.ID, gapStart.String(), gapEnd.String()),
		}

		s.notificationSender.Notify(ctx, db.NotifierIDs, notification)
	}
}

func (s *Manager) slotRebuildFullRequester(
	ctx context.Context,
	logger *slog.Logger,
	db databases.Database,
	backupConfig config.PhysicalBackupConfig,
) func(context.Context, string) error {
	return func(_ context.Context, reason string) error {
		if err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
			if err := s.configs.CheckStoragePublication(
				ctx,
				config.StoragePublication{
					DatabaseID: backupConfig.DatabaseID,
					StorageID:  *backupConfig.StorageID,
					Generation: backupConfig.StorageGeneration,
				},
			); err != nil {
				return err
			}
			return s.configs.RequestFullBackup(ctx, backupConfig.DatabaseID)
		}); err != nil {
			return err
		}

		logger.WarnContext(
			ctx,
			"requested out-of-cadence full backup after wal slot rebuild",
			slog.Any("reason", reason),
		)

		s.notifyChainBroken(ctx, db, backupConfig, chainAlert{
			Kind:    chainAlertSlotRebuilt,
			Heading: fmt.Sprintf("Physical WAL chain rebuilt for %q", db.Name),
			Message: fmt.Sprintf("database_id=%s reason=%s; a fresh full backup was requested to anchor the new chain",
				db.ID, reason),
		})

		return nil
	}
}

func (s *Manager) chainRiskNotifier(
	ctx context.Context,
	db databases.Database,
	backupConfig config.PhysicalBackupConfig,
) func(ChainRiskReport) {
	return func(report ChainRiskReport) {
		s.notifyChainBroken(ctx, db, backupConfig, buildChainRiskAlert(db, report))
	}
}

func buildChainRiskAlert(db databases.Database, report ChainRiskReport) chainAlert {
	switch report.Reason {
	case ChainRiskReasonArchiveStale:
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

	case ChainRiskReasonRotationDenied:
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

func (s *Manager) notifyChainBroken(
	ctx context.Context,
	db databases.Database,
	backupConfig config.PhysicalBackupConfig,
	alert chainAlert,
) {
	if !slices.Contains(backupConfig.NotificationTriggers, config.NotificationTriggerChainBroken) {
		return
	}

	if !s.recordChainAlertIfDue(chainAlertKey{DatabaseID: db.ID, Kind: alert.Kind}) {
		return
	}

	notification := notifiers.Notification{
		Category: notifiers.CategoryBackupFailed,
		Heading:  alert.Heading,
		Message:  alert.Message,
	}

	s.notificationSender.Notify(ctx, db.NotifierIDs, notification)
}

func (s *Manager) recordChainAlertIfDue(key chainAlertKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lastChainAlertAt.RecordIfDue(chainAlertAttempt{
		Key: key, Now: s.clock.Now(), MinInterval: s.chainAlertMinInterval,
	})
}

func (s *Manager) stopNonCandidates(ctx context.Context, candidates map[uuid.UUID]bool) {
	s.mu.Lock()
	var toStop []uuid.UUID

	for databaseID := range s.running {
		if !candidates[databaseID] {
			toStop = append(toStop, databaseID)
		}
	}
	s.mu.Unlock()

	for _, databaseID := range toStop {
		s.stopStreamer(ctx, databaseID, true)
	}
}

func (s *Manager) stopStreamer(ctx context.Context, databaseID uuid.UUID, shouldRemoveWatchDir bool) {
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
		s.logger.InfoContext(ctx, "wal streamer stopped for database", slog.Any("database_id", databaseID))
	case <-time.After(streamerStopTimeout):
		s.logger.WarnContext(ctx, "streamer stop timed out", slog.Any("database_id", databaseID))
	}

	s.mu.Lock()
	s.lastChainAlertAt.ForgetDatabase(databaseID)

	s.mu.Unlock()
}

func (s *Manager) heartbeatOwnedStreamers(ctx context.Context) {
	s.mu.Lock()
	owned := make([]uuid.UUID, 0, len(s.running))
	for _, streamer := range s.running {
		owned = append(owned, streamer.claimID)
	}
	s.mu.Unlock()

	for _, claimID := range owned {
		if err := s.walStreamerRepo.Heartbeat(ctx, claimID, s.clock.Now()); err != nil {
			s.logger.ErrorContext(ctx, "heartbeat failed", slog.Any("claim_id", claimID), slog.Any("error", err))
		}
	}
}

func (s *Manager) removeWatchDirIfRequested(ctx context.Context, logger *slog.Logger, streamer *runningStreamer) {
	if !streamer.shouldRemoveWatchDir.Load() {
		return
	}

	if err := os.RemoveAll(streamer.watchDir); err != nil {
		logger.WarnContext(
			ctx,
			"failed to remove wal queue directory",
			slog.Any("watch_dir", streamer.watchDir),
			slog.Any("error", err),
		)
	}
}

func (s *Manager) releaseClaim(ctx context.Context, logger *slog.Logger, claimID uuid.UUID) {
	if err := s.walStreamerRepo.MarkFailed(ctx, claimID, s.clock.Now()); err != nil {
		logger.ErrorContext(ctx, "failed to release streamer claim", slog.Any("error", err))
	}
}

func (s *Manager) getDatabase(ctx context.Context, databaseID uuid.UUID) (databases.Database, error) {
	listedDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}
	if len(listedDatabases) == 0 {
		return databases.Database{}, databases.ErrDatabaseNotFound
	}
	return listedDatabases[0], nil
}

func getStorage(ctx context.Context, module storages.Module, storageID uuid.UUID) (storages.Storage, error) {
	listedStorages, err := module.GetStorages(ctx, []uuid.UUID{storageID})
	if err != nil {
		return storages.Storage{}, err
	}
	if len(listedStorages) == 0 {
		return storages.Storage{}, errStorageNotFound
	}
	return listedStorages[0], nil
}

const (
	bastionOpenTimeout           = 30 * time.Second
	streamerStopTimeout          = 30 * time.Second
	defaultChainAlertMinInterval = 15 * time.Minute
)
