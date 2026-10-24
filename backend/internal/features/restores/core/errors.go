package restores_core

import "errors"

var ErrRestoreShutdown = errors.New("restore interrupted by application shutdown")
