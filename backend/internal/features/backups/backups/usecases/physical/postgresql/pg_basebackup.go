package usecases_physical_postgresql

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/util/walmath"
)

// killAfterCancel is how long we wait between sending SIGTERM (or its platform
// equivalent) and escalating to SIGKILL. The byte-stall watcher triggers cancel;
// pg_basebackup should exit cleanly on SIGTERM but a stuck network FD may keep it
// alive.
const killAfterCancel = 10 * time.Second

// pgBasebackupStartPointRegexp parses lines like
// "pg_basebackup: ... write-ahead log start point: 0/3000060 on timeline 1".
var pgBasebackupStartPointRegexp = regexp.MustCompile(
	`write-ahead log start point:\s+([0-9A-Fa-f]+/[0-9A-Fa-f]+)\s+on timeline\s+(\d+)`,
)

// pgBasebackupStopPointRegexp parses lines like
// "pg_basebackup: ... write-ahead log end point: 0/3000220".
var pgBasebackupStopPointRegexp = regexp.MustCompile(
	`write-ahead log end point:\s+([0-9A-Fa-f]+/[0-9A-Fa-f]+)`,
)

// newPgBasebackupCommand builds the pg_basebackup invocation shared by FULL and
// INCR. incrementalManifestPath is "" for a FULL; a non-empty path adds
// --incremental=<path>, the only CLI difference between the two backup kinds. The
// codec is supplied per-attempt by the fallback loop in streamWithCodecFallback.
func newPgBasebackupCommand(
	ctx context.Context,
	pgBin string,
	sourceDB *postgresql_physical.PostgresqlPhysicalDatabase,
	creds *postgresql_shared.CredentialTempFiles,
	label string,
	codec physical_enums.PhysicalBackupCompression,
	incrementalManifestPath string,
) (*exec.Cmd, error) {
	if _, err := exec.LookPath(pgBin); err != nil {
		return nil, fmt.Errorf("pg_basebackup binary not found at %s: %w", pgBin, err)
	}

	args := []string{
		"--pgdata=-",
		"--format=tar",
		// Server-side compression so only ~1/3 of the bytes cross the
		// PG->Databasus link (docs/architecture.md, "Physical backups"). server-zstd + tar-to-stdout makes
		// pg_basebackup refuse to inject the manifest, hence --no-manifest below.
		"--compress=" + compressFlag(codec),
		// --wal-method=fetch inlines WAL into the same tar; --wal-method=stream
		// requires a separate output stream which pg_basebackup refuses when
		// writing tar to stdout. Our persistent replication slot keeps the
		// fetched WAL segments retained on the source until the backup completes.
		"--wal-method=fetch",
		"--checkpoint=fast",
	}

	if incrementalManifestPath != "" {
		args = append(args, "--incremental="+incrementalManifestPath)
	}

	args = append(args,
		// --no-manifest: server compression forbids manifest injection, so we
		// reconstruct backup_manifest from the teed stream. --manifest-checksums
		// is intentionally omitted (it conflicts with --no-manifest); our
		// serializer fixes the per-file checksum at SHA256.
		"--no-manifest",
		"--label="+label,
		"--no-password",
		"--verbose",
		"-h", sourceDB.Host,
		"-p", strconv.Itoa(sourceDB.Port),
		"-U", sourceDB.Username,
	)

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, pgBin, args...)

	// -h above stays the real host so the certificate and .pgpass still match; PGHOSTADDR is what
	// redirects libpq at the tunnel.
	cmd.Env = append(os.Environ(),
		"PGPASSFILE="+creds.PgpassPath,
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT=30",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)
	cmd.Env = append(cmd.Env, postgresql_shared.GetPgHostAddrEnv(sourceDB.CredentialSpec())...)

	sslMode := sourceDB.SslMode
	if sslMode == "" {
		sslMode = postgresql_shared.PostgresSslModeDisable
	}

	cmd.Env = append(cmd.Env,
		"PGSSLMODE="+string(sslMode),
		"PGSSLCERT="+creds.ClientCertPath,
		"PGSSLKEY="+creds.ClientKeyPath,
		"PGSSLROOTCERT="+creds.RootCertPath,
		"PGSSLCRL=",
	)

	cmd.Cancel = func() error {
		return signalForGracefulCancel(cmd.Process)
	}

	cmd.WaitDelay = killAfterCancel

	return cmd, nil
}

// parseLsnsFromStderr extracts the start/stop LSN and timeline pg_basebackup
// prints to stderr in --verbose mode. Both points are only complete once the
// process exits, so the caller reads them after Wait.
func parseLsnsFromStderr(stderr []byte) (walmath.LSN, walmath.LSN, int, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(stderr)))

	var (
		startLSN, stopLSN walmath.LSN
		timelineID        int
		gotStart, gotStop bool
	)

	for scanner.Scan() {
		line := scanner.Text()

		if matches := pgBasebackupStartPointRegexp.FindStringSubmatch(line); matches != nil {
			parsed, err := walmath.ParseLSN(matches[1])
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse start LSN: %w", err)
			}

			tli, err := strconv.Atoi(matches[2])
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse timeline ID: %w", err)
			}

			startLSN = parsed
			timelineID = tli
			gotStart = true
		}

		if matches := pgBasebackupStopPointRegexp.FindStringSubmatch(line); matches != nil {
			parsed, err := walmath.ParseLSN(matches[1])
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse stop LSN: %w", err)
			}

			stopLSN = parsed
			gotStop = true
		}
	}

	if !gotStart || !gotStop {
		return 0, 0, 0, errors.New("pg_basebackup did not emit both start and end points")
	}

	return startLSN, stopLSN, timelineID, nil
}

func signalForGracefulCancel(p *os.Process) error {
	if p == nil {
		return nil
	}

	return p.Signal(os.Interrupt)
}

func truncateStderr(b []byte) string {
	const max = 2048

	if len(b) <= max {
		return string(b)
	}

	return string(b[len(b)-max:])
}

// stderrCapture drains a Reader (pg_basebackup stderr) without blocking the
// caller. The goroutine reads until EOF or stop() is called; contents() returns
// whatever was captured so far.
type stderrCapture struct {
	pipe io.ReadCloser
	mu   sync.Mutex
	buf  []byte
	done chan struct{}
	once sync.Once
}

func newStderrCapture(pipe io.ReadCloser) *stderrCapture {
	c := &stderrCapture{
		pipe: pipe,
		done: make(chan struct{}),
	}

	go func() {
		defer close(c.done)

		out, _ := io.ReadAll(pipe)

		c.mu.Lock()
		c.buf = out
		c.mu.Unlock()
	}()

	return c
}

// drain blocks until the capture goroutine reaches natural EOF (the child closed
// its stderr), so the full output is buffered, or until timeout. It must run
// BEFORE cmd.Wait(), which closes the read pipe and would otherwise truncate a
// fast-exiting child's stderr out from under the reader — os/exec: "it is
// incorrect to call Wait before all reads from the pipe have completed". Returns
// true on natural EOF; on timeout the caller falls back to stop()'s force-close.
func (c *stderrCapture) drain(timeout time.Duration) bool {
	select {
	case <-c.done:
		return true

	case <-time.After(timeout):
		return false
	}
}

func (c *stderrCapture) stop() {
	c.once.Do(func() {
		_ = c.pipe.Close()
		<-c.done
	})
}

func (c *stderrCapture) contents() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]byte, len(c.buf))
	copy(out, c.buf)

	return out
}
