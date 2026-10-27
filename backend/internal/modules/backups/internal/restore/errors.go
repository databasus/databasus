package restore

import (
	"fmt"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrRestoreNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the restore does not exist",
		Errors: nil,
	}
	ErrBackupNotCompleted = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "backupId", Detail: "must name a completed backup"},
	})
	ErrTargetOfAnotherEngine = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "target.type", Detail: "must be the engine of the backup's database"},
	})
	ErrRestoreAlreadyRunning = &apierror.Problem{
		Code:   apierror.CodeAlreadyRunning,
		Detail: "a restore of a backup of the same database is already running",
		Errors: nil,
	}
	ErrRestoreNotRunning = &apierror.Problem{
		Code:   apierror.CodeNotRunning,
		Detail: "the restore is no longer running",
		Errors: nil,
	}
)

// newTargetVersionTooOldError names both versions, which the user needs to pick another target.
func newTargetVersionTooOldError(cause error) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "target", Detail: cause.Error()},
	})
}

func newInvalidTargetTLSError(cause error) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "target.sslMode", Detail: cause.Error()},
	})
}

// newTargetConnectionError carries the reason the target's server gave, so the dialog can show it.
func newTargetConnectionError(cause error) error {
	return &apierror.Problem{
		Code:   apierror.CodeDatabaseConnectionFailed,
		Detail: cause.Error(),
		Errors: nil,
	}
}

func newInsufficientDiskSpaceError(neededBytes, freeBytes int64) error {
	return &apierror.Problem{
		Code: apierror.CodeInsufficientDiskSpace,
		Detail: fmt.Sprintf("the restore needs %d bytes of disk space for the backup's archive, but %d bytes are free",
			neededBytes, freeBytes),
		Errors: nil,
	}
}
