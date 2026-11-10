package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

const (
	testProgressInterval = 5 * time.Second
	noLaterCallWindow    = 50 * time.Millisecond
)

func TestLogicalBackupProgress(t *testing.T) {
	t.Parallel()

	t.Run("reports progress at most once per interval", func(t *testing.T) {
		t.Parallel()

		var reportCount atomic.Int64

		spec := newTestBackupSpec()
		spec.OnProgress = func(float64) { reportCount.Add(1) }

		_, err := newTestPipeline(t, newDiscardingStorage()).streamDumpToStorage(t.Context(), backupStream{
			spec:        spec,
			dumpSource:  openStubPostgresqlDumpSource(t, "head -c 3145728 /dev/urandom"),
			compression: getStoredCompressionNone(),
		})

		require.NoError(t, err)
		assert.Equal(t, int64(1), reportCount.Load(), "only the report made when the dump completes")
	})

	t.Run("keeps copying while the progress listener is blocked", func(t *testing.T) {
		t.Parallel()

		listener := newBlockingProgressListener(t)
		spec := newTestBackupSpec()
		spec.OnProgress = listener.report
		spec.ProgressInterval = time.Millisecond
		gatePath := filepath.Join(t.TempDir(), "gate")
		isFileRead := make(chan struct{})
		storage := &backgroundWritingStorage{writeFile: func(_ context.Context, file io.Reader) error {
			_, err := io.Copy(io.Discard, file)
			close(isFileRead)

			return err
		}}

		streamFailure := startStream(t, storage, spec, dumpWithGate(gatePath, "exit 0"))
		listener.waitUntilEntered(t)
		openGate(t, gatePath)
		waitForClosed(t, "the storage read the end of the file", isFileRead)
		assert.False(t, listener.isReleased.Load(), "the listener is still blocked")

		listener.release()

		require.NoError(t, <-streamFailure)
	})

	t.Run("records the final size when the backup completes between ticks", func(t *testing.T) {
		t.Parallel()

		var lastReportedMb float64

		spec := newTestBackupSpec()
		spec.OnProgress = func(storedSizeMb float64) { lastReportedMb = storedSizeMb }

		backupResult, err := newTestPipeline(t, newDiscardingStorage()).streamDumpToStorage(t.Context(), backupStream{
			spec:        spec,
			dumpSource:  openStubPostgresqlDumpSource(t, "head -c 3145728 /dev/urandom"),
			compression: getStoredCompressionNone(),
		})

		require.NoError(t, err)
		assert.InDelta(t, 3.0, backupResult.StoredSizeMb, 0)
		assert.InDelta(t, backupResult.StoredSizeMb, lastReportedMb, 0)
	})

	t.Run("calls the listener once per interval with the stored size", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			var storedSizeMb atomic.Int64

			var reportedSizes []float64

			stopReporting := startReportingProgress(t.Context(), progressReportSpec{
				interval:        testProgressInterval,
				getStoredSizeMb: func() float64 { return float64(storedSizeMb.Add(1)) },
				listener:        func(sizeMb float64) { reportedSizes = append(reportedSizes, sizeMb) },
				logger:          slog.New(slog.DiscardHandler),
			})
			defer stopReporting()

			advanceBubbleTime(3*testProgressInterval + time.Millisecond)

			assert.Equal(t, []float64{1, 2, 3}, reportedSizes)
		})
	})

	t.Run("calls the listener on each interval when the size is unchanged", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			var reportedSizes []float64

			stopReporting := startReportingProgress(t.Context(), progressReportSpec{
				interval:        testProgressInterval,
				getStoredSizeMb: func() float64 { return 7 },
				listener:        func(sizeMb float64) { reportedSizes = append(reportedSizes, sizeMb) },
				logger:          slog.New(slog.DiscardHandler),
			})
			defer stopReporting()

			advanceBubbleTime(2*testProgressInterval + time.Millisecond)

			assert.Equal(t, []float64{7, 7}, reportedSizes)
		})
	})

	t.Run("skips a tick that falls due while the previous call runs", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			var callCount atomic.Int64

			releaseFirstCall := make(chan struct{})

			stopReporting := startReportingProgress(t.Context(), progressReportSpec{
				interval:        testProgressInterval,
				getStoredSizeMb: func() float64 { return 0 },
				listener: func(float64) {
					if callCount.Add(1) == 1 {
						<-releaseFirstCall
					}
				},
				logger: slog.New(slog.DiscardHandler),
			})
			defer stopReporting()

			advanceBubbleTime(4*testProgressInterval + time.Millisecond)
			assert.Equal(t, int64(1), callCount.Load())

			close(releaseFirstCall)
			synctest.Wait()

			assert.Equal(t, int64(2), callCount.Load(), "one pending tick is left of the three that fell due")
		})
	})

	t.Run("makes no call after the stop", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			var callCount atomic.Int64

			stopReporting := startReportingProgress(t.Context(), progressReportSpec{
				interval:        testProgressInterval,
				getStoredSizeMb: func() float64 { return 0 },
				listener:        func(float64) { callCount.Add(1) },
				logger:          slog.New(slog.DiscardHandler),
			})

			advanceBubbleTime(testProgressInterval + time.Millisecond)
			stopReporting()
			stopReporting()
			advanceBubbleTime(10 * testProgressInterval)

			assert.Equal(t, int64(1), callCount.Load())
		})
	})

	t.Run("waits for a call in flight and makes no later call when the dump fails", func(t *testing.T) {
		t.Parallel()

		listener := newBlockingProgressListener(t)
		spec := newTestBackupSpec()
		spec.OnProgress = listener.report
		spec.ProgressInterval = time.Millisecond
		gatePath := filepath.Join(t.TempDir(), "gate")
		isFileRead := make(chan struct{})
		storage := &backgroundWritingStorage{writeFile: func(_ context.Context, file io.Reader) error {
			_, err := io.Copy(io.Discard, file)
			close(isFileRead)

			return err
		}}

		streamFailure := startStream(t, storage, spec, dumpWithGate(gatePath, "exit 1"))
		listener.waitUntilEntered(t)
		openGate(t, gatePath)
		waitForClosed(t, "the storage read the end of the file", isFileRead)

		select {
		case streamErr := <-streamFailure:
			t.Fatalf("the stream returned while a progress call was in flight: %v", streamErr)
		case <-time.After(noLaterCallWindow):
		}

		listener.release()
		require.Error(t, <-streamFailure)
		callCountAtReturn := listener.callCount.Load()

		<-time.After(noLaterCallWindow)

		assert.Equal(t, callCountAtReturn, listener.callCount.Load())
	})

	t.Run("completes the backup when a periodic report panics", func(t *testing.T) {
		t.Parallel()

		var (
			callCount atomic.Int64
			logOutput bytes.Buffer
		)

		isFirstCallMade := make(chan struct{})
		spec := newTestBackupSpec()
		spec.Logger = slog.New(slog.NewTextHandler(&logOutput, nil))
		spec.ProgressInterval = time.Millisecond
		spec.OnProgress = func(float64) {
			if callCount.Add(1) == 1 {
				close(isFirstCallMade)
				panic("the progress write panicked")
			}
		}
		gatePath := filepath.Join(t.TempDir(), "gate")

		streamFailure := startStream(t, newDiscardingStorage(), spec, dumpWithGate(gatePath, "exit 0"))
		waitForClosed(t, "the first periodic report was made", isFirstCallMade)
		openGate(t, gatePath)

		require.NoError(t, <-streamFailure)
		assert.Contains(t, logOutput.String(), "background goroutine panicked")
		assert.Contains(t, logOutput.String(), "the progress write panicked")
	})
}

// advanceBubbleTime moves the fake time of a synctest bubble and lets every goroutine in it settle.
func advanceBubbleTime(duration time.Duration) {
	<-time.After(duration)
	synctest.Wait()
}

// blockingProgressListener holds every report until the test releases it.
type blockingProgressListener struct {
	callCount   atomic.Int64
	entered     chan struct{}
	isReleased  atomic.Bool
	releaseOnce sync.Once
	released    chan struct{}
	enterOnce   sync.Once
}

func newBlockingProgressListener(t *testing.T) *blockingProgressListener {
	t.Helper()

	listener := &blockingProgressListener{
		callCount:   atomic.Int64{},
		entered:     make(chan struct{}),
		isReleased:  atomic.Bool{},
		releaseOnce: sync.Once{},
		released:    make(chan struct{}),
		enterOnce:   sync.Once{},
	}
	t.Cleanup(listener.release)

	return listener
}

func (l *blockingProgressListener) report(float64) {
	l.callCount.Add(1)
	l.enterOnce.Do(func() { close(l.entered) })
	<-l.released
}

func (l *blockingProgressListener) release() {
	l.releaseOnce.Do(func() {
		l.isReleased.Store(true)
		close(l.released)
	})
}

func (l *blockingProgressListener) waitUntilEntered(t *testing.T) {
	t.Helper()

	waitForClosed(t, "the progress listener was entered", l.entered)
}

func startStream(t *testing.T, storage fileStorage, spec BackupSpec, dumpScript string) <-chan error {
	t.Helper()

	dumpSource := openStubPostgresqlDumpSource(t, dumpScript)
	testPipeline := newTestPipeline(t, storage)
	streamFailure := make(chan error, 1)

	go func() {
		_, err := testPipeline.streamDumpToStorage(t.Context(), backupStream{
			spec:        spec,
			dumpSource:  dumpSource,
			compression: getStoredCompressionNone(),
		})
		streamFailure <- err
	}()

	return streamFailure
}

// dumpWithGate writes more than a megabyte, waits until the gate file exists, then runs lastCommand.
func dumpWithGate(gatePath, lastCommand string) string {
	return fmt.Sprintf(
		"head -c 2097152 /dev/urandom\nwhile [ ! -e %q ]; do sleep 0.01; done\nhead -c 1048576 /dev/urandom\n%s",
		gatePath, lastCommand,
	)
}

func openGate(t *testing.T, gatePath string) {
	t.Helper()

	require.NoError(t, os.WriteFile(gatePath, nil, 0o600))
}

func waitForClosed(t *testing.T, description string, signal <-chan struct{}) {
	t.Helper()

	testwait.Until(t, description, func(context.Context) error {
		select {
		case <-signal:
			return nil
		default:
			return errors.New("not yet")
		}
	})
}
