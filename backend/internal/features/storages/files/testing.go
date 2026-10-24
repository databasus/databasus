//go:build testhelpers

package storage_files

import (
	"context"
	"fmt"
	"time"

	db "databasus-backend/internal/storage"
)

// drainDeadlineForTest keeps a provider that never succeeds from hanging a test.
const drainDeadlineForTest = 30 * time.Second

// TimingsForTest keeps every bound long enough to be observable in a single test
// and short enough that no test waits on a production interval.
func TimingsForTest() Timings {
	return Timings{
		CommitWindow:   2 * time.Second,
		AttemptLease:   time.Second,
		RetryBaseDelay: 10 * time.Millisecond,
		RetryMaxDelay:  50 * time.Millisecond,
		WorkerTick:     10 * time.Millisecond,
		ClaimBatchSize: 10,
	}
}

// It watches only what the caller names, because the slot database is shared with
// every other test running in parallel.
func (w *DeletionWorker) DrainForTest(ctx context.Context, references ...StoredFileReference) error {
	deadline := time.Now().UTC().Add(drainDeadlineForTest)

	for {
		outstanding, err := w.repository.CountByReferences(db.GetDb(), references)
		if err != nil {
			return err
		}

		if outstanding == 0 {
			return nil
		}

		if time.Now().UTC().After(deadline) {
			return fmt.Errorf("%d file deletions still pending after %s", outstanding, drainDeadlineForTest)
		}

		w.RunOnce(ctx)

		time.Sleep(w.timings.RetryMaxDelay)
	}
}
