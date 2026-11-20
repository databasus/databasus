package physical

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/process"
)

const WalReceiverApplicationNamePrefix = "databasus_wal_receiver_"

type WalReceiverDisposition int

const (
	WalReceiverStopped WalReceiverDisposition = iota
	WalReceiverRetryable
	WalReceiverFatal
	WalReceiverResumeMismatch
	WalReceiverIdentityMismatch
	// WalReceiverForkRealignment means the server refused to resume beyond the fork point at which
	// its history left the requested timeline, as after a failover that lost the former primary's
	// last writes. Reception restarts from the segment holding the fork point's LSN.
	WalReceiverForkRealignment
)

type WalReceiverSpec struct {
	ExpectedSystemIdentifier string
	ClientTools              pgshared.ClientTools
	Version                  pgshared.Version
	ApplicationName          string
	WatchDir                 string
	SlotName                 string
	Clock                    clock.Source
	Logger                   *slog.Logger
	OnStarted                func()
}

type WalReceiverExit struct {
	Disposition WalReceiverDisposition
	RanFor      time.Duration
	FatalErr    error
	Fork        *WalReceiverFork
}

// WalReceiverFork is where the server's history left the timeline reception asked for.
type WalReceiverFork struct {
	LeftTimelineID int
	ForkLSN        walmath.LSN
}

func (s *Source) ReceiveWal(ctx context.Context, spec WalReceiverSpec) WalReceiverExit {
	attemptCtx, cancelAttempt := context.WithCancelCause(ctx)
	defer cancelAttempt(nil)
	setupCtx, cancelSetup := context.WithTimeout(ctx, 30*time.Second)
	defer cancelSetup()
	var isSetupPending atomic.Bool
	isSetupPending.Store(true)
	stopSetup := context.AfterFunc(setupCtx, func() {
		if errors.Is(setupCtx.Err(), context.DeadlineExceeded) && isSetupPending.Swap(false) {
			cancelAttempt(context.DeadlineExceeded)
		}
	})
	defer stopSetup()

	onStarted := spec.OnStarted
	spec.OnStarted = func() {
		if !isSetupPending.Swap(false) {
			return
		}

		cancelSetup()
		if onStarted != nil {
			onStarted()
		}
	}

	exit := s.receiveVerifiedWal(attemptCtx, spec)
	if exit.Disposition == WalReceiverStopped && ctx.Err() == nil {
		spec.Logger.WarnContext(ctx, "wal receiver setup timed out")
		exit.Disposition = WalReceiverRetryable
	}

	return exit
}

func (s *Source) receiveVerifiedWal(ctx context.Context, spec WalReceiverSpec) WalReceiverExit {
	toolPath := spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal)
	if _, err := exec.LookPath(toolPath); err != nil {
		spec.Logger.ErrorContext(
			ctx,
			"pg_receivewal binary not found",
			slog.String("path", toolPath),
			slog.Any("error", err),
		)
		return WalReceiverExit{Disposition: WalReceiverRetryable}
	}

	session, err := s.openWalSession(ctx, spec)
	if err != nil {
		if _, isMismatch := errors.AsType[*SourceIdentityError](err); isMismatch {
			return WalReceiverExit{Disposition: WalReceiverIdentityMismatch, FatalErr: err}
		}
		if errors.Is(err, ErrSourceIdentityMissing) {
			return WalReceiverExit{Disposition: WalReceiverFatal, FatalErr: err}
		}
		if ctx.Err() != nil {
			return WalReceiverExit{Disposition: WalReceiverStopped}
		}
		switch classifyServerError(err) {
		case serverErrorBadPassword,
			serverErrorNoHbaEntry,
			serverErrorNoReplicationPrivilege,
			serverErrorPermissionDenied:
			return WalReceiverExit{Disposition: WalReceiverFatal, FatalErr: err}
		case serverErrorOther, serverErrorNameTaken, serverErrorObjectMissing:
		}
		spec.Logger.WarnContext(ctx, "open verified WAL session", slog.Any("error", err))
		return WalReceiverExit{Disposition: WalReceiverRetryable}
	}
	defer session.Close(ctx)
	return s.runWalReceiver(
		ctx,
		walReceiverProcessSpec{Receiver: spec, ToolPath: toolPath, SocketDirectory: session.directory},
	)
}

type walReceiverProcessSpec struct {
	Receiver        WalReceiverSpec
	ToolPath        string
	SocketDirectory string
}

func (s *Source) runWalReceiver(ctx context.Context, processSpec walReceiverProcessSpec) WalReceiverExit {
	spec := processSpec.Receiver
	var stderr bytes.Buffer
	running, err := process.Start(ctx, process.Spec{
		Path: processSpec.ToolPath,
		Args: []string{
			"--directory=" + spec.WatchDir,
			"--slot=" + spec.SlotName,
			"--no-loop",
			"--synchronous",
			"--verbose",
			"--no-password",
			"-h",
			processSpec.SocketDirectory,
			"-p",
			"5432",
			"-U",
			s.settings.Username,
		},
		UnsetEnv: []string{"PGHOSTADDR", "PGSERVICE", "PGSERVICEFILE", "PGPASSWORD"},
		Env: []string{
			"PGAPPNAME=" + spec.ApplicationName,
			"PGSSLMODE=disable",
			"PGCONNECT_TIMEOUT=30",
			"LC_ALL=C",
			"PGPASSFILE=" + processSpec.SocketDirectory + "/.pgpass",
		},
		Stderr:     &stderr,
		StopSignal: syscall.SIGINT,
		KillAfter:  clientToolKillAfter,
	})
	if err != nil {
		if ctx.Err() != nil {
			return WalReceiverExit{Disposition: WalReceiverStopped}
		}
		spec.Logger.ErrorContext(ctx, "start pg_receivewal", slog.Any("error", err))
		return WalReceiverExit{Disposition: WalReceiverRetryable}
	}

	startedAt := spec.Clock.Now()

	spec.Logger.InfoContext(ctx, "pg_receivewal started", slog.String("watch_dir", spec.WatchDir))

	waitErr := running.Wait()
	ranFor := spec.Clock.Now().Sub(startedAt)
	if ctx.Err() != nil {
		return WalReceiverExit{Disposition: WalReceiverStopped, RanFor: ranFor}
	}

	if waitErr == nil {
		return WalReceiverExit{Disposition: WalReceiverRetryable, RanFor: ranFor}
	}

	exitErr := getClientExitError(waitErr)
	stderrBytes := stderr.Bytes()
	if leftTimelineID, isForkRefusal := getForkRefusalTimeline(stderrBytes); isForkRefusal {
		return s.getForkRealignmentExit(ctx, spec, leftTimelineID, ranFor)
	}

	if isResumeMismatchError(stderrBytes) {
		spec.Logger.WarnContext(
			ctx,
			"pg_receivewal asked for WAL the server has recycled; realigning the resume point",
			slog.Any("error", exitErr),
			slog.Duration("ran_for", ranFor),
			slog.String("stderr", truncateStderr(stderrBytes)),
		)
		return WalReceiverExit{Disposition: WalReceiverResumeMismatch, RanFor: ranFor}
	}

	if isFatalReceivewalError(stderrBytes) {
		return WalReceiverExit{
			Disposition: WalReceiverFatal,
			RanFor:      ranFor,
			FatalErr:    fmt.Errorf("pg_receivewal fatal error: %w; stderr: %s", exitErr, truncateStderr(stderrBytes)),
		}
	}
	spec.Logger.WarnContext(
		ctx,
		"pg_receivewal exited; will respawn",
		slog.Any("error", exitErr),
		slog.Duration("ran_for", ranFor),
		slog.String("stderr", truncateStderr(stderrBytes)),
	)

	return WalReceiverExit{Disposition: WalReceiverRetryable, RanFor: ranFor}
}

func (s *Source) getForkRealignmentExit(
	ctx context.Context,
	spec WalReceiverSpec,
	leftTimelineID int,
	ranFor time.Duration,
) WalReceiverExit {
	forkLSN, err := s.getLiveForkLSN(ctx, spec.ExpectedSystemIdentifier, leftTimelineID)
	if err != nil {
		spec.Logger.WarnContext(
			ctx,
			"pg_receivewal resumed beyond the server's history and its fork point could not be read",
			slog.Int("timeline_id", leftTimelineID),
			slog.Any("error", err),
		)

		return WalReceiverExit{Disposition: WalReceiverRetryable, RanFor: ranFor}
	}

	spec.Logger.WarnContext(
		ctx,
		"pg_receivewal resumed beyond the fork point of the server's history; realigning to the fork point",
		slog.Int("timeline_id", leftTimelineID),
		slog.String("fork_lsn", forkLSN.String()),
	)

	return WalReceiverExit{
		Disposition: WalReceiverForkRealignment,
		RanFor:      ranFor,
		Fork:        &WalReceiverFork{LeftTimelineID: leftTimelineID, ForkLSN: forkLSN},
	}
}

// A full / unwritable local disk, rejected credentials or a slot held by another
// consumer escalate to streamer-FAILED so the supervisor can reclaim it on a
// later tick, instead of this process crash-looping on an unfixable cause.
func isFatalReceivewalError(stderr []byte) bool {
	// Lower-case both sides: OS errno strings vary in case ("Permission denied")
	// while PG messages are lower-case, and we want to match either.
	text := strings.ToLower(string(stderr))

	for _, needle := range []string{
		"no space left on device",
		"could not write",
		"authentication failed",
		"no pg_hba.conf entry",
		"permission denied",
		"is active for", // replication slot "<name>" is active for PID <n>
	} {
		if strings.Contains(text, needle) {
			return true
		}
	}

	return false
}

// The resume point pg_receivewal derived from the local queue sits below what
// the server still has. A local respawn alone repeats it forever, so this is its
// own disposition: the supervision loop realigns the queue and tries again.
func isResumeMismatchError(stderr []byte) bool {
	return strings.Contains(strings.ToLower(string(stderr)), "has already been removed")
}

// The verbatim refusal from PostgreSQL's walsender.c when the requested start lies on a timeline the
// server left at a fork point before that position, as after a failover that lost the former
// primary's last writes.
var forkRefusalPattern = regexp.MustCompile(
	`requested starting point [0-9A-F]+/[0-9A-F]+ on timeline (\d+) is not in this server's history`,
)

// getForkRefusalTimeline returns the timeline pg_receivewal asked for when the server refused the
// start position as outside its history.
func getForkRefusalTimeline(stderr []byte) (int, bool) {
	match := forkRefusalPattern.FindSubmatch(stderr)
	if match == nil {
		return 0, false
	}

	timelineID, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0, false
	}

	return timelineID, true
}
