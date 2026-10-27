package background

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// CancellationRegistry holds the cancel functions of running work keyed by the work's ID, so a cancel
// request and the shutdown reach work that a goroutine of another request started.
type CancellationRegistry struct {
	mutex     sync.Mutex
	cancelFns map[uuid.UUID]func()
	// isEmpty is closed while nothing is registered and replaced by an open channel when the first work
	// registers.
	isEmpty chan struct{}
}

func NewCancellationRegistry() *CancellationRegistry {
	isEmpty := make(chan struct{})
	close(isEmpty)

	return &CancellationRegistry{
		mutex:     sync.Mutex{},
		cancelFns: make(map[uuid.UUID]func()),
		isEmpty:   isEmpty,
	}
}

// Register replaces the cancel function of a work registered under the same ID.
func (r *CancellationRegistry) Register(workID uuid.UUID, cancel func()) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.cancelFns) == 0 {
		r.isEmpty = make(chan struct{})
	}

	r.cancelFns[workID] = cancel
}

func (r *CancellationRegistry) Unregister(workID uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.remove(workID)
}

// Cancel calls and removes the cancel function of the work and reports whether one was registered.
func (r *CancellationRegistry) Cancel(workID uuid.UUID) bool {
	r.mutex.Lock()

	cancel, isRegistered := r.cancelFns[workID]
	if isRegistered {
		r.remove(workID)
	}

	r.mutex.Unlock()

	if isRegistered {
		cancel()
	}

	return isRegistered
}

// WaitUntilEmpty returns once no work is registered, or with ctx's error when ctx ends first.
func (r *CancellationRegistry) WaitUntilEmpty(ctx context.Context) error {
	r.mutex.Lock()
	isEmpty := r.isEmpty
	r.mutex.Unlock()

	select {
	case <-isEmpty:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *CancellationRegistry) remove(workID uuid.UUID) {
	if _, isRegistered := r.cancelFns[workID]; !isRegistered {
		return
	}

	delete(r.cancelFns, workID)

	if len(r.cancelFns) == 0 {
		close(r.isEmpty)
	}
}
