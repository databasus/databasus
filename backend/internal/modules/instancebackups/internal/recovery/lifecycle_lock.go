package recovery

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// LifecycleLock is an advisory lock on a file in the recovery directory. Every server and console
// process holds it shared while it uses the internal database; publication holds it exclusive, so
// no other process works on a database being replaced.
type LifecycleLock struct {
	file *os.File
}

func OpenLifecycleLock(directory Directory) (*LifecycleLock, error) {
	file, err := os.OpenFile(directory.getLockPath(), os.O_RDWR|os.O_CREATE, privateFileMode)
	if err != nil {
		return nil, fmt.Errorf("open the lifecycle lock: %w", err)
	}

	return &LifecycleLock{file: file}, nil
}

// TryExclusive takes the lock exclusive without waiting. It returns false while another process
// holds it in any mode.
func (l *LifecycleLock) TryExclusive() (bool, error) {
	return l.tryLock(syscall.LOCK_EX)
}

// TryShared takes the lock shared without waiting, or turns this process's exclusive hold into a
// shared one. It returns false while another process holds it exclusive.
func (l *LifecycleLock) TryShared() (bool, error) {
	return l.tryLock(syscall.LOCK_SH)
}

// Close releases the lock with its file.
func (l *LifecycleLock) Close() error {
	return l.file.Close()
}

func (l *LifecycleLock) tryLock(mode int) (bool, error) {
	err := syscall.Flock(int(l.file.Fd()), mode|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("take the lifecycle lock: %w", err)
	}

	return true, nil
}
