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
	"databasus-backend/internal/lib/toolstub"
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

	t.Run("adds Env to the environment the backend runs with", func(t *testing.T) {
		t.Parallel()

		toolOutput, err := CombinedOutput(t.Context(), Spec{
			Path: "sh",
			Args: []string{"-c", `echo "${PATH:+inherited}|$DATABASUS_PROCESS_TEST|$LC_ALL"`},
			Env:  []string{"DATABASUS_PROCESS_TEST=added", "LC_ALL=C"},
		})

		require.NoError(t, err)
		assert.Equal(t, "inherited|added|C\n", string(toolOutput))
	})
}

func TestParentDeath(t *testing.T) {
	if len(os.Args) > 2 && os.Args[len(os.Args)-2] == "process-parent-helper" {
		running, err := Start(t.Context(), Spec{Path: os.Args[len(os.Args)-1], Stdout: os.Stdout})
		require.NoError(t, err)
		_ = running.Wait()
		return
	}
	t.Parallel()
	t.Run("kills the tool when its parent dies", func(t *testing.T) {
		t.Parallel()
		toolPath := filepath.Join(t.TempDir(), "tool")
		toolstub.Write(t, toolPath, toolstub.Spec{Script: "echo $$; exec sleep 600"})
		parentOutput := &lockedBuffer{}

		parent := exec.CommandContext(
			t.Context(),
			os.Args[0],
			"-test.run=^TestParentDeath$",
			"process-parent-helper",
			toolPath,
		)
		parent.Stdout = parentOutput
		require.NoError(t, parent.Start())
		t.Cleanup(func() { _ = parent.Process.Kill() })

		var toolPID int
		testwait.Until(t, "parent reports the tool pid", func(context.Context) error {
			var err error
			toolPID, err = strconv.Atoi(strings.TrimSpace(parentOutput.String()))
			return err
		})
		t.Cleanup(func() { _ = syscall.Kill(toolPID, syscall.SIGKILL) })

		require.NoError(t, parent.Process.Kill())
		require.Error(t, parent.Wait())
		testwait.Until(t, "tool exits after its parent dies", func(context.Context) error {
			if syscall.Kill(toolPID, 0) == nil {
				return fmt.Errorf("process %d is still running", toolPID)
			}
			return nil
		})
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
