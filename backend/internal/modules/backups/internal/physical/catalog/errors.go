package catalog

import "errors"

var errBackupSuperseded = errors.New("physical backup is no longer running")
