package start

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	logFileName        = "databasus-verification.daemon.log"
	stopTimeout        = 30 * time.Second
	stopPollInterval   = 500 * time.Millisecond
	daemonStartupDelay = 500 * time.Millisecond
)

func Stop(ctx context.Context, log *slog.Logger) error {
	pid, err := ReadLockFilePID()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("agent is not running (no lock file found)")
		}

		return fmt.Errorf("failed to read lock file: %w", err)
	}

	if !isProcessAlive(pid) {
		_ = os.Remove(lockFileName)
		return fmt.Errorf("agent is not running (stale lock file removed, PID %d)", pid)
	}

	log.InfoContext(ctx, "sending SIGTERM to agent", slog.Any("pid", pid))

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to PID %d: %w", pid, err)
	}

	if err := pollProcess(
		ctx,
		processPoll{pid: pid, duration: stopTimeout, interval: stopPollInterval, shouldBeAlive: false},
	); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("agent (PID %d) did not stop within %s — process may be stuck", pid, stopTimeout)
		}
		return err
	}
	log.InfoContext(ctx, "agent stopped", slog.Int("pid", pid))
	return nil
}

func GetDaemonStatus() (string, error) {
	pid, err := ReadLockFilePID()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "Agent is not running", nil
		}
		return "", fmt.Errorf("failed to read lock file: %w", err)
	}
	if isProcessAlive(pid) {
		return fmt.Sprintf("Agent is running (PID %d)", pid), nil
	}
	_ = os.Remove(lockFileName)
	return "Agent is not running (stale lock file)", nil
}

func spawnDaemon(ctx context.Context, log *slog.Logger) (int, error) {
	execPath, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	args := []string{"_run"}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file %s: %w", logFileName, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		_ = logFile.Close()
		return 0, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Starts this agent's own executable in the background, not a user-supplied command.
	//nolint:forbidigo // The detached daemon must outlive its parent; lib/process kills tools with their parent.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(context.WithoutCancel(ctx), execPath, args...)
	cmd.Dir = cwd
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return 0, fmt.Errorf("failed to start daemon process: %w", err)
	}

	pid := cmd.Process.Pid

	// Detach — we don't wait for the child
	_ = logFile.Close()

	if err := pollProcess(
		ctx,
		processPoll{pid: pid, duration: daemonStartupDelay, interval: 10 * time.Millisecond, shouldBeAlive: true},
	); err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			return 0, err
		}
		return 0, fmt.Errorf("daemon process (PID %d) exited immediately — check %s for details", pid, logFileName)
	}

	log.InfoContext(ctx, "daemon spawned", slog.Any("pid", pid), slog.Any("log", logFileName))

	return pid, nil
}

type processPoll struct {
	pid           int
	duration      time.Duration
	interval      time.Duration
	shouldBeAlive bool
}

func pollProcess(ctx context.Context, spec processPoll) error {
	ticker := time.NewTicker(spec.interval)
	defer ticker.Stop()
	deadline := time.NewTimer(spec.duration)
	defer deadline.Stop()

	for {
		isAlive := isProcessAlive(spec.pid)
		if !isAlive {
			if spec.shouldBeAlive {
				return os.ErrProcessDone
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			if spec.shouldBeAlive {
				if !isProcessAlive(spec.pid) {
					return os.ErrProcessDone
				}
				return nil
			}
			if !isProcessAlive(spec.pid) {
				return nil
			}
			return context.DeadlineExceeded
		case <-ticker.C:
		}
	}
}
