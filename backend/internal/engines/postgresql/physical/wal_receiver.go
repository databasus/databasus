package physical

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

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
)

type WalReceiverSpec struct {
	ClientTools     pgshared.ClientTools
	Version         pgshared.Version
	ApplicationName string
	WatchDir        string
	SlotName        string
	Clock           clock.Source
	Logger          *slog.Logger
	OnStarted       func()
}

type WalReceiverExit struct {
	Disposition WalReceiverDisposition
	RanFor      time.Duration
	FatalErr    error
}

func (s *Source) ReceiveWal(ctx context.Context, spec WalReceiverSpec) WalReceiverExit {
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

	creds, err := pgshared.WriteCredentialFilesToTempDir(s.settings.GetCredentialSpec(), s.settings.Password)
	if err != nil {
		spec.Logger.ErrorContext(ctx, "write pg_receivewal credentials", slog.Any("error", err))
		return WalReceiverExit{Disposition: WalReceiverRetryable}
	}

	defer creds.Remove()
	var stderr bytes.Buffer
	running, err := process.Start(ctx, process.Spec{
		Path: toolPath,
		Args: []string{
			"--directory=" + spec.WatchDir,
			"--slot=" + spec.SlotName,
			"--no-loop",
			"--synchronous",
			"--verbose",
			"--no-password",
			"-h",
			s.settings.Host,
			"-p",
			strconv.Itoa(s.settings.Port),
			"-U",
			s.settings.Username,
		},
		Env:        append(s.getClientEnvironment(creds), "PGAPPNAME="+spec.ApplicationName),
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
	if spec.OnStarted != nil {
		spec.OnStarted()
	}

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
