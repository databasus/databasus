package httpapi

import "databasus-backend/internal/modules/users"

func toIdentityConfirmationBody(confirmation users.IdentityConfirmation) IdentityConfirmation {
	return IdentityConfirmation{
		Method:    IdentityConfirmationMethod(confirmation.Method),
		ExpiresAt: confirmation.ExpiresAt,
	}
}
