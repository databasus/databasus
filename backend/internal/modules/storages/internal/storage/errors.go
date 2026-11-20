package storage

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrCleanupPending = &apierror.Problem{
		Code:   apierror.CodeCleanupPending,
		Detail: "automatic file cleanup is still pending",
		Errors: nil,
	}
	ErrStorageNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the storage does not exist",
		Errors: nil,
	}
	ErrStorageInUse = &apierror.Problem{
		Code: apierror.CodeInUse,
		Detail: "a backup config uses the storage or it still holds backups; remove them first, " +
			"they are the only record of the files it holds",
		Errors: nil,
	}
	// ErrWorkspaceStorageInUse names no backup and no other workspace, so a refused workspace
	// deletion reveals nothing its owner may not see.
	ErrWorkspaceStorageInUse = &apierror.Problem{
		Code:   apierror.CodeInUse,
		Detail: "a storage of the workspace is still in use",
		Errors: nil,
	}
	ErrInvalidName = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "name", Detail: "must be between 1 and 255 characters long"},
	})
	ErrTypeChanged = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
		{Field: "settings.type", Detail: "must be the type the storage was created with"},
	})
	ErrActorNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the signed-in user does not exist",
		Errors: nil,
	}
	errSettingsMissing = errors.New("the storage has no row in the table of its provider")
)
