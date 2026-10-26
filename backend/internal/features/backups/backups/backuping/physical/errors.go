package backuping_physical

import "errors"

var (
	errBackupDecisionSuperseded = errors.New("physical backup decision superseded")
	errStorageNotFound          = errors.New("storage not found")
)
