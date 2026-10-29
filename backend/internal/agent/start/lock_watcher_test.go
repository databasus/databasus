//go:build !windows

package start

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLockWatcher(t *testing.T) {
	t.Run("captures inode", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		lockFile, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(lockFile)

		_, cancel := context.WithCancel(t.Context())
		defer cancel()

		watcher, err := NewLockWatcher(lockFile, cancel, log)
		require.NoError(t, err)
		assert.NotZero(t, watcher.originalInode)
	})
}

func TestLockWatcher(t *testing.T) {
	t.Run("keeps the context active when file unchanged", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		lockFile, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(lockFile)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		watcher, err := NewLockWatcher(lockFile, cancel, log)
		require.NoError(t, err)

		watcher.check(t.Context())
		watcher.check(t.Context())
		watcher.check(t.Context())

		select {
		case <-ctx.Done():
			t.Fatal("context should not be cancelled when lock file is unchanged")
		default:
		}
	})
	t.Run("cancels context when file deleted", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		lockFile, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(lockFile)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		watcher, err := NewLockWatcher(lockFile, cancel, log)
		require.NoError(t, err)

		err = os.Remove(lockFileName)
		require.NoError(t, err)

		watcher.check(t.Context())

		select {
		case <-ctx.Done():
		default:
			t.Fatal("context should be cancelled when lock file is deleted")
		}
	})
	t.Run("cancels context when file replaced with different inode", func(t *testing.T) {
		setupTempDir(t)
		log := slog.New(slog.DiscardHandler)

		lockFile, err := AcquireLock(t.Context(), log)
		require.NoError(t, err)
		defer ReleaseLock(lockFile)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		watcher, err := NewLockWatcher(lockFile, cancel, log)
		require.NoError(t, err)

		err = os.Remove(lockFileName)
		require.NoError(t, err)

		err = os.WriteFile(lockFileName, []byte("99999\n"), 0o644)
		require.NoError(t, err)

		watcher.check(t.Context())

		select {
		case <-ctx.Done():
		default:
			t.Fatal("context should be cancelled when lock file inode changes")
		}
	})
}
