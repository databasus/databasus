package start

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const lockFileName = "databasus-verification.lock"

func AcquireLock(ctx context.Context, log *slog.Logger) (*os.File, error) {
	f, err := os.OpenFile(lockFileName, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		if err := writePID(f); err != nil {
			_ = f.Close()
			return nil, err
		}

		log.InfoContext(ctx, "process lock acquired", slog.Any("pid", os.Getpid()), slog.Any("lock_file", lockFileName))

		return f, nil
	}

	if !errors.Is(err, syscall.EWOULDBLOCK) {
		_ = f.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	pid, pidErr := readLockPID(f)
	_ = f.Close()

	if pidErr != nil {
		return nil, fmt.Errorf("another instance is already running")
	}

	return nil, fmt.Errorf("another instance is already running (PID %d)", pid)
}

func ReleaseLock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	lockedStat, lockedErr := f.Stat()
	_ = f.Close()

	if lockedErr != nil {
		_ = os.Remove(lockFileName)
		return
	}

	diskStat, diskErr := os.Stat(lockFileName)
	if diskErr != nil {
		return
	}

	if os.SameFile(lockedStat, diskStat) {
		_ = os.Remove(lockFileName)
	}
}

func ReadLockFilePID() (int, error) {
	f, err := os.Open(lockFileName)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	return readLockPID(f)
}

func writePID(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate lock file: %w", err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek lock file: %w", err)
	}

	if _, err := fmt.Fprintf(f, "%d\n", os.Getpid()); err != nil {
		return fmt.Errorf("failed to write PID to lock file: %w", err)
	}

	return f.Sync()
}

func readLockPID(f *os.File) (int, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}

	s := strings.TrimSpace(string(data))
	if s == "" {
		return 0, errors.New("lock file is empty")
	}

	pid, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in lock file: %w", err)
	}

	return pid, nil
}

func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}

	if errors.Is(err, syscall.EPERM) {
		return true
	}

	return false
}
