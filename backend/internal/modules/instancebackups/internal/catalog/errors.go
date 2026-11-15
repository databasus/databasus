package catalog

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrNoDestinations = &apierror.Problem{
		Code:   apierror.CodeInvalidRequest,
		Detail: "select at least one destination before starting a Databasus backup",
		Errors: nil,
	}
	ErrAlreadyRunning = &apierror.Problem{
		Code:   apierror.CodeAlreadyRunning,
		Detail: "a Databasus backup is already being made",
		Errors: nil,
	}
	ErrBackupNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the Databasus backup does not exist",
		Errors: nil,
	}
	// ErrCopyNotFound is a copy of another backup or one that holds no stored file.
	ErrCopyNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the backup has no stored copy with this ID",
		Errors: nil,
	}
	// ErrCopyUnavailable is a stored copy whose storage was deleted, is not validated on this host
	// or no longer holds the file.
	ErrCopyUnavailable = &apierror.Problem{
		Code:   apierror.CodeStorageUnavailable,
		Detail: "the storage of the copy cannot serve its file",
		Errors: nil,
	}
	ErrTokenNotUsable = &apierror.Problem{
		Code:   apierror.CodeUnauthenticated,
		Detail: "the download token is unknown, used or expired",
		Errors: nil,
	}
	errSpoolFileMissing = errors.New("the Databasus backup spool file is missing")
)

func newInvalidCursorProblem() *apierror.Problem {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: "after", Detail: "the cursor is not valid"}})
}
