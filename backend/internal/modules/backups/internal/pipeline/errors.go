package pipeline

import (
	"errors"
	"fmt"
)

var (
	ErrBackupShutdown  = errors.New("backup interrupted by application shutdown")
	ErrBackupTimeout   = fmt.Errorf("backup exceeded the time limit of %d hours", int(runTimeLimit.Hours()))
	ErrBackupCancelled = errors.New("backup cancelled")

	ErrRestoreShutdown  = errors.New("restore interrupted by application shutdown")
	ErrRestoreTimeout   = fmt.Errorf("restore exceeded the time limit of %d hours", int(runTimeLimit.Hours()))
	ErrRestoreCancelled = errors.New("restore cancelled")

	// ErrNotLogicalDatabase answers a logical operation on a database type the registry has no
	// engine for, which is the physical PostgreSQL database.
	ErrNotLogicalDatabase = errors.New("the database has no logical backups")
	// ErrTargetOfAnotherEngine answers a restore target whose engine is not the backup's.
	ErrTargetOfAnotherEngine = errors.New("the restore target is not of the backup's engine")
	// ErrTargetVersionTooOld answers a restore target whose server cannot take a backup of a newer
	// server version.
	ErrTargetVersionTooOld = errors.New("the restore target's server version is lower than the backup's")

	errCancelRequested = errors.New("cancel requested")
	errRunEnded        = errors.New("run ended")

	errBackupStreamFailed = errors.New(
		"backup stream from storage failed mid-restore, target database is left partially restored",
	)
)

// TargetConnectionError is a restore target that could not be reached or that refused the
// credentials, with the reason the target's server gave.
type TargetConnectionError struct {
	Err error
}

func (e *TargetConnectionError) Error() string {
	return "connect to the restore target: " + e.Err.Error()
}

func (e *TargetConnectionError) Unwrap() error {
	return e.Err
}

// InvalidTargetTLSError is a PostgreSQL restore target whose SSL settings do not fit together.
type InvalidTargetTLSError struct {
	Err error
}

func (e *InvalidTargetTLSError) Error() string {
	return "the SSL settings of the restore target are invalid: " + e.Err.Error()
}

func (e *InvalidTargetTLSError) Unwrap() error {
	return e.Err
}
