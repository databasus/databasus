package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
)

type shutdownRecorder struct {
	beginShutdownCalls atomic.Int32
	flushLogsCalls     atomic.Int32
	shutdownBegan      chan struct{}
}

func newShutdownRecorder() *shutdownRecorder {
	return &shutdownRecorder{shutdownBegan: make(chan struct{}, 2)}
}

func (r *shutdownRecorder) beginShutdown() {
	r.beginShutdownCalls.Add(1)
	r.shutdownBegan <- struct{}{}
}

func (r *shutdownRecorder) flushLogs(time.Duration) {
	r.flushLogsCalls.Add(1)
}

func Test_StartGracefulShutdownOnStopSignal_WhenNothingRuns_FinishesAtOnce(t *testing.T) {
	recorder := newShutdownRecorder()
	shutdownFinished := startGracefulShutdownOnStopSignal(
		newTestShutdownSpec(t, task_cancellation.NewRegistry(), recorder, shutdownDrainTimeout),
	)

	signalStartedAt := time.Now().UTC()
	sendStopSignalToTestProcess(t)

	waitForShutdownToFinish(t, shutdownFinished, 3*time.Second)
	assert.Less(t, time.Since(signalStartedAt), 3*time.Second)
	assert.Equal(t, int32(1), recorder.flushLogsCalls.Load())
}

func Test_StartGracefulShutdownOnStopSignal_WhenTaskStaysRegistered_FinishesAtDrainDeadline(t *testing.T) {
	const drainTimeout = 300 * time.Millisecond

	registry := task_cancellation.NewRegistry()
	registry.RegisterTask(uuid.New(), func() {})

	recorder := newShutdownRecorder()
	shutdownFinished := startGracefulShutdownOnStopSignal(
		newTestShutdownSpec(t, registry, recorder, drainTimeout),
	)

	signalStartedAt := time.Now().UTC()
	sendStopSignalToTestProcess(t)

	waitForShutdownToFinish(t, shutdownFinished, 5*time.Second)
	assert.GreaterOrEqual(t, time.Since(signalStartedAt), drainTimeout)
	assert.Equal(t, int32(1), recorder.flushLogsCalls.Load())
}

func Test_StartGracefulShutdownOnStopSignal_WhenSecondSignalArrives_ShutsDownOnce(t *testing.T) {
	const drainTimeout = time.Second

	registry := task_cancellation.NewRegistry()
	registry.RegisterTask(uuid.New(), func() {})

	recorder := newShutdownRecorder()
	shutdownFinished := startGracefulShutdownOnStopSignal(
		newTestShutdownSpec(t, registry, recorder, drainTimeout),
	)

	signalStartedAt := time.Now().UTC()
	sendStopSignalToTestProcess(t)

	select {
	case <-recorder.shutdownBegan:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not begin after the first signal")
	}

	sendStopSignalToTestProcess(t)

	waitForShutdownToFinish(t, shutdownFinished, 5*time.Second)
	assert.Less(t, time.Since(signalStartedAt), drainTimeout+shutdownLogFlushTimeout)
	assert.Equal(t, int32(1), recorder.beginShutdownCalls.Load())
	assert.Equal(t, int32(1), recorder.flushLogsCalls.Load())
}

func newTestShutdownSpec(
	t *testing.T,
	registry *task_cancellation.Registry,
	recorder *shutdownRecorder,
	drainTimeout time.Duration,
) gracefulShutdownSpec {
	t.Helper()

	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(testServer.Close)

	return gracefulShutdownSpec{
		server:                   testServer.Config,
		taskCancellationRegistry: registry,
		beginShutdown:            recorder.beginShutdown,
		flushLogs:                recorder.flushLogs,
		drainTimeout:             drainTimeout,
		logFlushTimeout:          shutdownLogFlushTimeout,
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func sendStopSignalToTestProcess(t *testing.T) {
	t.Helper()

	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
}

func waitForShutdownToFinish(t *testing.T, shutdownFinished <-chan struct{}, timeout time.Duration) {
	t.Helper()

	select {
	case <-shutdownFinished:
	case <-time.After(timeout):
		t.Fatal("shutdown did not finish in time")
	}
}
