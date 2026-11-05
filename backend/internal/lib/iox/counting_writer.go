package iox

import (
	"io"
	"sync/atomic"
)

// CountingWriter counts atomically because compressing writers hand their output
// to internal goroutines, so the count is read from a different goroutine than
// the one that produced the bytes.
type CountingWriter struct {
	writer       io.Writer
	bytesWritten atomic.Int64
}

func NewCountingWriter(writer io.Writer) *CountingWriter {
	return &CountingWriter{writer: writer}
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.writer.Write(p)
	if n > 0 {
		cw.bytesWritten.Add(int64(n))
	}

	return n, err
}

func (cw *CountingWriter) GetBytesWritten() int64 {
	return cw.bytesWritten.Load()
}
