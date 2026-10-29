package api

import (
	"context"
	"fmt"
	"io"
	"time"
)

// IdleTimeoutReader wraps an io.Reader and cancels the associated context
// if no bytes are successfully read within the specified timeout duration.
// This detects stalled uploads where the network or source stops transmitting data.
//
// When the idle timeout fires, the reader is also closed (if it implements io.Closer)
// to unblock any goroutine blocked on the underlying Read.
type IdleTimeoutReader struct {
	reader  io.Reader
	timeout time.Duration
	cancel  context.CancelCauseFunc
	timer   *time.Timer
}

// NewIdleTimeoutReader creates a reader that cancels the context via cancel
// if Read does not return any bytes for the given timeout duration.
func NewIdleTimeoutReader(reader io.Reader, timeout time.Duration, cancel context.CancelCauseFunc) *IdleTimeoutReader {
	r := &IdleTimeoutReader{
		reader:  reader,
		timeout: timeout,
		cancel:  cancel,
	}

	r.timer = time.AfterFunc(timeout, func() {
		cancel(fmt.Errorf("upload idle timeout: no bytes transmitted for %v", timeout))

		if closer, ok := reader.(io.Closer); ok {
			_ = closer.Close()
		}
	})

	return r
}

func (r *IdleTimeoutReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)

	if n > 0 {
		r.timer.Reset(r.timeout)
	}

	if err != nil && err != io.EOF {
		r.Stop()
	}

	return n, err
}

// Stop cancels the idle timer. Must be called when the reader is no longer needed.
func (r *IdleTimeoutReader) Stop() {
	r.timer.Stop()
}
