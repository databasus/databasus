package background

import (
	"sync"

	"github.com/google/uuid"
)

// CancellationRegistry holds the cancel functions of running work keyed by the work's ID, so a cancel
// request and the shutdown reach work that a goroutine of another request started.
type CancellationRegistry struct {
	mutex     sync.Mutex
	cancelFns map[uuid.UUID]func()
}

func NewCancellationRegistry() *CancellationRegistry {
	return &CancellationRegistry{
		mutex:     sync.Mutex{},
		cancelFns: make(map[uuid.UUID]func()),
	}
}

// Register replaces the cancel function of a work registered under the same ID.
func (r *CancellationRegistry) Register(workID uuid.UUID, cancel func()) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.cancelFns[workID] = cancel
}

func (r *CancellationRegistry) Unregister(workID uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.cancelFns, workID)
}

// Cancel calls and removes the cancel function of the work and reports whether one was registered.
func (r *CancellationRegistry) Cancel(workID uuid.UUID) bool {
	r.mutex.Lock()

	cancel, isRegistered := r.cancelFns[workID]
	delete(r.cancelFns, workID)

	r.mutex.Unlock()

	if isRegistered {
		cancel()
	}

	return isRegistered
}
