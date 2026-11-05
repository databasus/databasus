package walstreaming

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

const (
	receivewalRespawnBackoff = 2 * time.Second

	receivewalRespawnMaxBackoff = 30 * time.Minute

	receivewalBastionDownMaxBackoff = 30 * time.Second

	receivewalMinHealthyUptime = 15 * time.Second

	receivewalMaxRapidFailures = 5

	receivewalMaxResumeMismatches = 2

	bastionProbeTimeout = 3 * time.Second

	pausePollInterval = 1 * time.Second
)

type resumeMismatchAction int

const (
	resumeMismatchActionRetry resumeMismatchAction = iota
	resumeMismatchActionRebuildSlot
)

type resumeMismatchEscalator struct {
	mismatchCount int
}

func (e *resumeMismatchEscalator) recordMismatchAndDecideEscalation() resumeMismatchAction {
	e.mismatchCount++

	if e.mismatchCount >= receivewalMaxResumeMismatches {
		e.mismatchCount = 0

		return resumeMismatchActionRebuildSlot
	}

	return resumeMismatchActionRetry
}

func (e *resumeMismatchEscalator) reset() {
	e.mismatchCount = 0
}

type rapidFailureEscalator struct {
	rapidFailureCount int
}

type retryableExitDecision struct {
	isEscalationRequired bool
	isBackoffResettable  bool
	maxRespawnBackoff    time.Duration
}

func (e *rapidFailureEscalator) recordExitAndDecideRetry(
	ranFor time.Duration,
	isBastionReachable bool,
) retryableExitDecision {
	if !isBastionReachable {
		return retryableExitDecision{maxRespawnBackoff: receivewalBastionDownMaxBackoff}
	}

	if ranFor >= receivewalMinHealthyUptime {
		e.rapidFailureCount = 0

		return retryableExitDecision{
			isBackoffResettable: true,
			maxRespawnBackoff:   receivewalRespawnMaxBackoff,
		}
	}

	e.rapidFailureCount++

	return retryableExitDecision{
		isEscalationRequired: e.rapidFailureCount >= receivewalMaxRapidFailures,
		maxRespawnBackoff:    receivewalRespawnMaxBackoff,
	}
}

func (e *rapidFailureEscalator) reset() {
	e.rapidFailureCount = 0
}

type WalStreamSpec struct {
	ClaimID      uuid.UUID
	Publication  config.PublicationFence
	Clock        clock.Source
	Transactor   *database.Transactor
	ClientTools  pgshared.ClientTools
	DatabaseID   uuid.UUID
	SourceDB     databases.PostgresqlPhysicalConnectionSettings
	StorageID    uuid.UUID
	FileStore    storages.Module
	BackupStatus BackupStatusRecorder

	Databases StreamedDatabases

	Encryption     catalog.BackupEncryption
	MasterKey      *crypto.MasterKey
	WalSegmentRepo *catalog.WalSegmentRepository
	HistoryRepo    *catalog.WalHistoryRepository

	WalStreamSampleRepo *catalog.WalStreamSampleRepository

	WalStreamerRepo *catalog.WalStreamerRepository

	WatchDirRoot string

	WalLagThresholdBytes int64

	ForcedRotationInterval time.Duration
	LocalQueuePauseBytes   int64

	ArchiveStalenessThreshold time.Duration

	OnGapDetected func(gapStart, gapEnd walmath.LSN)

	OnSlotRebuilt func(ctx context.Context, reason string) error

	OnChainAtRisk func(report ChainRiskReport)

	IsBastionReachable func(ctx context.Context) bool

	Logger *slog.Logger
}

type Streamer struct {
	spec     WalStreamSpec
	logger   *slog.Logger
	uploader *WalUploader
	watchDir string
	slotName string

	sessionStartedAt time.Time

	highWatermarkBytes atomic.Int64
	lowWatermarkBytes  atomic.Int64

	restartLatch receiverRestartLatch

	isPaused atomic.Bool

	receiverRunStartedAtUnixNano atomic.Int64

	isBackpressureStopPending atomic.Bool

	receiverRespawns receiverRespawnCounter

	isForcedRotationDenied atomic.Bool

	rebuildMu         sync.Mutex
	rebuildTimestamps []time.Time

	lowestQueuedSegmentMutex sync.Mutex
	lowestQueuedSegment      *walPosition

	latestProbe atomic.Pointer[timedSlotProbe]
}

func NewStreamer(spec WalStreamSpec) *Streamer {
	watchDir := filepath.Join(spec.WatchDirRoot, spec.DatabaseID.String())

	scopedLogger := spec.Logger.With(
		slog.Any("database_id", spec.DatabaseID),
		slog.Any("slot_name", spec.SourceDB.WalStreamSlotName),
	)

	uploader := NewWalUploader(WalUploadDeps{
		Publication: spec.Publication,
		Clock:       spec.Clock, Transactor: spec.Transactor,
		DatabaseID:          spec.DatabaseID,
		StorageID:           spec.StorageID,
		FileStore:           spec.FileStore,
		BackupStatus:        spec.BackupStatus,
		Encryption:          spec.Encryption,
		MasterKey:           spec.MasterKey,
		WalSegmentRepo:      spec.WalSegmentRepo,
		WalSegmentSizeBytes: walSegmentSizeBytes(spec.SourceDB),
		Logger:              scopedLogger,
		WatchDir:            watchDir,
		OnGapDetected:       spec.OnGapDetected,
	})

	supervisor := &Streamer{
		spec:             spec,
		logger:           scopedLogger,
		uploader:         uploader,
		watchDir:         watchDir,
		slotName:         spec.SourceDB.WalStreamSlotName,
		sessionStartedAt: spec.Clock.Now(),
	}

	pauseBytes := spec.LocalQueuePauseBytes
	if pauseBytes <= 0 {
		pauseBytes = getDefaultLocalQueuePauseBytes(spec.SourceDB)
	}
	supervisor.setLocalQueuePauseBytes(pauseBytes)

	return supervisor
}

func (s *Streamer) Run(ctx context.Context) error {
	logger := s.logger

	if err := os.MkdirAll(filepath.Join(s.watchDir, "archive_status"), 0o700); err != nil {
		return fmt.Errorf("create wal watch dir: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(s.watchDir, pendingUploadDirName), 0o700); err != nil {
		return fmt.Errorf("create pending upload dir: %w", err)
	}

	if err := s.createSlotUnlessDatabaseDeleted(ctx, logger); err != nil {
		if errors.Is(err, errStreamedDatabaseDeleted) {
			logger.InfoContext(ctx, "wal streamer not started because the database is deleted")

			return nil
		}

		return fmt.Errorf("verify persistent replication slot: %w", err)
	}

	s.removePartials(ctx, logger)
	s.recoverLocalSegmentsOnStartup(ctx, logger)

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	runs := background.New(background.Deps{Logger: logger})

	for _, loop := range []func(context.Context, *slog.Logger){
		s.runUploaderLoop,
		s.runBackpressureMonitor,
		s.runSlotLsnWatcher,
		s.runLagMonitor,
		s.runForcedWalRotation,
	} {
		runs.Go(runCtx, "wal_stream_loop", func(ctx context.Context) error { loop(ctx, logger); return nil })
	}

	fatalErr := s.runReceivewalSupervision(runCtx, logger)

	cancelRun()
	_ = runs.Wait(context.WithoutCancel(ctx))

	if errors.Is(fatalErr, errStreamedDatabaseDeleted) {
		logger.InfoContext(ctx, "wal streamer stopped because the database is deleted")

		return nil
	}

	if fatalErr != nil {
		logger.ErrorContext(ctx, "wal streamer stopping with fatal error", slog.Any("error", fatalErr))

		return fatalErr
	}

	logger.InfoContext(ctx, "wal streamer stopped")

	return nil
}

func (s *Streamer) runReceivewalSupervision(ctx context.Context, logger *slog.Logger) error {
	respawnBackoff := receivewalRespawnBackoff

	var (
		mismatchEscalator resumeMismatchEscalator
		failureEscalator  rapidFailureEscalator
	)

	hasSpawnedReceiver := false

	for {
		if ctx.Err() != nil {
			return nil
		}

		if !s.waitWhilePaused(ctx) {
			return nil
		}

		hasWaitedForBacklogDrain, isRunning := s.waitForBacklogBelowLow(ctx, logger)
		if !isRunning {
			return nil
		}

		s.clearRestartRequest()
		s.removePartials(ctx, logger)
		s.realignResumePath(ctx, logger)

		isBackpressureStop := s.isBackpressureStopPending.Swap(false)

		if hasSpawnedReceiver {
			s.receiverRespawns.recordRespawn(hasWaitedForBacklogDrain || isBackpressureStop)
		}

		hasSpawnedReceiver = true

		receiverRun := s.spawnAndSupervise(ctx, logger)

		isBastionReachable := true
		if receiverRun.Exit == receiverFatal || receiverRun.Exit == receiverRetryable {
			isBastionReachable = s.isBastionReachable(ctx)
		}

		receiverRun = demoteFatalExitWhenBastionUnreachable(ctx, logger, receiverRun, isBastionReachable)

		switch receiverRun.Exit {
		case receiverCtxCancelled:
			return nil

		case receiverRetryable:
		case receiverFatal:
			return receiverRun.FatalErr

		case receiverInternalRestart:

			respawnBackoff = receivewalRespawnBackoff

			failureEscalator.reset()
			mismatchEscalator.reset()

			continue

		case receiverResumeMismatch:
			if mismatchEscalator.recordMismatchAndDecideEscalation() == resumeMismatchActionRebuildSlot {
				if err := s.rebuildSlot(ctx, logger, breakReasonSlotLost); err != nil {
					return fmt.Errorf("rebuild slot after repeated wal resume mismatch: %w", err)
				}
			}

			if !sleepCtx(ctx, receivewalRespawnBackoff) {
				return nil
			}

			continue
		}

		mismatchEscalator.reset()

		if !isBastionReachable {
			logger.WarnContext(ctx, "wal receiver exited while the bastion was unreachable, waiting for the transport")
		}

		decision := failureEscalator.recordExitAndDecideRetry(receiverRun.RanFor, isBastionReachable)
		if decision.isEscalationRequired {

			logger.ErrorContext(
				ctx,
				"pg_receivewal crash-looped",
				slog.Any("rapid_failures", receivewalMaxRapidFailures),
				slog.Any("ran_for", receiverRun.RanFor),
			)

			return fmt.Errorf(
				"pg_receivewal crash-looped: %d rapid failures, escalating for reassignment",
				receivewalMaxRapidFailures,
			)
		}

		if decision.isBackoffResettable {
			respawnBackoff = receivewalRespawnBackoff
		}

		logger.DebugContext(
			ctx,
			"respawning pg_receivewal in", slog.Any("backoff", respawnBackoff), slog.Any("ran_for", receiverRun.RanFor),
		)

		if !sleepCtx(ctx, respawnBackoff) {
			return nil
		}

		respawnBackoff = min(respawnBackoff*2, decision.maxRespawnBackoff)
	}
}

func demoteFatalExitWhenBastionUnreachable(ctx context.Context,
	logger *slog.Logger,
	receiverRun receiverRunResult,
	isBastionReachable bool,
) receiverRunResult {
	if receiverRun.Exit != receiverFatal || isBastionReachable {
		return receiverRun
	}

	logger.WarnContext(
		ctx,
		"wal receiver reported a non-retryable error while the bastion was unreachable, retrying",
		slog.Any("error", receiverRun.FatalErr),
	)

	receiverRun.Exit = receiverRetryable

	return receiverRun
}

func (s *Streamer) isBastionReachable(ctx context.Context) bool {
	if s.spec.IsBastionReachable == nil {
		return true
	}

	probeCtx, cancel := context.WithTimeout(ctx, bastionProbeTimeout)
	defer cancel()

	return s.spec.IsBastionReachable(probeCtx)
}

func (s *Streamer) waitWhilePaused(ctx context.Context) bool {
	if !s.isPaused.Load() {
		return true
	}

	ticker := time.NewTicker(pausePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false

		case <-ticker.C:
			if !s.isPaused.Load() {
				return true
			}
		}
	}
}

func (s *Streamer) signalRestart() {
	s.restartLatch.Request()
}

func (s *Streamer) clearRestartRequest() {
	s.restartLatch.Reset()
}

func (s *Streamer) getReceiverRunStartedAt() time.Time {
	startedAtUnixNano := s.receiverRunStartedAtUnixNano.Load()
	if startedAtUnixNano == 0 {
		return time.Time{}
	}

	return time.Unix(0, startedAtUnixNano).UTC()
}

func (s *Streamer) isReceiverRunning() bool {
	return s.receiverRunStartedAtUnixNano.Load() != 0
}

func walSegmentSizeBytes(sourceDB databases.PostgresqlPhysicalConnectionSettings) int64 {
	if sourceDB.WalSegmentSizeBytes != nil && *sourceDB.WalSegmentSizeBytes > 0 {
		return *sourceDB.WalSegmentSizeBytes
	}

	return int64(walmath.DefaultSegmentSizeBytes)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false

	case <-timer.C:
		return true
	}
}
