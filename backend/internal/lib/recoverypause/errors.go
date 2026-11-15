package recoverypause

import "databasus-backend/internal/lib/apierror"

var ErrRecoveryPaused = &apierror.Problem{
	Code: apierror.CodeRecoveryPaused,
	Detail: "the instance is paused after a recovery until root admin confirms the old instance is stopped " +
		"and activates it",
	Errors: nil,
}
