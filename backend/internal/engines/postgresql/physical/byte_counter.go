package physical

import (
	"io"
	"sync/atomic"
)

type byteCounter struct {
	writer io.Writer
	bytes  atomic.Int64
}

func (b *byteCounter) Write(p []byte) (int, error) {
	n, err := b.writer.Write(p)
	if n > 0 {
		b.bytes.Add(int64(n))
	}

	return n, err
}
func (b *byteCounter) GetBytesWritten() int64 { return b.bytes.Load() }
