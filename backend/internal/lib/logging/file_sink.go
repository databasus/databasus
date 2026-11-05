package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	fileQueueCapacity  = 4096
	fileEnqueueTimeout = 1 * time.Second
	dropReportInterval = 30 * time.Second
)

// rotatingFileWriter decouples log calls from disk with its own queue. Unlike the remote sink it
// applies backpressure before dropping: the file is the forensic copy, and local disk is fast
// enough that waiting beats losing lines. The timeout still caps the damage a hung disk can do.
type rotatingFileWriter struct {
	queue            chan []byte
	rotator          *lumberjack.Logger
	stop             chan struct{}
	drained          chan struct{}
	shutdownOnce     func()
	droppedCount     atomic.Int64
	lastDropReportAt atomic.Int64
}

type rotatingFileSpec struct {
	path       string
	maxSizeMB  int
	maxBackups int
}

func newRotatingFileWriter(spec rotatingFileSpec) (*rotatingFileWriter, error) {
	if err := os.MkdirAll(filepath.Dir(spec.path), 0o750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	writer := &rotatingFileWriter{
		queue: make(chan []byte, fileQueueCapacity),
		rotator: &lumberjack.Logger{
			Filename:   spec.path,
			MaxSize:    spec.maxSizeMB,
			MaxBackups: spec.maxBackups,
		},
		stop:    make(chan struct{}),
		drained: make(chan struct{}),
	}

	writer.shutdownOnce = sync.OnceFunc(func() { close(writer.stop) })

	go writer.drain()

	return writer, nil
}

func (w *rotatingFileWriter) Write(line []byte) (int, error) {
	queued := slices.Clone(line)

	select {
	case w.queue <- queued:
		return len(line), nil
	case <-w.stop:
		return len(line), nil
	default:
	}

	timeout := time.NewTimer(fileEnqueueTimeout)
	defer timeout.Stop()

	select {
	case w.queue <- queued:
	case <-w.stop:
	case <-timeout.C:
		w.recordDrop()
	}

	return len(line), nil
}

func (w *rotatingFileWriter) Shutdown(ctx context.Context) error {
	w.shutdownOnce()

	select {
	case <-w.drained:
	case <-ctx.Done():
		return fmt.Errorf("log file writer did not drain in time: %w", ctx.Err())
	}

	return w.rotator.Close()
}

func (w *rotatingFileWriter) drain() {
	defer close(w.drained)

	for {
		select {
		case line := <-w.queue:
			w.writeLine(line)

		case <-w.stop:
			for {
				select {
				case line := <-w.queue:
					w.writeLine(line)
				default:
					return
				}
			}
		}
	}
}

func (w *rotatingFileWriter) writeLine(line []byte) {
	if _, err := w.rotator.Write(line); err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to write to log file: %v\n", err)
	}
}

// recordDrop reports to stderr rather than through the logger: the sink is saturated at this
// point, so routing the report back through it would re-enter the very path that is failing.
func (w *rotatingFileWriter) recordDrop() {
	dropped := w.droppedCount.Add(1)

	now := time.Now().UTC().UnixNano()
	lastReportedAt := w.lastDropReportAt.Load()

	if now-lastReportedAt < int64(dropReportInterval) {
		return
	}

	if !w.lastDropReportAt.CompareAndSwap(lastReportedAt, now) {
		return
	}

	fmt.Fprintf(os.Stderr, "logger: log file queue is full, dropped %d entries so far\n", dropped)
}
