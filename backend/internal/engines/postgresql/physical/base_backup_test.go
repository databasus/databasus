package physical

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
)

// baseBackupFileSizeFittingAnyPipe keeps the archive inside the pipe buffer, so the tool can exit
// while the output is stuck. The kernel gives a new pipe only 8 KiB once the user holds too many
// pipes, which happens while several test runs share a machine.
const baseBackupFileSizeFittingAnyPipe = 1024

func newBaseBackupTestSpec(t *testing.T) BaseBackupSpec {
	t.Helper()
	return BaseBackupSpec{
		ClientTools: pgshared.ClientTools{Root: t.TempDir()}, Version: pgshared.Version17,
		Credentials: &pgshared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"}, Label: "test-label", Codec: CodecNone,
		SystemID: 7361234567890123456, Clock: clock.NewSystem(), FirstByteTimeout: time.Hour, StallTimeout: time.Hour,
		StallPollInterval: time.Millisecond, ProgressInterval: time.Millisecond, Logger: slog.New(slog.DiscardHandler),
	}
}

func writeBaseBackupStub(t *testing.T, spec BaseBackupSpec, script string) *toolstub.Stub {
	t.Helper()
	return toolstub.Write(
		t,
		spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgBasebackup),
		toolstub.Spec{Script: script},
	)
}

func writeBaseBackupTar(t *testing.T) string {
	t.Helper()
	return writeBaseBackupTarWithFile(t, []byte("17\n"))
}

func writeBaseBackupTarWithFile(t *testing.T, fileContent []byte) string {
	t.Helper()
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	require.NoError(t, writer.WriteHeader(&tar.Header{
		Name:     "PG_VERSION",
		Mode:     0o600,
		Size:     int64(len(fileContent)),
		Typeflag: tar.TypeReg,
	}))
	_, err := writer.Write(fileContent)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	path := filepath.Join(t.TempDir(), "base.tar")
	require.NoError(t, os.WriteFile(path, archive.Bytes(), 0o600))
	return path
}

const baseBackupLsnScript = `echo 'pg_basebackup: write-ahead log start point: 0/3000060 on timeline 1' >&2
 echo 'pg_basebackup: write-ahead log end point: 0/3000220' >&2`

// baseBackupTwoWritesScript sends the rest of the archive only after the first write was accepted.
// Otherwise the stream could read the small archive at once, and no second write would ever happen.
func baseBackupTwoWritesScript(tarPath, firstWriteMarkerPath string) string {
	return fmt.Sprintf(
		"head -c 512 %q\nuntil [ -e %q ]; do sleep 0.01; done\ntail -c +513 %q",
		tarPath, firstWriteMarkerPath, tarPath,
	)
}

func baseBackupSuccessScript(path string) string {
	return fmt.Sprintf("cat %q\n%s", path, baseBackupLsnScript)
}

func TestStreamBaseBackup(t *testing.T) {
	t.Parallel()
	t.Run("returns the manifest and lsn range when the tool streams a backup", func(t *testing.T) {
		t.Parallel()
		spec := newBaseBackupTestSpec(t)
		writeBaseBackupStub(t, spec, baseBackupSuccessScript(writeBaseBackupTar(t)))

		var output bytes.Buffer
		backup, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, &output)
		require.NoError(t, err)

		assert.Equal(t, "0/3000060", backup.StartLSN.String())
		assert.Equal(t, "0/3000220", backup.StopLSN.String())
		assert.Equal(t, 1, backup.TimelineID)
		assert.Equal(t, int64(output.Len()), backup.StreamedSize)
		assert.Contains(t, string(backup.Manifest), `"Path": "PG_VERSION"`)
	})

	t.Run("returns the backup when a write holds past the limit after the storage accepted data", func(t *testing.T) {
		t.Parallel()
		spec := newBaseBackupTestSpec(t)
		var isOutputReleased atomic.Bool
		spec.ReleaseOutput = func(error) { isOutputReleased.Store(true) }
		tarPath := writeBaseBackupTarWithFile(t, make([]byte, baseBackupFileSizeFittingAnyPipe))
		firstWriteMarkerPath := filepath.Join(t.TempDir(), "first-write-accepted")
		writeBaseBackupStub(
			t,
			spec,
			fmt.Sprintf("%s\n%s", baseBackupTwoWritesScript(tarPath, firstWriteMarkerPath), baseBackupLsnScript),
		)
		slowOutput := &secondWriteHoldingOutput{
			firstWriteMarkerPath: firstWriteMarkerPath,
			hold:                 clientToolKillAfter + 2*time.Second,
		}

		backup, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, slowOutput)

		require.NoError(t, err)
		tarContent, err := os.ReadFile(tarPath)
		require.NoError(t, err)
		assert.Equal(t, tarContent, slowOutput.received.Bytes())
		assert.Equal(t, int64(len(tarContent)), backup.StreamedSize)
		assert.False(t, isOutputReleased.Load())
	})

	t.Run("reports a stalled stream when the output accepts nothing more after the tool exits", func(t *testing.T) {
		t.Parallel()
		spec := newBaseBackupTestSpec(t)
		manualClock := &observedManualClock{source: clock.NewManual(time.Now())}
		spec.Clock = manualClock
		spec.Logger = slog.New(slog.NewTextHandler(stallLogWriter{isStalled: &manualClock.hasStalled}, nil))
		spec.StallTimeout = time.Minute
		firstWriteMarkerPath := filepath.Join(t.TempDir(), "first-write-accepted")
		stuckOutput := &stuckAfterFirstWriteOutput{
			firstWriteMarkerPath: firstWriteMarkerPath,
			isStuck:              make(chan struct{}),
			released:             make(chan struct{}),
		}
		spec.ReleaseOutput = func(error) { close(stuckOutput.released) }
		exitMarkerPath := filepath.Join(t.TempDir(), "exited")
		tarPath := writeBaseBackupTarWithFile(t, make([]byte, baseBackupFileSizeFittingAnyPipe))
		writeBaseBackupStub(
			t,
			spec,
			fmt.Sprintf("%s\ntouch %q", baseBackupTwoWritesScript(tarPath, firstWriteMarkerPath), exitMarkerPath),
		)

		completed := make(chan error, 1)
		go func() {
			_, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, stuckOutput)
			completed <- err
		}()

		testwait.Until(t, "tool exits and the output is stuck", func(context.Context) error {
			if _, err := os.Stat(exitMarkerPath); err != nil {
				return err
			}

			select {
			case <-stuckOutput.isStuck:
				return nil
			default:
				return errors.New("output is not stuck yet")
			}
		})
		waitForClockSamples(t, manualClock)
		manualClock.source.Advance(time.Minute + time.Second)

		var streamErr error
		testwait.Until(t, "watcher ends the stalled stream", func(context.Context) error {
			select {
			case streamErr = <-completed:
				return nil
			default:
				return errors.New("stream is still running")
			}
		})

		var failure *BaseBackupError
		require.ErrorAs(t, streamErr, &failure)
		assert.Equal(t, BaseBackupStalled, failure.Kind)
	})

	for _, scenario := range []struct {
		name    string
		script  string
		kind    BaseBackupErrorKind
		message string
	}{
		{"returns codec rejection when the server lacks compression", `echo 'this build does not support compression with ZSTD' >&2; exit 1`, BaseBackupCodecRejected, "does not support compression with"},
		{"keeps the exit status and rejection when the tool fails", `echo 'server rejects replication connection' >&2; exit 1`, BaseBackupToolFailed, "exit status 1; stderr: server rejects replication connection"},
		{"returns manifest corruption when the tar header is malformed", `head -c 1024 /dev/zero | tr '\0' 'x'`, BaseBackupManifestBroken, "walk tar for manifest"},
		{"returns manifest corruption when a successful stream is truncated", `printf 'partial physical backup'`, BaseBackupManifestBroken, "walk tar for manifest"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			spec := newBaseBackupTestSpec(t)
			writeBaseBackupStub(t, spec, scenario.script)

			_, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, io.Discard)
			var failure *BaseBackupError
			require.ErrorAs(t, err, &failure)
			assert.Equal(t, scenario.kind, failure.Kind)
			assert.Contains(t, failure.Error(), scenario.message)
			assert.NotContains(t, failure.Error(), spec.ClientTools.Root)
		})
	}
	t.Run("keeps the missing tool message when the installation is broken", func(t *testing.T) {
		t.Parallel()
		spec := newBaseBackupTestSpec(t)
		_, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, io.Discard)
		require.ErrorContains(
			t,
			err,
			"pg_basebackup binary not found at "+spec.ClientTools.GetExecutable(
				spec.Version,
				pgshared.ExecutablePgBasebackup,
			),
		)
	})
	for _, scenario := range []struct {
		name        string
		hasSystemID bool
		hasLsns     bool
		message     string
	}{
		{"returns manifest corruption when the system identifier is zero", false, true, "source system identifier is 0"},
		{"returns manifest corruption when the lsn range is missing", true, false, "pg_basebackup did not emit both start and end points"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			spec := newBaseBackupTestSpec(t)
			if !scenario.hasSystemID {
				spec.SystemID = 0
			}
			script := fmt.Sprintf("cat %q", writeBaseBackupTar(t))
			if scenario.hasLsns {
				script = baseBackupSuccessScript(writeBaseBackupTar(t))
			}
			writeBaseBackupStub(t, spec, script)
			_, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, io.Discard)
			var failure *BaseBackupError
			require.ErrorAs(t, err, &failure)
			assert.Equal(t, BaseBackupManifestBroken, failure.Kind)
			assert.Contains(t, failure.Error(), scenario.message)
		})
	}
	t.Run("waits ten seconds before killing a tool that ignores the interrupt", func(t *testing.T) {
		t.Parallel()
		spec := newBaseBackupTestSpec(t)
		pidPath := filepath.Join(t.TempDir(), "pid")
		writeBaseBackupStub(t, spec, fmt.Sprintf("trap '' INT\necho $$ > %q\nwhile true; do sleep 0.1; done", pidPath))

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		finished := make(chan error, 1)
		go func() { _, err := NewSource(Settings{}).StreamBaseBackup(ctx, spec, io.Discard); finished <- err }()

		toolPID := waitForToolPID(t, pidPath)
		startedAt := time.Now()
		cancel()
		require.Error(t, <-finished)
		assert.GreaterOrEqual(t, time.Since(startedAt), 10*time.Second)
		assert.Less(t, time.Since(startedAt), 15*time.Second)
		require.ErrorIs(t, syscall.Kill(toolPID, 0), syscall.ESRCH)
	})
}

func TestBaseBackupClientArguments(t *testing.T) {
	t.Parallel()
	for _, hasTunnel := range []bool{true, false} {
		scenario := "keeps the source host and hostaddr when a tunnel is used"
		if !hasTunnel {
			scenario = "omits hostaddr when no tunnel is used"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			spec := newBaseBackupTestSpec(t)
			spec.Codec = CodecGzip
			spec.IncrementalManifestPath = "/tmp/parent.manifest"
			envPath := filepath.Join(t.TempDir(), "env")
			stub := writeBaseBackupStub(t, spec, fmt.Sprintf("env > %q\nexit 1", envPath))

			settings := Settings{
				Host:     "cluster.internal",
				Port:     41000,
				Username: "replicator",
				TLS:      pgshared.TLS{Mode: pgshared.SslModeVerifyFull},
			}
			if hasTunnel {
				settings.HostAddr = "127.0.0.1"
			}
			_, _ = NewSource(settings).StreamBaseBackup(t.Context(), spec, io.Discard)
			assert.Equal(
				t,
				[]string{
					"--pgdata=-",
					"--format=tar",
					"--compress=server-gzip:6",
					"--wal-method=fetch",
					"--checkpoint=fast",
					"--incremental=/tmp/parent.manifest",
					"--no-manifest",
					"--label=test-label",
					"--no-password",
					"--verbose",
					"-h",
					"cluster.internal",
					"-p",
					"41000",
					"-U",
					"replicator",
				},
				stub.GetLastArgs(t),
			)

			env, err := os.ReadFile(envPath)
			require.NoError(t, err)
			assert.Contains(t, string(env), "PGPASSFILE=/tmp/pgpass")
			assert.Contains(t, string(env), "PGSSLMODE=verify-full")
			assert.Contains(t, string(env), "PGCONNECT_TIMEOUT=30")
			assert.Equal(t, hasTunnel, strings.Contains(string(env), "PGHOSTADDR=127.0.0.1\n"))
		})
	}
}

func waitForToolPID(t *testing.T, path string) int {
	t.Helper()
	var toolPID int
	testwait.Until(t, "tool writes its pid", func(context.Context) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		toolPID, err = strconv.Atoi(strings.TrimSpace(string(contents)))
		return err
	})
	t.Cleanup(func() { _ = syscall.Kill(toolPID, syscall.SIGKILL) })
	return toolPID
}

func TestBaseBackupTiming(t *testing.T) {
	t.Parallel()
	for _, hasBytes := range []bool{false, true} {
		scenario := "uses the first byte budget when no bytes arrive"
		if hasBytes {
			scenario = "uses the stall budget after the first byte"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			spec := newBaseBackupTestSpec(t)
			manualClock := &observedManualClock{source: clock.NewManual(time.Now())}
			spec.Clock = manualClock
			spec.Logger = slog.New(slog.NewTextHandler(stallLogWriter{isStalled: &manualClock.hasStalled}, nil))
			spec.FirstByteTimeout = 10 * time.Minute
			spec.StallTimeout = time.Minute
			pidPath := filepath.Join(t.TempDir(), "pid")

			script := fmt.Sprintf("echo $$ > %q\n", pidPath)
			if hasBytes {
				script += "printf x\n"
			}
			script += "while true; do sleep 0.1; done"
			writeBaseBackupStub(t, spec, script)

			var reports atomic.Int64
			spec.ProgressListener = func(_ float64, _ int64) { reports.Add(1) }
			completed := make(chan error, 1)
			go func() {
				_, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, io.Discard)
				completed <- err
			}()

			waitForToolPID(t, pidPath)
			if hasBytes {
				testwait.Until(t, "progress reports the streamed byte", func(context.Context) error {
					manualClock.source.Advance(time.Millisecond)
					if reports.Load() == 0 {
						return errors.New("progress not reported")
					}
					return nil
				})
				waitForClockSamples(t, manualClock)
				manualClock.source.Advance(time.Minute + time.Second)
			} else {
				manualClock.source.Advance(time.Minute + time.Second)
				waitForClockSamples(t, manualClock)
				require.False(
					t,
					manualClock.hasStalled.Load(),
					"the watcher must not cancel before the first-byte deadline",
				)
				manualClock.source.Advance(10 * time.Minute)
			}

			var streamErr error
			testwait.Until(t, "watcher ends the stalled stream", func(context.Context) error {
				select {
				case streamErr = <-completed:
					return nil
				default:
				}
				return errors.New("stream is still running")
			})
			var failure *BaseBackupError
			require.ErrorAs(t, streamErr, &failure)
			assert.Equal(t, BaseBackupStalled, failure.Kind)
		})
	}
}

// secondWriteHoldingOutput accepts one write, then holds the next for a set time, the way a busy
// storage blocks the stream after it has accepted data.
type secondWriteHoldingOutput struct {
	firstWriteMarkerPath string
	hold                 time.Duration
	writeCount           int
	received             bytes.Buffer
}

func (o *secondWriteHoldingOutput) Write(p []byte) (int, error) {
	o.writeCount++
	if o.writeCount == 2 {
		<-time.After(o.hold)
	}

	written, err := o.received.Write(p)
	if err != nil {
		return written, err
	}

	if o.writeCount == 1 {
		err = os.WriteFile(o.firstWriteMarkerPath, nil, 0o600)
	}

	return written, err
}

// stuckAfterFirstWriteOutput accepts one write, then blocks until it is released, the way a storage
// that stopped accepting data blocks the stream.
type stuckAfterFirstWriteOutput struct {
	firstWriteMarkerPath string
	writeCount           int
	isStuck              chan struct{}
	released             chan struct{}
}

func (o *stuckAfterFirstWriteOutput) Write(p []byte) (int, error) {
	o.writeCount++
	if o.writeCount == 1 {
		if err := os.WriteFile(o.firstWriteMarkerPath, nil, 0o600); err != nil {
			return 0, err
		}

		return len(p), nil
	}

	close(o.isStuck)
	<-o.released

	return 0, io.ErrClosedPipe
}

type observedManualClock struct {
	source     *clock.Manual
	samples    atomic.Int64
	hasStalled atomic.Bool
}

func (c *observedManualClock) Now() time.Time {
	now := c.source.Now()
	c.samples.Add(1)
	return now
}

func waitForClockSamples(t *testing.T, source *observedManualClock) {
	t.Helper()
	samples := source.samples.Load()
	testwait.Until(t, "watcher samples the clock twice", func(context.Context) error {
		if source.hasStalled.Load() {
			return nil
		}
		if source.samples.Load() < samples+2 {
			return errors.New("watcher has not sampled the clock")
		}
		return nil
	})
}

type stallLogWriter struct{ isStalled *atomic.Bool }

func (w stallLogWriter) Write(output []byte) (int, error) {
	if bytes.Contains(output, []byte("byte-stall timeout tripped")) {
		w.isStalled.Store(true)
	}
	return len(output), nil
}
