package keyexport

import "databasus-backend/internal/lib/apierror"

var (
	ErrInsecureTransferNotAcknowledged = apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
		Field:  "isInsecureTransferAcknowledged",
		Detail: "over unencrypted HTTP the risk of transferring the key must be acknowledged",
	}})
	ErrTokenNotUsable = &apierror.Problem{
		Code:   apierror.CodeUnauthenticated,
		Detail: "the download token is unknown, used or expired",
		Errors: nil,
	}
	ErrAcknowledgmentNotSaved = apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
		Field:  "isSaved",
		Detail: "only a saved key can be acknowledged",
	}})
)
