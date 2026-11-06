package backuping

import "errors"

var (
	errBackupFileNameMissing = errors.New("cannot upload metadata: file_name is empty")
	errBackupRowSuperseded   = errors.New("physical backup row is no longer in progress")
	errDatabaseNotFound      = errors.New("database not found")
	errStorageNotFound       = errors.New("storage not found")
)
