package restores_core

import "errors"

var (
	ErrRestoreShutdown         = errors.New("restore interrupted by application shutdown")
	ErrRestoreTypeNotSupported = errors.New("database type not supported")
	ErrRestoreTargetMissing    = errors.New("restore request has no target database of the backup's type")
)
