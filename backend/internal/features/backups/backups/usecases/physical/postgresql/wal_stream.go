package usecases_physical_postgresql

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

	"databasus-backend/internal/config"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/util/walmath"
)

const (
	// Initial pause between a pg_receivewal exit and respawn so a hard-failing
	// source (auth, pg_hba) is not hammered. --no-loop makes pg_receivewal exit
	// on connection loss; this loop is its supervision.
	receivewalRespawnBackoff = 2 * time.Second

	receivewalRespawnMaxBackoff = 30 * time.Minute

	// While the bastion is down the backoff must keep growing, or the loop hammers it at the floor
	// interval — but not up to the ceiling above, which would leave half an hour of silence on an
	// already healed transport, WAL piling up on the source and a staleness alert firing after the
	// fact. Half a minute is neither hammering nor a wait anyone notices.
	receivewalBastionDownMaxBackoff = 30 * time.Second

	// A receiver that streamed at least this long before exiting counts as a transient blip (network
	// drop, slot resend) and resets the crash-loop counter; a shorter run counts toward escalation.
	// A run whose bastion was already gone when it exited is not weighed against this at all — see
	// recordExitAndDecideRetry.
	receivewalMinHealthyUptime = 15 * time.Second

	// This many back-to-back sub-uptime exits escalate to a fatal supervisor
	// error so the streamer row is marked FAILED and the supervisor can reclaim
	// it on a later tick, instead of crash-looping locally forever on a condition
	// a local respawn can never fix (ENOSPC, bad creds, a slot held by a thief).
	receivewalMaxRapidFailures = 5

	// A realign empties the resume path, so the very next spawn starts from the
	// slot. A second mismatch means the slot itself no longer covers what the
	// server retains — only a rebuild clears that.
	receivewalMaxResumeMismatches = 2

	// Sits between two receiver spawns, so it has to answer fast; a bastion that cannot say hello in
	// this long is unreachable enough for the decision at hand.
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

// All three hang on the same "was that run healthy" judgement, so they come out of one call;
// deriving them separately lets them drift.
type retryableExitDecision struct {
	isEscalationRequired bool
	isBackoffResettable  bool
	maxRespawnBackoff    time.Duration
}

// A run whose bastion was already gone when it exited says nothing about the source, so it is no
// evidence of a crash loop: counting it would let a restarted sshd mark the streamer FAILED in half
// a minute, firing a chain-broken alert and leaving nobody consuming the slot until a later
// supervisor tick reclaims it. It does not reset the backoff either, or the loop would hammer the
// bastion for as long as it stays down.
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
	DatabaseID   uuid.UUID
	SourceDB     databases.PostgresqlPhysicalConnectionSettings
	StorageID    uuid.UUID
	FileStore    FileStore
	BackupStatus BackupStatusRecorder

	// Checked after every slot creation, so a deleted database keeps no slot; nil checks nothing.
	Databases StreamedDatabases

	Encryption     backups_core_enums.BackupEncryption
	MasterKey      string
	WalSegmentRepo *physical_repositories.PhysicalWalSegmentRepository
	HistoryRepo    *physical_repositories.PhysicalWalHistoryRepository

	// Where the lag monitor records one sample per tick; nil records none.
	WalStreamSampleRepo *physical_repositories.PhysicalWalStreamSampleRepository

	// The lag monitor clears the streamer's failing time with a session's first
	// sample; nil leaves it alone.
	WalStreamerRepo *physical_repositories.PhysicalWalStreamerRepository

	// config.DataFolder; the per-DB queue lives under
	// <root>/wal-queue/<database_id>/. It must survive a process restart so crash
	// recovery can re-process finalized-but-not-uploaded segments.
	WatchDirRoot string

	// A slot lag over this many bytes that the running receiver does not reduce
	// triggers a rebuild (slot_lag_window.go).
	WalLagThresholdBytes int64

	// How often a source that is being written to is asked to close its current WAL
	// segment, so the newest WAL reaches storage without waiting for the segment to
	// fill (wal_rotation.go). Zero disables it.
	ForcedRotationInterval time.Duration

	// How far the archived recovery point may fall behind a source that keeps
	// writing before the operator is alerted (wal_staleness.go). Zero disables it.
	ArchiveStalenessThreshold time.Duration

	// Fires once per newly-observed WAL gap (see WalUploader); nil disables
	// notification.
	OnGapDetected func(gapStart, gapEnd walmath.LSN)

	// Fires after the persistent slot has been recreated. Callers use it to
	// request a fresh base backup that anchors the new WAL chain.
	OnSlotRebuilt func(ctx context.Context, reason string) error

	// Fires while the chain is still intact but degrading (slot retention
	// warnings, a wedged receiver); nil disables notification.
	OnChainAtRisk func(report ChainRiskReport)

	// Separates "the bastion went away" from "the receiver is crash-looping". Without it a restarted
	// sshd looks like five rapid failures and tears the streamer down. Nil means no bastion, so
	// every failure is the source's own.
	IsBastionReachable func(ctx context.Context) bool

	Logger *slog.Logger
}

// One supervisor owns one database's pg_receivewal process. Run blocks until ctx
// is cancelled.
type WalStreamSupervisor struct {
	spec     WalStreamSpec
	logger   *slog.Logger
	uploader *WalUploader
	watchDir string
	slotName string

	// Every sample of this session carries it, so the read path can count session
	// starts within a window.
	sessionStartedAt time.Time

	// Derived once from the source's (immutable) wal_segment_size; recomputing
	// them on every poll tick would be wasted work. Atomic because the
	// back-pressure monitor, the supervision loop and the lag monitor read them
	// from their own goroutines while a test may lower them.
	highWatermarkBytes atomic.Int64
	lowWatermarkBytes  atomic.Int64

	// Asks the supervision loop to SIGTERM the current pg_receivewal and respawn
	// (sent by the back-pressure monitor and the slot-LSN watcher). Buffered size
	// 1; sends are non-blocking and coalesced.
	restartSignal chan struct{}

	// Holds the supervision loop between pg_receivewal runs so a slot rebuild can
	// drop+recreate the slot without the receiver re-attaching.
	isPaused atomic.Bool

	// When the current pg_receivewal run started, as Unix nanoseconds; zero while
	// no receiver runs. One value carries both "running" and "since when", so the
	// slot-LSN watcher and the lag monitor can tell a real problem from lag we are
	// causing ourselves by holding the receiver down, and never see a running
	// receiver without its start time.
	receiverRunStartedAtUnixNano atomic.Int64

	// Set by the back-pressure monitor when it stops the receiver, so the respawn
	// that follows counts as a pause rather than a restart.
	isBackpressureStopPending atomic.Bool

	// Taken and reset by the lag monitor with each sample.
	receiverRespawns receiverRespawnCounter

	// The source refused pg_switch_wal in this session; every later sample says so.
	isForcedRotationDenied atomic.Bool

	// rebuildMu serializes slot rebuilds in this process; rebuildTimestamps powers
	// the per-hour loop-protection cap. One supervisor owns a DB at a time (the
	// physical_wal_streamers heartbeat claim), so this is the only guard needed.
	rebuildMu         sync.Mutex
	rebuildTimestamps []time.Time

	// Written by the uploader dispatcher, read by the lag monitor (wal_staleness.go).
	lowestQueuedSegmentMutex sync.Mutex
	lowestQueuedSegment      *walPosition

	// The newest slot probe of any caller, read by GetLiveState.
	latestProbe atomic.Pointer[timedSlotProbe]
}

func NewWalStreamSupervisor(spec WalStreamSpec) *WalStreamSupervisor {
	watchDir := filepath.Join(spec.WatchDirRoot, "wal-queue", spec.DatabaseID.String())

	// The uploader logs per segment, so it needs the same scoping the supervisor uses; handing it the
	// raw spec logger leaves every archiving line unattributable on a multi-database instance.
	scopedLogger := spec.Logger.With(
		"database_id", spec.DatabaseID,
		"slot_name", spec.SourceDB.Engine.ReplicationSlotName,
	)

	uploader := NewWalUploader(WalUploadDeps{
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

	supervisor := &WalStreamSupervisor{
		spec:             spec,
		logger:           scopedLogger,
		uploader:         uploader,
		watchDir:         watchDir,
		slotName:         spec.SourceDB.Engine.ReplicationSlotName,
		sessionStartedAt: time.Now().UTC(),
		restartSignal:    make(chan struct{}, 1),
	}

	supervisor.setLocalQueuePauseBytes(getDefaultLocalQueuePauseBytes(spec.SourceDB))

	return supervisor
}

func (s *WalStreamSupervisor) Run(ctx context.Context) error {
	logger := s.logger

	// pg_receivewal finalizes a segment by writing a marker into <dir>/archive_status/
	// and refuses to start (or errors mid-stream) if that subdirectory is absent — it
	// does not create it itself. Create both up front.
	if err := os.MkdirAll(filepath.Join(s.watchDir, "archive_status"), 0o700); err != nil {
		return fmt.Errorf("create wal watch dir: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(s.watchDir, pendingUploadDirName), 0o700); err != nil {
		return fmt.Errorf("create pending upload dir: %w", err)
	}

	if err := s.createSlotUnlessDatabaseDeleted(ctx, logger); err != nil {
		if errors.Is(err, errStreamedDatabaseDeleted) {
			logger.InfoContext(ctx, "wal stream supervisor not started because the database is deleted")

			return nil
		}

		return fmt.Errorf("verify persistent replication slot: %w", err)
	}

	// Crash recovery: clear torn *.partial files (the slot resends them) and take
	// over any finalized-but-not-uploaded segments left by a previous crash, so
	// recovery does not wait on the cleaner's grace sweep. Runs before the receiver
	// spawns, so there is no concurrent writer in watch_dir.
	s.removePartials(logger)
	s.recoverLocalSegmentsOnStartup(ctx, logger)

	// A fatal pg_receivewal exit (disk full, auth, stolen slot, crash loop) must
	// tear down the whole supervisor — not just the receiver — so the streamer
	// row is marked FAILED and reclaimed on a later supervisor tick. Derive a cancelable
	// ctx the auxiliary loops share and cancel it when supervision returns fatal.
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	var wg sync.WaitGroup

	for _, loop := range []func(context.Context, *slog.Logger){
		s.runUploaderLoop,
		s.runBackpressureMonitor,
		s.runSlotLsnWatcher,
		s.runLagMonitor,
		s.runForcedWalRotation,
	} {
		wg.Go(func() { loop(runCtx, logger) })
	}

	fatalErr := s.runReceivewalSupervision(runCtx, logger)

	cancelRun()
	wg.Wait()

	if errors.Is(fatalErr, errStreamedDatabaseDeleted) {
		logger.InfoContext(ctx, "wal stream supervisor stopped because the database is deleted")

		return nil
	}

	if fatalErr != nil {
		logger.ErrorContext(ctx, "wal stream supervisor stopping with fatal error", "error", fatalErr)

		return fatalErr
	}

	logger.InfoContext(ctx, "wal stream supervisor stopped")

	return nil
}

// Returns a non-nil error only when the receiver is unrecoverable here (fatal
// exit or a crash loop), so Run can mark the streamer FAILED for reclaim on a
// later tick.
func (s *WalStreamSupervisor) runReceivewalSupervision(ctx context.Context, logger *slog.Logger) error {
	pgBin := pgshared.ClientTools{
		Root: config.GetEnv().ClientToolsDir,
	}.GetExecutable(
		s.spec.SourceDB.Version,
		pgshared.ExecutablePgReceivewal,
	)
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

		// Clear any stale restart signal so a spawn does not get cancelled by a
		// signal raised while no process was running.
		s.drainRestartSignal()
		s.removePartials(logger)
		s.realignResumePath(ctx, logger)

		isBackpressureStop := s.isBackpressureStopPending.Swap(false)

		if hasSpawnedReceiver {
			s.receiverRespawns.recordRespawn(hasWaitedForBacklogDrain || isBackpressureStop)
		}

		hasSpawnedReceiver = true

		receiverRun := s.spawnAndSupervise(ctx, logger, pgBin)

		// One probe per iteration, shared by both dispositions that judge the source: asking twice
		// lets a bastion that comes back in between answer "gone" to the demotion and "here" to the
		// crash-loop counter, which is how a transport blip still ends up counted as a rapid failure.
		// The dispositions that never read the answer do not pay for the round trip, and default to
		// the same answer a database with no bastion in front of it gets.
		isBastionReachable := true
		if receiverRun.Exit == receiverFatal || receiverRun.Exit == receiverRetryable {
			isBastionReachable = s.isBastionReachable(ctx)
		}

		receiverRun = demoteFatalExitWhenBastionUnreachable(logger, receiverRun, isBastionReachable)

		switch receiverRun.Exit {
		case receiverCtxCancelled:
			return nil

		case receiverFatal:
			return receiverRun.FatalErr

		case receiverInternalRestart:
			// Our own SIGTERM (back pressure or slot stall): respawn promptly; the
			// top-of-loop backlog/pause gates already throttle the cause.
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
			// The streamer is about to be marked FAILED and handed to another instance; without this
			// the only evidence is a generic fatal at the top of Run.
			logger.ErrorContext(ctx, fmt.Sprintf(
				"pg_receivewal crash-looped: %d rapid failures, escalating for reassignment",
				receivewalMaxRapidFailures), "ran_for", receiverRun.RanFor)

			return fmt.Errorf(
				"pg_receivewal crash-looped: %d rapid failures, escalating for reassignment",
				receivewalMaxRapidFailures,
			)
		}

		if decision.isBackoffResettable {
			respawnBackoff = receivewalRespawnBackoff
		}

		// A streamer sitting in a long backoff is otherwise completely silent for its duration.
		logger.DebugContext(ctx, fmt.Sprintf("respawning pg_receivewal in %s", respawnBackoff),
			"ran_for", receiverRun.RanFor)

		if !sleepCtx(ctx, respawnBackoff) {
			return nil
		}

		respawnBackoff = min(respawnBackoff*2, decision.maxRespawnBackoff)
	}
}

// A fatal verdict is read off the receiver's stderr, and a receiver whose transport was already gone
// never got far enough to say anything about the source. The needles that classify a run as fatal
// are broad ("could not write", "permission denied"), so a run dying with the bastion can match one
// on the way down and hand the streamer back as FAILED over a blip the forwarder heals on its own.
// While the bastion is unreachable that verdict is not evidence; once it is back, the next one is
// read off trustworthy output and escalates if the cause was real.
func demoteFatalExitWhenBastionUnreachable(
	logger *slog.Logger,
	receiverRun receiverRunResult,
	isBastionReachable bool,
) receiverRunResult {
	if receiverRun.Exit != receiverFatal || isBastionReachable {
		return receiverRun
	}

	logger.Warn("wal receiver reported a non-retryable error while the bastion was unreachable, retrying",
		"error", receiverRun.FatalErr)

	receiverRun.Exit = receiverRetryable

	return receiverRun
}

func (s *WalStreamSupervisor) isBastionReachable(ctx context.Context) bool {
	if s.spec.IsBastionReachable == nil {
		return true
	}

	probeCtx, cancel := context.WithTimeout(ctx, bastionProbeTimeout)
	defer cancel()

	return s.spec.IsBastionReachable(probeCtx)
}

func (s *WalStreamSupervisor) waitWhilePaused(ctx context.Context) bool {
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

func (s *WalStreamSupervisor) signalRestart() {
	select {
	case s.restartSignal <- struct{}{}:
	default:
	}
}

func (s *WalStreamSupervisor) drainRestartSignal() {
	select {
	case <-s.restartSignal:
	default:
	}
}

// Zero while no receiver runs.
func (s *WalStreamSupervisor) getReceiverRunStartedAt() time.Time {
	startedAtUnixNano := s.receiverRunStartedAtUnixNano.Load()
	if startedAtUnixNano == 0 {
		return time.Time{}
	}

	return time.Unix(0, startedAtUnixNano).UTC()
}

func (s *WalStreamSupervisor) isReceiverRunning() bool {
	return s.receiverRunStartedAtUnixNano.Load() != 0
}

func walSegmentSizeBytes(sourceDB databases.PostgresqlPhysicalConnectionSettings) int64 {
	if sourceDB.WalSegmentSizeBytes != nil && *sourceDB.WalSegmentSizeBytes > 0 {
		return *sourceDB.WalSegmentSizeBytes
	}

	return int64(walmath.WalSegmentSize)
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
