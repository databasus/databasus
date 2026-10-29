package verifications

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrVerificationGone     = &apierror.Problem{Code: apierror.CodeVerificationGone, Detail: "verification gone"}
	ErrVerificationNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "verification not found",
		Errors: nil,
	}
	ErrDatabaseNotFound   = &apierror.Problem{Code: apierror.CodeNotFound, Detail: "database not found", Errors: nil}
	ErrBackupNotFound     = &apierror.Problem{Code: apierror.CodeNotFound, Detail: "backup not found", Errors: nil}
	ErrBackupNotCompleted = &apierror.Problem{
		Code:   apierror.CodeBackupNotCompleted,
		Detail: "only completed backups can be verified",
		Errors: nil,
	}
	ErrAlreadyRunning = &apierror.Problem{
		Code:   apierror.CodeAlreadyRunning,
		Detail: "a manual verification is already pending or running",
		Errors: nil,
	}
	ErrNotRunning = &apierror.Problem{
		Code:   apierror.CodeNotRunning,
		Detail: "verification is not pending or running",
		Errors: nil,
	}
	errClaimConflict         = errors.New("verification claim failed: row no longer pending")
	errDatabaseNotVerifiable = errors.New("database cannot be verified")
)

func hideNotFound(err, errorForMissing error) error {
	var problem *apierror.Problem
	if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
		return errorForMissing
	}
	return err
}
