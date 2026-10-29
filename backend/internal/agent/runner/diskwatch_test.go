package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type fakeDiskProbe struct {
	mu     sync.Mutex
	values []int64
	errs   []error
	calls  int
}

func (p *fakeDiskProbe) GetDiskUsageBytes(context.Context) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	i := p.calls
	if i >= len(p.values) {
		i = len(p.values) - 1
	}

	p.calls++

	var err error
	if p.calls-1 < len(p.errs) {
		err = p.errs[p.calls-1]
	}

	return p.values[i], err
}

func newTestDiskWatcher(probe diskUsageProber, ceilingBytes int64, onLimitExceeded func()) *diskWatcher {
	w := newDiskWatcher(diskWatchSpec{
		probe: probe, ceilingBytes: ceilingBytes, interval: time.Millisecond,
		onLimitExceeded: onLimitExceeded, log: slog.New(slog.DiscardHandler),
	})

	return w
}

func TestDiskWatcher(t *testing.T) {
	t.Run("fires once when disk usage reaches the ceiling", func(t *testing.T) {
		var fired atomic.Int32
		done := make(chan struct{})

		w := newTestDiskWatcher(
			&fakeDiskProbe{values: []int64{100}}, 100,
			func() { fired.Add(1); close(done) },
		)

		stopped := make(chan struct{})
		go func() { w.run(t.Context()); close(stopped) }()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("watcher never fired despite usage at ceiling")
		}

		<-stopped
		assert.Equal(t, int32(1), fired.Load(), "onLimitExceeded must fire exactly once")
	})
	t.Run("does not fire when disk usage stays below the ceiling", func(t *testing.T) {
		var fired atomic.Int32
		ctx, cancel := context.WithCancel(t.Context())

		w := newTestDiskWatcher(
			&fakeDiskProbe{values: []int64{50}}, 100,
			func() { fired.Add(1) },
		)

		stopped := make(chan struct{})
		go func() { w.run(ctx); close(stopped) }()

		testwait.Until(t, "disk usage probe", func(context.Context) error {
			probe, isFakeProbe := w.probe.(*fakeDiskProbe)
			if !isFakeProbe {
				return fmt.Errorf("unexpected disk probe")
			}
			probe.mu.Lock()
			defer probe.mu.Unlock()
			if probe.calls == 0 {
				return fmt.Errorf("probe has not run")
			}
			return nil
		})
		cancel()

		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Fatal("watcher did not stop on context cancel")
		}

		assert.Equal(t, int32(0), fired.Load(), "must not fire while under the ceiling")
	})
	t.Run("does not trip on error when the disk probe recovers after errors", func(t *testing.T) {
		var fired atomic.Int32
		done := make(chan struct{})

		probe := &fakeDiskProbe{
			values: []int64{0, 0, 200},
			errs:   []error{errors.New("daemon busy"), errors.New("daemon busy"), nil},
		}

		w := newTestDiskWatcher(probe, 100, func() { fired.Add(1); close(done) })

		stopped := make(chan struct{})
		go func() { w.run(t.Context()); close(stopped) }()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("watcher never fired after probe recovered above ceiling")
		}

		assert.Equal(t, int32(1), fired.Load(),
			"a probe error must not trip the watcher nor stop the loop")
	})
	t.Run("stops without firing when the context is cancelled", func(t *testing.T) {
		var fired atomic.Int32
		ctx, cancel := context.WithCancel(t.Context())

		w := newTestDiskWatcher(
			&fakeDiskProbe{values: []int64{10}}, 100,
			func() { fired.Add(1) },
		)

		stopped := make(chan struct{})
		go func() { w.run(ctx); close(stopped) }()

		cancel()

		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Fatal("watcher did not return after context cancel")
		}

		require.Equal(t, int32(0), fired.Load())
	})
}
