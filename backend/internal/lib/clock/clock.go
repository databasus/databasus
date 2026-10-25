// Package clock gives the time that drives behavior, so a test can move it instead of waiting.
package clock

import (
	"sync"
	"time"
)

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

// Manual stands still until the test moves it.
type Manual struct {
	mutex sync.Mutex
	now   time.Time
}

func NewManual(start time.Time) *Manual {
	return &Manual{mutex: sync.Mutex{}, now: start.UTC()}
}

func (m *Manual) Now() time.Time {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.now
}

func (m *Manual) Advance(duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.now = m.now.Add(duration)
}

func (m *Manual) Set(now time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.now = now.UTC()
}
