package configs

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var errConfigMissing = errors.New("the Databasus backup settings row is missing")

// ErrDatabasusBackupInUse refuses deleting a storage, or a workspace holding one, while root admin
// has it selected as a Databasus backup destination.
var ErrDatabasusBackupInUse = &apierror.Problem{
	Code: apierror.CodeDatabasusBackupInUse,
	Detail: "the storage is used for Databasus backups; remove it from the Databasus backup settings " +
		"first, or ask the server administrator to do it",
	Errors: nil,
}

func newInvalidStorageIDsProblem(detail string) *apierror.Problem {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: "storageIds", Detail: detail}})
}

func newInvalidScheduleProblem(detail string) *apierror.Problem {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: "schedule", Detail: detail}})
}

func newInvalidRetentionProblem() *apierror.Problem {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "retentionDays", Detail: "the retention must be between 1 and 3650 days"},
	})
}

func newInvalidCursorProblem() *apierror.Problem {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: "after", Detail: "the cursor is not valid"}})
}
