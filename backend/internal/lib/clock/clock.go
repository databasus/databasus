// Package clock gives the time that drives behavior, so a test can move it instead of waiting.
package clock

import "time"

// Source returns the current time in UTC.
type Source interface {
	Now() time.Time
}

type system struct{}

func NewSystem() Source {
	return system{}
}

func (system) Now() time.Time {
	return time.Now().UTC()
}
