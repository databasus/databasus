package backuping_physical

import "errors"

var (
	errBackupDecisionSuperseded = errors.New("physical backup decision superseded")
	errBackupRowSuperseded      = errors.New("physical backup row is no longer in progress")
	errStorageNotFound          = errors.New("storage not found")
)
