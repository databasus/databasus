// Package shutdown turns a stop signal into a cancelled context and drains the shutdown steps
// within a fixed limit.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ListenForStopSignal registers the process's only SIGTERM and SIGINT handler and returns a
// context that ends on the first signal. The registration stays until stopListening is called, so
// a second signal during shutdown lands in the already ended context instead of killing the process
// (spec application-shutdown). Call stopListening once the shutdown has finished.
func ListenForStopSignal(ctx context.Context) (stopSignalCtx context.Context, stopListening context.CancelFunc) {
	return signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
}
