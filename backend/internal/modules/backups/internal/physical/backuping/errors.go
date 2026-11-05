package backuping

import "errors"

var (
	errBackupRowSuperseded = errors.New("physical backup row is no longer in progress")
	errDatabaseNotFound    = errors.New("database not found")
	errStorageNotFound     = errors.New("storage not found")
)
