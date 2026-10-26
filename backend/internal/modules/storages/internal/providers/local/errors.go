package local

import "errors"

var (
	ErrProbeSaveFailed   = errors.New("save a file through local storage")
	ErrProbeDeleteFailed = errors.New("delete a file through local storage")
)
