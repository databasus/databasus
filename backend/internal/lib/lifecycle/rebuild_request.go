// Package lifecycle lets work inside the application ask the process that runs it to stop the
// application and build it again on the same port, such as after root admin accepted a recovery
// that replaces the internal database.
package lifecycle

// RebuildRequest is one per process. A request waits until the process takes it, and a second
// request before then changes nothing.
type RebuildRequest struct {
	signal chan struct{}
}

func NewRebuildRequest() *RebuildRequest {
	return &RebuildRequest{signal: make(chan struct{}, 1)}
}

func (r *RebuildRequest) Request() {
	select {
	case r.signal <- struct{}{}:
	default:
	}
}

func (r *RebuildRequest) GetSignal() <-chan struct{} {
	return r.signal
}
