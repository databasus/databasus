//go:build !windows

package start

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireLock(t *testing.T) {
	t.Run("lock file created with pid", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		lockFile, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(lockFile)

		data, err := os.ReadFile(lockFileName)
		require.NoError(t, err)

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		require.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})
	t.Run("second acquire fails when first held", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		first, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(first)

		second, err := AcquireLock(t.Context(), log)
		assert.Nil(t, second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "another instance is already running")
		assert.Contains(t, err.Error(), fmt.Sprintf("PID %d", os.Getpid()))
	})
	t.Run("stale lock reacquired when process dead", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		err := os.WriteFile(lockFileName, []byte("999999999\n"), 0o644)
		require.NoError(t, err)

		lockFile, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(lockFile)

		data, err := os.ReadFile(lockFileName)
		require.NoError(t, err)

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		require.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})
	t.Run("reacquired after release", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		first, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		ReleaseLock(first)

		second, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(second)

		data, err := os.ReadFile(lockFileName)
		require.NoError(t, err)

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		require.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})
}

func TestReleaseLock(t *testing.T) {
	t.Run("lock file removed", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		lockFile, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)

		ReleaseLock(lockFile)

		_, err = os.Stat(lockFileName)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestIsProcessAlive(t *testing.T) {
	t.Run("returns true for self", func(t *testing.T) {
		assert.True(t, isProcessAlive(os.Getpid()))
	})
	t.Run("returns false for non existent pid", func(t *testing.T) {
		assert.False(t, isProcessAlive(999999999))
	})
}

func TestReadLockPID(t *testing.T) {
	t.Run("parses valid pid", func(t *testing.T) {
		setupTempDir(t)

		f, err := os.CreateTemp(t.TempDir(), "lock-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })

		_, err = f.WriteString("12345\n")
		require.NoError(t, err)

		pid, err := readLockPID(f)
		require.NoError(t, err)
		assert.Equal(t, 12345, pid)
	})
	t.Run("returns error for empty file", func(t *testing.T) {
		setupTempDir(t)

		f, err := os.CreateTemp(t.TempDir(), "lock-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })

		_, err = readLockPID(f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lock file is empty")
	})
}

func setupTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)
	return dir
}
