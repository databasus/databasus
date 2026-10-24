package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	logTypeKey   = "log_type"
	logTypeAudit = "audit"

	flushOnExitTimeout = 5 * time.Second
)

var (
	loggerInstance      *slog.Logger
	auditLoggerInstance *slog.Logger
	sinkShutdowns       []func(context.Context) error

	// isInitialized publishes the writes above so Shutdown can read them without initializing.
	isInitialized atomic.Bool
)

var initLogger = sync.OnceFunc(func() {
	loadedSettings := mustLoadSettings()

	level := new(slog.LevelVar)
	level.Set(loadedSettings.level)

	assembledSinks := buildSinks(loadedSettings)
	sinkShutdowns = assembledSinks.shutdowns

	fanOut := newFanOutHandler(assembledSinks.handlers, level)
	loggerInstance = slog.New(fanOut)
	auditLoggerInstance = slog.New(fanOut.withLevelBypass()).With(logTypeKey, logTypeAudit)

	isInitialized.Store(true)

	for _, failure := range assembledSinks.failures {
		loggerInstance.Warn(failure.message, "error", failure.err)
	}
})

func GetLogger() *slog.Logger {
	initLogger()

	return loggerInstance
}

// GetAuditLogger's records carry log_type=audit and ignore LOG_LEVEL, so raising the level to
// error never silently drops the audit trail.
func GetAuditLogger() *slog.Logger {
	initLogger()

	return auditLoggerInstance
}

// ExitAfterFlush exists because os.Exit skips deferred work while the file and OTLP sinks buffer:
// without it the line describing a fatal startup failure - the one an operator actually needs -
// is the single class of log guaranteed never to reach the forensic copy.
func ExitAfterFlush(code int) {
	FlushAndCloseSinks(flushOnExitTimeout)
	os.Exit(code)
}

// FlushAndCloseSinks is one-way - the file writer stops accepting writes and the OTLP provider
// becomes a no-op - so it belongs only at process exit. It reports to stderr rather than through
// the logger: by the time it can fail, the sinks it would report through are the ones it closed.
func FlushAndCloseSinks(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "logger: failed to flush log sinks: %v\n", err)
	}
}

// Shutdown deliberately does not initialize: building the sinks only to close them would create
// the log file on a process that never logged, and would let a malformed OPEN_TELEMETRY_URL exit
// the process from inside a shutdown path.
func Shutdown(ctx context.Context) error {
	if !isInitialized.Load() {
		return nil
	}

	var shutdownErrors []error
	for _, shutdownSink := range sinkShutdowns {
		if err := shutdownSink(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	return errors.Join(shutdownErrors...)
}

type sinkFailure struct {
	message string
	err     error
}

type builtSinks struct {
	handlers  []slog.Handler
	shutdowns []func(context.Context) error
	failures  []sinkFailure
}

// buildSinks reports failures instead of logging them: it runs before the logger exists, and an
// optional sink that fails to open must not take the process down the way malformed config does.
func buildSinks(loadedSettings settings) builtSinks {
	assembledSinks := builtSinks{handlers: []slog.Handler{newStdoutHandler()}}

	if loadedSettings.file.isEnabled {
		fileWriter, err := newRotatingFileWriter(rotatingFileSpec{
			path:       loadedSettings.file.path,
			maxSizeMB:  logFileMaxSizeMB,
			maxBackups: logFileMaxBackups,
		})
		if err == nil {
			assembledSinks.handlers = append(assembledSinks.handlers,
				slog.NewJSONHandler(fileWriter, &slog.HandlerOptions{Level: slog.LevelDebug}))
			assembledSinks.shutdowns = append(assembledSinks.shutdowns, fileWriter.Shutdown)
		} else {
			assembledSinks.failures = append(assembledSinks.failures, sinkFailure{"log file sink disabled", err})
		}
	}

	if loadedSettings.openTelemetry != nil {
		otelHandler, shutdownProvider, err := newOpenTelemetryHandler(
			context.Background(),
			*loadedSettings.openTelemetry,
			loadedSettings.serviceVersion,
		)
		if err == nil {
			assembledSinks.handlers = append(assembledSinks.handlers, otelHandler)
			assembledSinks.shutdowns = append(assembledSinks.shutdowns, shutdownProvider)
		} else {
			assembledSinks.failures = append(assembledSinks.failures, sinkFailure{"otlp log sink disabled", err})
		}
	}

	return assembledSinks
}
