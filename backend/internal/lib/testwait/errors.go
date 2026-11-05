package testwait

import "errors"

var (
	ErrTimedOut  = errors.New("timed out waiting")
	ErrTestEnded = errors.New("stopped waiting because the test ended")
)
