package audit

import "databasus-backend/internal/lib/apierror"

var ErrInvalidCursor = apierror.NewInvalidFieldsProblem([]apierror.FieldError{
	{Field: "after", Detail: "is not a cursor of this list"},
})
