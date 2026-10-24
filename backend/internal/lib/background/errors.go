package background

import "fmt"

type PanicError struct {
	Value any
	Stack string
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Value)
}
