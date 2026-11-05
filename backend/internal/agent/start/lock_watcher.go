package start

import (
	"context"
	"log/slog"
	"os"
	"syscall"
	"time"
)

const lockWatchInterval = 5 * time.Second

type LockWatcher struct {
	originalInode uint64
	cancel        context.CancelFunc
	log           *slog.Logger
}

func NewLockWatcher(lockFile *os.File, cancel context.CancelFunc, log *slog.Logger) (*LockWatcher, error) {
	inode, err := getFileInode(lockFile)
	if err != nil {
		return nil, err
	}

	return &LockWatcher{
		originalInode: inode,
		cancel:        cancel,
		log:           log,
	}, nil
}

func (w *LockWatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(lockWatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.check(ctx)
		}
	}
}

func (w *LockWatcher) check(ctx context.Context) {
	info, err := os.Stat(lockFileName)
	if err != nil {
		w.log.ErrorContext(
			ctx,
			"lock file disappeared, shutting down",
			slog.Any("file", lockFileName),
			slog.Any("error", err),
		)
		w.cancel()

		return
	}

	currentInode, err := getStatInode(info)
	if err != nil {
		w.log.ErrorContext(ctx, "failed to read lock file inode, shutting down", slog.Any("error", err))
		w.cancel()

		return
	}

	if currentInode != w.originalInode {
		w.log.ErrorContext(
			ctx,
			"lock file was replaced (inode changed), shutting down",
			slog.Any("original_inode", w.originalInode),
			slog.Any("current_inode", currentInode),
		)
		w.cancel()
	}
}

func getFileInode(f *os.File) (uint64, error) {
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}

	return getStatInode(info)
}

func getStatInode(info os.FileInfo) (uint64, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, os.ErrInvalid
	}

	return stat.Ino, nil
}
