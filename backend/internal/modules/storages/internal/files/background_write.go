package files

import (
	"context"
	"io"
)

// BackgroundWrite is an upload running beside the producer that feeds it through a pipe. Logical
// dumps and physical streams drive it the same way: peek at the error while the producer is still
// running, then take the receipt once the writer is done.
type BackgroundWrite struct {
	// Errors stays writable because the engines put a peeked value back and their cancellation
	// cleanup drains it.
	Errors   chan error
	Receipts chan WriteReceipt
}

type BackgroundWriteSpec struct {
	Reference StoredFileReference
	Source    *io.PipeReader
	// OnFailure is called with the error of a failed upload, so the producer stops feeding the pipe.
	OnFailure context.CancelCauseFunc
}

// StartBackgroundWrite sends the receipt before the error, so a caller holding the error knows the
// receipt is already waiting.
func (s *Store) StartBackgroundWrite(ctx context.Context, spec BackgroundWriteSpec) BackgroundWrite {
	write := BackgroundWrite{
		Errors:   make(chan error, 1),
		Receipts: make(chan WriteReceipt, 1),
	}

	s.uploads.Go(ctx, "storage background write", func(ctx context.Context) error {
		receipt, err := WriteReceipt{}, ErrBackgroundWriteStopped

		// The deferred delivery also runs when the upload panics, so the caller never waits for a
		// result that cannot come.
		defer func() {
			if err != nil {
				_ = spec.Source.CloseWithError(err)
				spec.OnFailure(err)
			}

			write.Receipts <- receipt
			write.Errors <- err
		}()

		receipt, err = s.WriteFile(ctx, spec.Reference, spec.Source)

		return nil
	})

	return write
}
