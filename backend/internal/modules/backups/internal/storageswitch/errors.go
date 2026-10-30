package storageswitch

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrNotFound     = &apierror.Problem{Code: apierror.CodeNotFound, Detail: "the storage switch does not exist"}
	errMissingBatch = errors.New("storage switch deletion batch is missing")
)
