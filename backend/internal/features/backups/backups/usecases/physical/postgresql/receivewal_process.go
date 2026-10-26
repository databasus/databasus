package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
)

const receivewalApplicationNamePrefix = "databasus_wal_receiver_"

// receiverExit is decided by spawnAndSupervise and acted on by runReceivewalSupervision.
type receiverExit int

const (
	receiverCtxCancelled    receiverExit = iota // ctx cancelled — stop the loop, no error
	receiverInternalRestart                     // our own SIGTERM (back pressure / slot stall) — respawn promptly
	receiverRetryable                           // non-zero exit that a local respawn may fix (network)
	receiverFatal                               // non-retryable exit — escalate so the supervisor reclaims it on a later tick
	receiverResumeMismatch                      // asked for WAL the server no longer has — realign the resume point and retry
)

type receiverRunResult struct {
	Exit     receiverExit
	RanFor   time.Duration
	FatalErr error
}

type receivewalCommandSpec struct {
	PgBin    string
	SourceDB databases.PostgresqlPhysicalConnectionSettings
	Creds    *pgshared.CredentialTempFiles
	WatchDir string
	SlotName string
}

func (s *WalStreamSupervisor) spawnAndSupervise(
	ctx context.Context,
	logger *slog.Logger,
	pgBin string,
) receiverRunResult {
	creds, err := pgshared.WriteCredentialFilesToTempDir(
		s.spec.SourceDB.Engine.GetCredentialSpec(), s.spec.SourceDB.Engine.Password,
	)
	if err != nil {
		logger.ErrorContext(ctx, "write pg_receivewal credentials", "error", err)

		return receiverRunResult{Exit: receiverRetryable}
	}
	defer creds.Remove()

	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	cmd, err := newReceivewalCommand(procCtx, receivewalCommandSpec{
		PgBin:    pgBin,
		SourceDB: s.spec.SourceDB,
		Creds:    creds,
		WatchDir: s.watchDir,
		SlotName: s.slotName,
	})
	if err != nil {
		logger.ErrorContext(ctx, "build pg_receivewal command", "error", err)

		return receiverRunResult{Exit: receiverRetryable}
	}

	// Wait returns only after the whole error output is in the buffer, bounded by the command's
	// WaitDelay, so reading it once exited delivers always sees what pg_receivewal wrote.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		logger.ErrorContext(ctx, "start pg_receivewal", "error", err)

		return receiverRunResult{Exit: receiverRetryable}
	}

	startedAt := time.Now().UTC()

	s.receiverRunStartedAtUnixNano.Store(startedAt.UnixNano())
	defer s.receiverRunStartedAtUnixNano.Store(0)

	logger.InfoContext(ctx, "pg_receivewal started", "watch_dir", s.watchDir)

	exited := make(chan error, 1)

	go func() { exited <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		procCancel()
		<-exited

		return receiverRunResult{Exit: receiverCtxCancelled}

	case <-s.restartSignal:
		logger.InfoContext(ctx, "restarting pg_receivewal on internal signal (back pressure or slot stall)")
		procCancel()
		<-exited

		return receiverRunResult{Exit: receiverInternalRestart, RanFor: time.Since(startedAt)}

	case waitErr := <-exited:
		ranFor := time.Since(startedAt)

		if waitErr == nil || procCtx.Err() != nil {
			logger.DebugContext(ctx, "pg_receivewal exited cleanly", "ran_for", ranFor)

			return receiverRunResult{Exit: receiverRetryable, RanFor: ranFor}
		}

		stderrText := stderr.Bytes()

		if isResumeMismatchError(stderrText) {
			logger.WarnContext(ctx, "pg_receivewal asked for WAL the server has recycled; realigning the resume point",
				"error", waitErr, "ran_for", ranFor, "stderr", truncateStderr(stderrText))

			return receiverRunResult{Exit: receiverResumeMismatch, RanFor: ranFor}
		}

		if isFatalReceivewalError(stderrText) {
			logger.ErrorContext(ctx, "pg_receivewal exited with a non-retryable error",
				"error", waitErr, "ran_for", ranFor, "stderr", truncateStderr(stderrText))

			return receiverRunResult{
				Exit:   receiverFatal,
				RanFor: ranFor,
				FatalErr: fmt.Errorf(
					"pg_receivewal fatal error: %w; stderr: %s", waitErr, truncateStderr(stderrText),
				),
			}
		}

		logger.WarnContext(ctx, "pg_receivewal exited; will respawn",
			"error", waitErr, "ran_for", ranFor, "stderr", truncateStderr(stderrText))

		return receiverRunResult{Exit: receiverRetryable, RanFor: ranFor}
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

// WAL is left uncompressed locally (no --compress) because the uploader
// re-compresses with zstd on upload; --no-loop makes the process exit on
// connection loss so the supervision loop owns retry; --synchronous flushes each
// segment promptly. SSL is supplied through the same PGSSL* env path
// pg_basebackup uses, so mTLS needs no extra handling here.
func newReceivewalCommand(ctx context.Context, spec receivewalCommandSpec) (*exec.Cmd, error) {
	if _, err := exec.LookPath(spec.PgBin); err != nil {
		return nil, fmt.Errorf("pg_receivewal binary not found at %s: %w", spec.PgBin, err)
	}

	args := []string{
		"--directory=" + spec.WatchDir,
		"--slot=" + spec.SlotName,
		"--no-loop",
		"--synchronous",
		"--verbose",
		"--no-password",
		"-h", spec.SourceDB.Engine.Host,
		"-p", strconv.Itoa(spec.SourceDB.Engine.Port),
		"-U", spec.SourceDB.Engine.Username,
	}

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, spec.PgBin, args...)

	// -h above stays the real host so the certificate and .pgpass still match; PGHOSTADDR is what
	// redirects libpq at the tunnel.
	cmd.Env = append(os.Environ(),
		"PGPASSFILE="+spec.Creds.PgpassPath,
		"PGAPPNAME="+receivewalApplicationName(spec.SourceDB),
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT=30",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)
	cmd.Env = append(cmd.Env, pgshared.GetPgHostAddrEnv(spec.SourceDB.Engine.HostAddr)...)

	sslMode := spec.SourceDB.Engine.TLS.Mode
	if sslMode == "" {
		sslMode = pgshared.SslModeDisable
	}

	cmd.Env = append(cmd.Env,
		"PGSSLMODE="+string(sslMode),
		"PGSSLCERT="+spec.Creds.ClientCertPath,
		"PGSSLKEY="+spec.Creds.ClientKeyPath,
		"PGSSLROOTCERT="+spec.Creds.RootCertPath,
		"PGSSLCRL=",
	)

	cmd.Cancel = func() error {
		return signalForGracefulCancel(cmd.Process)
	}

	cmd.WaitDelay = killAfterCancel
	setReceivewalProcessAttributes(cmd)

	return cmd, nil
}

func receivewalApplicationName(sourceDB databases.PostgresqlPhysicalConnectionSettings) string {
	return receivewalApplicationNamePrefix + sourceDB.DatabaseID.String()
}

// Pdeathsig makes the kernel SIGTERM pg_receivewal if the Databasus process
// dies, so a crashed supervisor never leaks an orphaned receiver that keeps
// holding the replication slot. Linux-only; Databasus ships Linux containers
// exclusively.
func setReceivewalProcessAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}
