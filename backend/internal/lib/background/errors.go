package background

import (
	"errors"
	"fmt"
)

var ErrUnknownJob = errors.New("unknown background job")

type PanicError struct {
	Value any
	Stack string
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Value)
}
