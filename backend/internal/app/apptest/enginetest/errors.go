package enginetest

import "errors"

var (
	errBackupRunning  = errors.New("the logical backup is still in progress")
	errRestoreRunning = errors.New("the restore is still in progress")
)
