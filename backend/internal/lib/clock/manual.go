package clock

import (
	"sync"
	"time"
)

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
