package agents

import "databasus-backend/internal/lib/apierror"

var (
	ErrAgentNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "verification agent not found",
		Errors: nil,
	}
	ErrInvalidAgentToken = &apierror.Problem{
		Code:   apierror.CodeUnauthenticated,
		Detail: "invalid agent credentials",
		Errors: nil,
	}
)
