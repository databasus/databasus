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
	"sync/atomic"
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
	t.Parallel()

	if len(os.Args) > 2 && os.Args[len(os.Args)-2] == "process-parent-helper" {
		running, err := Start(t.Context(), Spec{Path: os.Args[len(os.Args)-1], Stdout: os.Stdout})
		require.NoError(t, err)
		_ = running.Wait()
		return
	}

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

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("waits for a slow stdout writer after the tool exits", func(t *testing.T) {
		t.Parallel()
		slowOutput := &holdingWriter{hold: time.Second}

		err := Run(t.Context(), Spec{
			Path:      "sh",
			Args:      []string{"-c", "printf dump-output"},
			Stdout:    slowOutput,
			KillAfter: 200 * time.Millisecond,
		})

		require.NoError(t, err)
		assert.Equal(t, "dump-output", slowOutput.String())
	})

	t.Run("returns the exit error when the tool fails while the writer is slow", func(t *testing.T) {
		t.Parallel()
		slowOutput := &holdingWriter{hold: time.Second}

		err := Run(t.Context(), Spec{
			Path:      "sh",
			Args:      []string{"-c", "printf dump-output; exit 3"},
			Stdout:    slowOutput,
			KillAfter: 200 * time.Millisecond,
		})

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 3, exitErr.ExitCode())
		assert.Equal(t, "dump-output", slowOutput.String())
	})

	t.Run("stops waiting for a descendant that holds the pipe after cancellation", func(t *testing.T) {
		t.Parallel()
		toolOutput := &lockedBuffer{}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		running, err := Start(ctx, Spec{
			Path:      "sh",
			Args:      []string{"-c", `setsid sh -c 'echo $$; exec sleep 600' & wait`},
			Stdout:    toolOutput,
			KillAfter: 200 * time.Millisecond,
		})
		require.NoError(t, err)

		killDescendantAfterTest(t, toolOutput)
		cancel()
		waitErr := running.Wait()

		require.ErrorIs(t, waitErr, context.Canceled)
	})

	t.Run("returns the wait delay error when a descendant holds the pipe after the tool exits", func(t *testing.T) {
		t.Parallel()
		toolOutput := &lockedBuffer{}

		running, err := Start(t.Context(), Spec{
			Path:      "sh",
			Args:      []string{"-c", "sleep 600 & echo $!"},
			Stdout:    toolOutput,
			KillAfter: 200 * time.Millisecond,
		})
		require.NoError(t, err)

		killDescendantAfterTest(t, toolOutput)
		waitErr := running.Wait()

		require.ErrorIs(t, waitErr, exec.ErrWaitDelay)
	})

	t.Run("returns the wait delay error once when a descendant holds both stdout and stderr", func(t *testing.T) {
		t.Parallel()
		toolOutput := &lockedBuffer{}
		killAfter := 2 * time.Second
		startedAt := time.Now()

		running, err := Start(t.Context(), Spec{
			Path:      "sh",
			Args:      []string{"-c", "sleep 600 & echo $!"},
			Stdout:    toolOutput,
			Stderr:    &lockedBuffer{},
			KillAfter: killAfter,
		})
		require.NoError(t, err)

		killDescendantAfterTest(t, toolOutput)
		waitErr := running.Wait()

		require.ErrorIs(t, waitErr, exec.ErrWaitDelay)
		assert.Less(t, time.Since(startedAt), 2*killAfter)
	})

	t.Run("returns the cancellation only after a blocked writer has returned", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		blockedOutput := &contextBlockedWriter{ctx: ctx, hasEntered: make(chan struct{})}

		running, err := Start(ctx, Spec{
			Path:      "sh",
			Args:      []string{"-c", "printf dump-output"},
			Stdout:    blockedOutput,
			KillAfter: 200 * time.Millisecond,
		})
		require.NoError(t, err)

		testwait.Until(t, "writer receives the output", func(context.Context) error {
			select {
			case <-blockedOutput.hasEntered:
				return nil
			default:
				return errors.New("writer is not called yet")
			}
		})

		cancel()
		waitErr := running.Wait()

		assert.True(t, blockedOutput.hasReturned.Load())
		require.ErrorIs(t, waitErr, context.Canceled)
		require.NotErrorIs(t, waitErr, errWriterFailed)
	})

	t.Run("stops a descendant that holds the pipe with a zero limit after cancellation", func(t *testing.T) {
		t.Parallel()
		toolOutput := &lockedBuffer{}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		running, err := Start(ctx, Spec{Path: "sh", Args: []string{"-c", "sleep 600 & echo $!"}, Stdout: toolOutput})
		require.NoError(t, err)

		killDescendantAfterTest(t, toolOutput)

		waited := make(chan error, 1)
		go func() { waited <- running.Wait() }()

		testwait.Until(t, "tool exits", func(context.Context) error {
			if syscall.Kill(running.cmd.Process.Pid, 0) == nil {
				return errors.New("tool is still running")
			}

			return nil
		})

		cancel()

		var waitErr error
		testwait.Until(t, "wait returns after the cancellation", func(context.Context) error {
			select {
			case waitErr = <-waited:
				return nil
			default:
				return errors.New("wait has not returned")
			}
		})

		require.ErrorIs(t, waitErr, context.Canceled)
	})

	t.Run("stops the tool when the writer fails", func(t *testing.T) {
		t.Parallel()

		err := Run(t.Context(), Spec{
			Path:      "sh",
			Args:      []string{"-c", "yes"},
			Stdout:    failingWriter{},
			KillAfter: time.Minute,
		})

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
	})

	t.Run("returns the writer error when the tool succeeds", func(t *testing.T) {
		t.Parallel()

		err := Run(t.Context(), Spec{
			Path:      "sh",
			Args:      []string{"-c", "printf dump-output"},
			Stdout:    failingWriter{},
			KillAfter: time.Minute,
		})

		require.ErrorIs(t, err, errWriterFailed)
	})

	t.Run("keeps the order of stdout and stderr when both use one writer", func(t *testing.T) {
		t.Parallel()
		var toolOutput bytes.Buffer

		err := Run(t.Context(), Spec{
			Path:   "sh",
			Args:   []string{"-c", "echo first; echo second >&2; echo third; echo fourth >&2"},
			Stdout: &toolOutput,
			Stderr: &toolOutput,
		})

		require.NoError(t, err)
		assert.Equal(t, "first\nsecond\nthird\nfourth\n", toolOutput.String())
	})

	t.Run("waits for a slow writer with a zero limit", func(t *testing.T) {
		t.Parallel()
		slowOutput := &holdingWriter{hold: 300 * time.Millisecond}

		err := Run(t.Context(), Spec{Path: "sh", Args: []string{"-c", "printf dump-output"}, Stdout: slowOutput})

		require.NoError(t, err)
		assert.Equal(t, "dump-output", slowOutput.String())
	})

	t.Run("runs the tool with writers of an uncomparable type", func(t *testing.T) {
		t.Parallel()
		toolOutput := uncomparableWriter{sink: &lockedBuffer{}, tags: []string{"output"}}

		err := Run(t.Context(), Spec{
			Path:   "sh",
			Args:   []string{"-c", "printf dump-output"},
			Stdout: toolOutput,
			Stderr: toolOutput,
		})

		require.NoError(t, err)
		assert.Equal(t, "dump-output", toolOutput.sink.String())
	})

	t.Run("returns the start error when the tool cannot start with a stdout writer", func(t *testing.T) {
		t.Parallel()

		_, err := Start(t.Context(), Spec{Path: filepath.Join(t.TempDir(), "missing"), Stdout: &lockedBuffer{}})

		require.Error(t, err)
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

// killDescendantAfterTest reads the pid the tool printed, since nothing else ends a descendant that
// outlives its tool.
func killDescendantAfterTest(t *testing.T, toolOutput *lockedBuffer) {
	t.Helper()

	var descendantPid int

	testwait.Until(t, "tool prints the descendant's pid", func(context.Context) error {
		var err error
		descendantPid, err = strconv.Atoi(strings.TrimSpace(toolOutput.String()))

		return err
	})
	t.Cleanup(func() { _ = syscall.Kill(descendantPid, syscall.SIGKILL) })
}

// holdingWriter holds its first write for a set time, the way a busy storage blocks the dump.
type holdingWriter struct {
	hold   time.Duration
	isHeld atomic.Bool
	output lockedBuffer
}

func (w *holdingWriter) Write(p []byte) (int, error) {
	if w.isHeld.CompareAndSwap(false, true) {
		<-time.After(w.hold)
	}

	return w.output.Write(p)
}

func (w *holdingWriter) String() string {
	return w.output.String()
}

// contextBlockedWriter blocks until its context ends and takes a moment more to return, so a Wait
// that stopped waiting for it would return first.
type contextBlockedWriter struct {
	ctx         context.Context
	hasEntered  chan struct{}
	hasReturned atomic.Bool
}

func (w *contextBlockedWriter) Write([]byte) (int, error) {
	close(w.hasEntered)
	<-w.ctx.Done()
	<-time.After(200 * time.Millisecond)
	w.hasReturned.Store(true)

	return 0, errWriterFailed
}

var errWriterFailed = errors.New("writer failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWriterFailed
}

// uncomparableWriter panics when two values of it are compared, because of its slice field.
type uncomparableWriter struct {
	sink *lockedBuffer
	tags []string
}

func (w uncomparableWriter) Write(p []byte) (int, error) {
	return w.sink.Write(p)
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
