package auditlog

import "databasus-backend/internal/lib/apierror"

var (
	ErrGlobalViewForInstanceAdmins = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "only an instance admin reads the audit log of the whole instance",
		Errors: nil,
	}
	ErrAnotherUsersView = &apierror.Problem{
		Code:   apierror.CodePermissionDenied,
		Detail: "only that user and an instance admin read the audit log of a user",
		Errors: nil,
	}
)
