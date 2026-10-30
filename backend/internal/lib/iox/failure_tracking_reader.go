package iox

import (
	"errors"
	"io"
	"sync"
)

// FailureTrackingReader exists because os/exec discards the stdin copy error whenever the
// child process exits non-zero, so a download that dies mid-stream surfaces as the child's
// own downstream parse error instead of the transport failure that actually caused it.
// Tracking only the read side keeps the EPIPE from an early-dying child out of the result.
type FailureTrackingReader struct {
	reader         io.Reader
	mu             sync.Mutex
	firstReadError error
}

func NewFailureTrackingReader(reader io.Reader) *FailureTrackingReader {
	return &FailureTrackingReader{reader: reader}
}

func (ftr *FailureTrackingReader) Read(p []byte) (int, error) {
	n, err := ftr.reader.Read(p)

	if err != nil && !errors.Is(err, io.EOF) {
		ftr.mu.Lock()
		if ftr.firstReadError == nil {
			ftr.firstReadError = err
		}
		ftr.mu.Unlock()
	}

	return n, err
}

func (ftr *FailureTrackingReader) GetFirstReadError() error {
	ftr.mu.Lock()
	defer ftr.mu.Unlock()

	return ftr.firstReadError
}
