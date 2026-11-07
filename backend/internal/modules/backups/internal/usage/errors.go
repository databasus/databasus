package usage

import "databasus-backend/internal/lib/apierror"

var errStorageNotFound = &apierror.Problem{
	Code:   apierror.CodeNotFound,
	Detail: "the storage does not exist",
	Errors: nil,
}
