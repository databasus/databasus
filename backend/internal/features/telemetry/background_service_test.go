package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
)

type sequenceSender struct {
	results []error
	calls   atomic.Int32
	done    chan struct{}
	doneAt  int32
}

func (s *sequenceSender) Send(_ context.Context, _ *CollectRequest) error {
	idx := s.calls.Add(1) - 1

	var err error
	if int(idx) < len(s.results) {
		err = s.results[int(idx)]
	}

	if s.done != nil && idx+1 == s.doneAt {
		close(s.done)
	}

	return err
}

func newBackgroundServiceUnderTest(t *testing.T, sender TelemetrySender) *TelemetryBackgroundService {
	t.Helper()

	loader := NewInstanceFileLoader(
		filepath.Join(t.TempDir(), "instance.json"),
		slog.New(slog.DiscardHandler),
	)

	service := NewTelemetryService(
		loader,
		sender,
		&fakeDatabaseLister{},
		&fakeStorageTypeLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeLatestPhysicalBackupReader{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{},
		"9.9.9",
		slog.New(slog.DiscardHandler),
	)

	bg := NewTelemetryBackgroundService(service, slog.New(slog.DiscardHandler))
	// Compress all timings so tests run in milliseconds.
	bg.WarmupDuration = 5 * time.Millisecond
	bg.SuccessInterval = 20 * time.Millisecond
	bg.Jitter = 0
	bg.InitialBackoff = 5 * time.Millisecond
	bg.MaxBackoff = 40 * time.Millisecond
	return bg
}

func Test_Run_WhenDisabled_ReturnsImmediately(t *testing.T) {
	original := config.GetEnv().IsDisableAnonymousTelemetry
	config.GetEnv().IsDisableAnonymousTelemetry = true
	t.Cleanup(func() {
		config.GetEnv().IsDisableAnonymousTelemetry = original
	})

	sender := &sequenceSender{}
	bg := newBackgroundServiceUnderTest(t, sender)

	finished := make(chan struct{})
	go func() {
		bg.Run(context.Background())
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("Run did not return when telemetry is disabled")
	}

	assert.Equal(t, int32(0), sender.calls.Load())
}

func Test_Run_WhenContextCancelledDuringWarmup_ReturnsCleanly(t *testing.T) {
	sender := &sequenceSender{}
	bg := newBackgroundServiceUnderTest(t, sender)
	bg.WarmupDuration = 5 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		bg.Run(ctx)
		close(finished)
	}()

	cancel()

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancellation")
	}

	assert.Equal(t, int32(0), sender.calls.Load())
}

func Test_Run_OnFailure_RetriesWithExponentialBackoff(t *testing.T) {
	done := make(chan struct{})
	sender := &sequenceSender{
		results: []error{
			errors.New("err1"),
			errors.New("err2"),
			errors.New("err3"),
		},
		done:   done,
		doneAt: 3,
	}
	bg := newBackgroundServiceUnderTest(t, sender)

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		bg.Run(ctx)
		close(finished)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("did not see three retries within timeout")
	}

	cancel()
	<-finished

	assert.GreaterOrEqual(t, sender.calls.Load(), int32(3))
}

func Test_Run_OnSuccessAfterFailure_ResetsBackoffAndContinues(t *testing.T) {
	done := make(chan struct{})
	sender := &sequenceSender{
		results: []error{errors.New("transient"), nil, nil},
		done:    done,
		doneAt:  3,
	}
	bg := newBackgroundServiceUnderTest(t, sender)

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		bg.Run(ctx)
		close(finished)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("did not see recovery within timeout")
	}

	cancel()
	<-finished

	assert.GreaterOrEqual(t, sender.calls.Load(), int32(3))
}

func Test_Run_WhenCalledTwice_Panics(t *testing.T) {
	sender := &sequenceSender{}
	bg := newBackgroundServiceUnderTest(t, sender)

	// A cancelled context makes the first Run return at once, after it has marked itself as run.
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	bg.Run(cancelledCtx)

	require.Panics(t, func() {
		bg.Run(cancelledCtx)
	})
}
