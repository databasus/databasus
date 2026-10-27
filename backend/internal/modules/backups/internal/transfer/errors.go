package transfer

import "databasus-backend/internal/lib/apierror"

var (
	ErrDatabaseNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the database does not exist",
		Errors: nil,
	}
	ErrDatabaseHasNoStorage = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "storage.type", Detail: "the database has no storage to move with it"},
	})
	ErrTargetStorageNotInTargetWorkspace = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "storage.storageId", Detail: "must name a storage of the target workspace"},
	})
)
