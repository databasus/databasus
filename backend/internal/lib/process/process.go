// Package process starts external tools bound to the operation's context. A tool runs in its own
// process group, so ending the context stops the tool together with every child it started.
package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Spec struct {
	Path string
	Args []string
	// Env is added to the environment the backend runs with; a variable set in both takes the
	// value from Env.
	Env    []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// StopSignal is sent to the process group when ctx ends; nil sends SIGKILL at once.
	StopSignal os.Signal
	// KillAfter is how long the group may take to exit after StopSignal before SIGKILL. It also
	// bounds how long Wait waits for more output from pipes that a child keeps open after the tool
	// exited. The time Stdout takes to accept output does not count against it.
	KillAfter time.Duration
}

// Running is a started tool and its process group.
type Running struct {
	ctx         context.Context
	cmd         *exec.Cmd
	outputDrain *outputDrain

	mu        sync.Mutex
	killTimer *time.Timer
}

func Start(ctx context.Context, spec Spec) (*Running, error) {
	// Callers select an executable and separate arguments; no shell interprets the command.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	if len(spec.Env) > 0 {
		cmd.Env = append(cmd.Environ(), spec.Env...)
	}
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr

	drain, err := attachOutputDrain(cmd, spec)
	if err != nil {
		return nil, err
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	cmd.WaitDelay = spec.KillAfter

	running := &Running{ctx: ctx, cmd: cmd, outputDrain: drain}
	cmd.Cancel = func() error { return running.stopGroup(spec) }

	if err := cmd.Start(); err != nil {
		if drain != nil {
			drain.closeBothEnds()
		}

		return nil, fmt.Errorf("start %s: %w", spec.Path, err)
	}

	if drain != nil {
		drain.start(ctx)
	}

	return running, nil
}

// Wait waits for the tool to exit. When ctx ended the tool, the error wraps ctx's error, together
// with the tool's exit error when it exited with a failure, so a caller can tell a cancellation
// from a failed tool with errors.Is.
func (r *Running) Wait() error {
	waitErr := r.cmd.Wait()

	r.mu.Lock()
	if r.killTimer != nil {
		r.killTimer.Stop()
	}
	r.mu.Unlock()

	drainErr := r.finishOutputDrain(waitErr)

	ctxErr := r.ctx.Err()
	if ctxErr == nil {
		if waitErr != nil {
			return fmt.Errorf("run %s: %w", r.cmd.Path, waitErr)
		}

		if drainErr != nil {
			return fmt.Errorf("run %s: %w", r.cmd.Path, drainErr)
		}

		return nil
	}

	// The group may already be gone; the kill only makes sure no child outlives the context.
	_ = r.signalGroup(syscall.SIGKILL)

	if exitErr, isExitErr := errors.AsType[*exec.ExitError](waitErr); isExitErr {
		return fmt.Errorf("run %s: %w: %w", r.cmd.Path, ctxErr, exitErr)
	}

	return fmt.Errorf("run %s: %w", r.cmd.Path, ctxErr)
}

func Run(ctx context.Context, spec Spec) error {
	running, err := Start(ctx, spec)
	if err != nil {
		return err
	}

	return running.Wait()
}

// CombinedOutput runs the tool and returns what it wrote to stdout and stderr together; it ignores
// Stdout and Stderr of spec.
func CombinedOutput(ctx context.Context, spec Spec) ([]byte, error) {
	var output bytes.Buffer
	spec.Stdout = &output
	spec.Stderr = &output

	err := Run(ctx, spec)

	return output.Bytes(), err
}

// finishOutputDrain waits for the writer to take the remaining output. exec.ErrWaitDelay means a
// descendant also holds Stderr and has used up the limit already, so the pipe is closed at once.
func (r *Running) finishOutputDrain(waitErr error) error {
	if r.outputDrain == nil {
		return nil
	}

	if errors.Is(waitErr, exec.ErrWaitDelay) {
		r.outputDrain.closeReader()
	} else {
		r.outputDrain.startLimitingReads()
	}

	select {
	case <-r.outputDrain.done:
	case <-r.ctx.Done():
		// A descendant in the group may still hold the pipe; the kill lets the copy see EOF.
		_ = r.signalGroup(syscall.SIGKILL)
		<-r.outputDrain.done
	}

	return r.outputDrain.copyErr
}

func (r *Running) stopGroup(spec Spec) error {
	if spec.StopSignal == nil {
		return r.signalGroup(syscall.SIGKILL)
	}

	r.mu.Lock()
	r.killTimer = time.AfterFunc(spec.KillAfter, func() { _ = r.signalGroup(syscall.SIGKILL) })
	r.mu.Unlock()

	return r.signalGroup(spec.StopSignal)
}

// signalGroup sends signal to the negative process ID, which reaches every process of the group.
func (r *Running) signalGroup(signal os.Signal) error {
	unixSignal, isUnixSignal := signal.(syscall.Signal)
	if !isUnixSignal {
		return fmt.Errorf("%w: %v", ErrUnsupportedSignal, signal)
	}

	err := syscall.Kill(-r.cmd.Process.Pid, unixSignal)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}

	return err
}
