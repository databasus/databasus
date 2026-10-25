package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"databasus-backend/internal/lib/logging"
)

const flushOnExitTimeout = 5 * time.Second

var currentLogging atomic.Pointer[logging.Logging]

var initWithoutApp = sync.OnceFunc(func() {
	currentLogging.CompareAndSwap(nil, loadWithoutApp())
})

func Install(installedLogging *logging.Logging) {
	currentLogging.Store(installedLogging)
}

func GetLogger() *slog.Logger {
	return getLogging().GetLogger()
}

func GetAuditLogger() *slog.Logger {
	return getLogging().GetAuditLogger()
}

// ExitAfterFlush exists because os.Exit skips deferred work while the file and OTLP sinks buffer:
// without it the line describing a fatal startup failure - the one an operator actually needs -
// is the single class of log guaranteed never to reach the forensic copy.
func ExitAfterFlush(code int) {
	FlushAndCloseSinks(flushOnExitTimeout)
	os.Exit(code)
}

// FlushAndCloseSinks reports to stderr rather than through the logger: by the time it can fail,
// the sinks it would report through are the ones it closed.
func FlushAndCloseSinks(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to flush log sinks: %v\n", err)
	}
}

// Shutdown deliberately does not build the logging: building the sinks only to close them would
// create the log file on a process that never logged.
func Shutdown(ctx context.Context) error {
	loadedLogging := currentLogging.Load()
	if loadedLogging == nil {
		return nil
	}

	return loadedLogging.Shutdown(ctx)
}

func Redact(message string) string {
	return logging.Redact(message)
}

func getLogging() *logging.Logging {
	if loadedLogging := currentLogging.Load(); loadedLogging != nil {
		return loadedLogging
	}

	initWithoutApp()

	return currentLogging.Load()
}
