package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestStart(t *testing.T) {
	t.Parallel()

	t.Run("kills a child the tool started when the context is cancelled", func(t *testing.T) {
		t.Parallel()
		childPidFile := filepath.Join(t.TempDir(), "child.pid")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		running, err := Start(ctx, Spec{
			Path: "sh",
			Args: []string{"-c", fmt.Sprintf("sleep 600 & echo $! > %s; wait", childPidFile)},
		})
		require.NoError(t, err)

		childPid := waitForPidFile(t, childPidFile)
		cancel()
		waitErr := running.Wait()

		require.ErrorIs(t, waitErr, context.Canceled)
		testwait.Until(t, "child process exits", func(context.Context) error {
			if syscall.Kill(childPid, 0) == nil {
				return fmt.Errorf("process %d is still running", childPid)
			}

			return nil
		})
	})

	t.Run("lets the tool exit with its own status when it handles the stop signal", func(t *testing.T) {
		t.Parallel()
		toolOutput := &lockedBuffer{}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		running, err := Start(ctx, Spec{
			Path:       "sh",
			Args:       []string{"-c", `trap 'echo stopped; exit 7' TERM; echo ready; while true; do sleep 0.1; done`},
			Stdout:     toolOutput,
			StopSignal: syscall.SIGTERM,
			KillAfter:  time.Minute,
		})
		require.NoError(t, err)

		waitForOutput(t, toolOutput, "ready")
		cancel()
		waitErr := running.Wait()

		require.ErrorIs(t, waitErr, context.Canceled)
		var exitErr *exec.ExitError
		require.ErrorAs(t, waitErr, &exitErr)
		assert.Equal(t, 7, exitErr.ExitCode())
		assert.Contains(t, toolOutput.String(), "stopped")
	})

	t.Run("kills the tool after the grace period when it ignores the stop signal", func(t *testing.T) {
		t.Parallel()
		toolOutput := &lockedBuffer{}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		running, err := Start(ctx, Spec{
			Path:       "sh",
			Args:       []string{"-c", `trap '' TERM; echo ready; while true; do sleep 0.1; done`},
			Stdout:     toolOutput,
			StopSignal: syscall.SIGTERM,
			KillAfter:  200 * time.Millisecond,
		})
		require.NoError(t, err)

		waitForOutput(t, toolOutput, "ready")
		cancel()
		waitErr := running.Wait()

		require.ErrorIs(t, waitErr, context.Canceled)
		var exitErr *exec.ExitError
		require.ErrorAs(t, waitErr, &exitErr)
		waitStatus, isWaitStatus := exitErr.Sys().(syscall.WaitStatus)
		require.True(t, isWaitStatus)
		assert.Equal(t, syscall.SIGKILL, waitStatus.Signal())
	})
}

func TestWait(t *testing.T) {
	t.Parallel()

	t.Run("returns the exit error without a cancellation when the tool fails", func(t *testing.T) {
		t.Parallel()

		waitErr := Run(t.Context(), Spec{Path: "sh", Args: []string{"-c", "exit 3"}})

		require.Error(t, waitErr)
		require.NotErrorIs(t, waitErr, context.Canceled)
		var exitErr *exec.ExitError
		require.ErrorAs(t, waitErr, &exitErr)
		assert.Equal(t, 3, exitErr.ExitCode())
	})

	t.Run("returns nil when the tool succeeds", func(t *testing.T) {
		t.Parallel()

		toolOutput, err := CombinedOutput(t.Context(), Spec{Path: "sh", Args: []string{"-c", "echo out; echo err >&2"}})

		require.NoError(t, err)
		assert.Equal(t, "out\nerr\n", string(toolOutput))
	})
}

func waitForPidFile(t *testing.T, path string) int {
	t.Helper()

	var pid int

	testwait.Until(t, "tool writes the child's pid", func(context.Context) error {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		pid, err = strconv.Atoi(strings.TrimSpace(string(content)))

		return err
	})

	return pid
}

func waitForOutput(t *testing.T, output *lockedBuffer, line string) {
	t.Helper()

	testwait.Until(t, "tool prints "+line, func(context.Context) error {
		if !strings.Contains(output.String(), line) {
			return errors.New("not printed yet")
		}

		return nil
	})
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buffer.String()
}
