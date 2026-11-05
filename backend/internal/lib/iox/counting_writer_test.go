package iox

import (
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CountingWriter_WhenWrittenFromManyGoroutines_CountsEveryByte(t *testing.T) {
	const writerCount = 16
	const writesPerWriter = 64
	const payloadSize = 128

	countingWriter := NewCountingWriter(io.Discard)
	payload := make([]byte, payloadSize)

	var writers sync.WaitGroup

	for range writerCount {
		writers.Go(func() {
			for range writesPerWriter {
				_, err := countingWriter.Write(payload)
				assert.NoError(t, err)
			}
		})
	}

	writers.Wait()

	require.Equal(
		t,
		int64(writerCount*writesPerWriter*payloadSize),
		countingWriter.GetBytesWritten(),
	)
}
