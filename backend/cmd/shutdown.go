package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
)

// The drain and the log flush together stay under 8 seconds, which leaves the embedded PostgreSQL
// server time for a clean shutdown within Docker's default stop timeout of 10 seconds.
const (
	shutdownDrainTimeout    = 6 * time.Second
	shutdownLogFlushTimeout = 2 * time.Second
)

type gracefulShutdownSpec struct {
	server                   *http.Server
	taskCancellationRegistry *task_cancellation.Registry
	// Closes the application's shutdown signal and cancels the background services.
	beginShutdown   func()
	flushLogs       func(timeout time.Duration)
	drainTimeout    time.Duration
	logFlushTimeout time.Duration
	logger          *slog.Logger
}

// startGracefulShutdownOnStopSignal registers the process's only SIGTERM and SIGINT handler before
// it returns. The registration stays until the shutdown has finished, so a second signal lands in
// the already fired context instead of killing the process. The returned channel closes once the
// shutdown has finished.
func startGracefulShutdownOnStopSignal(spec gracefulShutdownSpec) <-chan struct{} {
	stopSignalCtx, stopListening := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	shutdownFinished := make(chan struct{})

	go func() {
		defer close(shutdownFinished)
		defer stopListening()

		<-stopSignalCtx.Done()

		shutDown(spec)
	}()

	return shutdownFinished
}

func shutDown(spec gracefulShutdownSpec) {
	spec.logger.Info("shutdown signal received")

	spec.beginShutdown()

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), spec.drainTimeout)
	defer cancelDrain()

	var shutdownSteps sync.WaitGroup

	shutdownSteps.Go(func() {
		if err := spec.server.Shutdown(drainCtx); err != nil {
			spec.logger.Error("http server forced to shut down", "error", err)
			return
		}

		spec.logger.Info("http server gracefully stopped")
	})

	shutdownSteps.Go(func() {
		if err := spec.taskCancellationRegistry.WaitUntilNoTasksRegistered(drainCtx); err != nil {
			spec.logger.Warn("running tasks did not finish before shutdown", "error", err)
			return
		}

		spec.logger.Info("running tasks finished")
	})

	shutdownSteps.Wait()

	// Flushed after the drain so the logs of in-flight requests and tasks are included, on its own
	// deadline so a slow drain cannot hand the flush an already expired context.
	spec.flushLogs(spec.logFlushTimeout)
}
