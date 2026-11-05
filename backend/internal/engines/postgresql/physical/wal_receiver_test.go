package physical

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/toolstub"
)

func newWalReceiverTestSpec(t *testing.T) WalReceiverSpec {
	t.Helper()
	return WalReceiverSpec{
		ClientTools:     pgshared.ClientTools{Root: t.TempDir()},
		Version:         pgshared.Version17,
		ApplicationName: WalReceiverApplicationNamePrefix + "test",
		WatchDir:        t.TempDir(),
		SlotName:        "slot",
		Clock:           clock.NewSystem(),
		Logger:          slog.New(slog.DiscardHandler),
	}
}

func TestReceiveWal(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name        string
		stderr      string
		disposition WalReceiverDisposition
	}{
		{"returns fatal when the disk is full", "could not write 16777216 bytes to WAL file: No space left on device", WalReceiverFatal},
		{"returns fatal when authentication fails", `FATAL: password authentication failed for user "repl"`, WalReceiverFatal},
		{"returns fatal when pg hba rejects the connection", "FATAL: no pg_hba.conf entry for replication connection", WalReceiverFatal},
		{"returns fatal when writing is denied", "could not create archive status file: Permission denied", WalReceiverFatal},
		{"returns fatal when another receiver holds the slot", `replication slot "db_slot" is active for PID 4242`, WalReceiverFatal},
		{"returns resume mismatch when the segment was recycled", "requested WAL segment has already been removed", WalReceiverResumeMismatch},
		{"returns retryable when the connection closes", "server closed the connection unexpectedly", WalReceiverRetryable},
		{"returns retryable when the connection is refused", "connection to server failed: Connection refused", WalReceiverRetryable},
		{"returns retryable when error output is empty", "", WalReceiverRetryable},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			spec := newWalReceiverTestSpec(t)
			toolstub.Write(
				t,
				spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
				toolstub.Spec{Script: fmt.Sprintf("echo '%s' >&2; exit 1", scenario.stderr)},
			)

			receiver := NewSource(Settings{}).ReceiveWal(t.Context(), spec)
			assert.Equal(t, scenario.disposition, receiver.Disposition)
			if scenario.disposition == WalReceiverFatal {
				require.ErrorContains(
					t,
					receiver.FatalErr,
					"pg_receivewal fatal error: exit status 1; stderr: "+scenario.stderr,
				)
				assert.NotContains(t, receiver.FatalErr.Error(), spec.ClientTools.Root)
			}
		})
	}
	t.Run("captures the rejection when stderr arrives after the receiver exits", func(t *testing.T) {
		t.Parallel()
		spec := newWalReceiverTestSpec(t)
		toolstub.Write(
			t,
			spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
			toolstub.Spec{Script: "(sleep 0.2; echo 'FATAL: password authentication failed' >&2) >/dev/null &\nexit 1"},
		)
		receiver := NewSource(Settings{}).ReceiveWal(t.Context(), spec)
		assert.Equal(t, WalReceiverFatal, receiver.Disposition)
		require.ErrorContains(t, receiver.FatalErr, "password authentication failed")
	})
	t.Run("waits ten seconds before killing a receiver that ignores the interrupt", func(t *testing.T) {
		t.Parallel()
		spec := newWalReceiverTestSpec(t)
		pidPath := filepath.Join(t.TempDir(), "pid")
		toolstub.Write(
			t,
			spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
			toolstub.Spec{Script: fmt.Sprintf("trap '' INT\necho $$ > %q\nwhile true; do sleep 0.1; done", pidPath)},
		)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		completed := make(chan WalReceiverExit, 1)
		go func() { completed <- NewSource(Settings{}).ReceiveWal(ctx, spec) }()
		receiverPID := waitForToolPID(t, pidPath)

		startedAt := time.Now()
		cancel()
		receiver := <-completed
		assert.Equal(t, WalReceiverStopped, receiver.Disposition)
		assert.GreaterOrEqual(t, time.Since(startedAt), 10*time.Second)
		assert.Less(t, time.Since(startedAt), 15*time.Second)
		require.ErrorIs(t, syscall.Kill(receiverPID, 0), syscall.ESRCH)
	})
}

func TestWalReceiverClientArguments(t *testing.T) {
	t.Parallel()
	for _, hasTunnel := range []bool{true, false} {
		scenario := "keeps hostaddr and the source host when a tunnel is used"
		if !hasTunnel {
			scenario = "omits hostaddr when no tunnel is used"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			spec := newWalReceiverTestSpec(t)
			envPath := filepath.Join(t.TempDir(), "env")
			stub := toolstub.Write(
				t,
				spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgReceivewal),
				toolstub.Spec{Script: fmt.Sprintf("env > %q", envPath)},
			)

			settings := Settings{Host: "cluster.internal", Port: 41000, Username: "replicator"}
			if hasTunnel {
				settings.HostAddr = "127.0.0.1"
			}
			receiver := NewSource(settings).ReceiveWal(t.Context(), spec)
			assert.Equal(t, WalReceiverRetryable, receiver.Disposition)
			assert.Equal(
				t,
				[]string{
					"--directory=" + spec.WatchDir,
					"--slot=slot",
					"--no-loop",
					"--synchronous",
					"--verbose",
					"--no-password",
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
			assert.Contains(t, string(env), "PGAPPNAME="+spec.ApplicationName)
			assert.Contains(t, string(env), "PGSSLMODE=disable")
			if hasTunnel {
				assert.Contains(t, string(env), "PGHOSTADDR=127.0.0.1")
			} else {
				assert.NotContains(t, string(env), "PGHOSTADDR=")
			}
		})
	}
}
