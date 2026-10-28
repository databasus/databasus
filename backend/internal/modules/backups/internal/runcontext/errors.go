package runcontext

import (
	"errors"
	"fmt"
)

var (
	ErrBackupShutdown  = errors.New("backup interrupted by application shutdown")
	ErrBackupTimeout   = fmt.Errorf("backup exceeded the time limit of %d hours", int(LogicalTimeLimit.Hours()))
	ErrBackupCancelled = errors.New("backup cancelled")

	ErrRestoreShutdown  = errors.New("restore interrupted by application shutdown")
	ErrRestoreTimeout   = fmt.Errorf("restore exceeded the time limit of %d hours", int(LogicalTimeLimit.Hours()))
	ErrRestoreCancelled = errors.New("restore cancelled")

	errCancelRequested = errors.New("cancel requested")
	errRunEnded        = errors.New("run ended")
)
