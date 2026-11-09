package providertest

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"databasus-backend/internal/lib/testwait"
)

type PausedReader struct {
	ctx       context.Context
	content   *bytes.Reader
	started   chan struct{}
	released  chan struct{}
	startOnce sync.Once
	closeOnce sync.Once
	readCount atomic.Int64
}

func NewPausedReader(ctx context.Context, content []byte) *PausedReader {
	return &PausedReader{
		ctx: ctx, content: bytes.NewReader(content),
		started: make(chan struct{}), released: make(chan struct{}),
	}
}

func (r *PausedReader) Read(buffer []byte) (int, error) {
	r.startOnce.Do(func() { close(r.started) })
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	case <-r.released:
	}

	// Unknown-length streams expose the SFTP library's sequential-copy path, unlike bytes.Reader.
	readBytes, err := r.content.Read(buffer[:min(len(buffer), 32*1024)])
	r.readCount.Add(1)

	return readBytes, err
}

func (r *PausedReader) WaitUntilRead(t *testing.T) {
	t.Helper()

	testwait.Until(t, "the provider to read its upload source", func(context.Context) error {
		select {
		case <-r.started:
			return nil
		default:
			return errors.New("the upload source has not been read")
		}
	})
}

func (r *PausedReader) Resume() {
	r.closeOnce.Do(func() { close(r.released) })
}

func (r *PausedReader) GetReadCount() int64 {
	return r.readCount.Load()
}
