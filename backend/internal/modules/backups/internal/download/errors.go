package download

import (
	"databasus-backend/internal/lib/apierror"
)

var (
	// ErrUserStreamHeld refuses a download, or a token for one, while another stream of the user runs.
	ErrUserStreamHeld = &apierror.Problem{
		Code:   apierror.CodeAlreadyRunning,
		Detail: "another download or restore stream of the user is running",
		Errors: nil,
	}
	// ErrTokenNotUsable answers an expired, used, unknown or mismatched token alike, so the answer
	// tells nothing about which tokens exist.
	ErrTokenNotUsable = &apierror.Problem{
		Code:   apierror.CodeUnauthenticated,
		Detail: "the download token is not valid for this backup",
		Errors: nil,
	}
)
