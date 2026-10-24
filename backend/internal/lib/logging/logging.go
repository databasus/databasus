// Package logging builds the application's slog pipeline: a stdout console sink, an optional
// rotating file, and an optional OpenTelemetry exporter, fanned out from a single handler.
//
// Redaction lives here rather than at call sites because every sink must be covered by it, and a
// call site cannot know that its line is about to be written to disk and shipped off-box.
package logging

import (
	"context"
	"errors"
	"log/slog"

	"databasus-backend/internal/lib/config"
)

const (
	logTypeKey   = "log_type"
	logTypeAudit = "audit"
)

type Logging struct {
	logger        *slog.Logger
	auditLogger   *slog.Logger
	sinkShutdowns []func(context.Context) error
}

// New fails on a malformed log export setting: such a typo's only alternative is shipping no logs
// and no error. A sink that fails to open only disables itself and is reported as a warning.
func New(ctx context.Context, loadedConfig config.Config) (*Logging, error) {
	loadedSettings, err := newSettings(loadedConfig)
	if err != nil {
		return nil, err
	}

	assembledSinks := buildSinks(ctx, loadedSettings)
	logging := assemble(assembledSinks, loadedSettings.level)

	for _, failure := range assembledSinks.failures {
		logging.logger.WarnContext(
			ctx,
			"log sink disabled",
			slog.String("sink", failure.sink),
			slog.Any("error", failure.err),
		)
	}

	return logging, nil
}

// NewStdout logs to stdout alone. It reports a configuration that could not be loaded, when the
// settings of the other sinks are unknown.
func NewStdout() *Logging {
	return assemble(builtSinks{handlers: []slog.Handler{newStdoutHandler()}}, slog.LevelInfo)
}

func (l *Logging) GetLogger() *slog.Logger {
	return l.logger
}

// GetAuditLogger's records carry log_type=audit and ignore LOG_LEVEL, so raising the level to
// error never silently drops the audit trail.
func (l *Logging) GetAuditLogger() *slog.Logger {
	return l.auditLogger
}

// Shutdown is one-way - the file writer stops accepting writes and the OTLP provider becomes a
// no-op - so it belongs only at process exit.
func (l *Logging) Shutdown(ctx context.Context) error {
	var shutdownErrors []error

	for _, shutdownSink := range l.sinkShutdowns {
		if err := shutdownSink(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	return errors.Join(shutdownErrors...)
}

func assemble(assembledSinks builtSinks, minimumLevel slog.Level) *Logging {
	level := new(slog.LevelVar)
	level.Set(minimumLevel)

	fanOut := newFanOutHandler(assembledSinks.handlers, level)

	return &Logging{
		logger:        slog.New(fanOut),
		auditLogger:   slog.New(fanOut.withLevelBypass()).With(slog.String(logTypeKey, logTypeAudit)),
		sinkShutdowns: assembledSinks.shutdowns,
	}
}

type sinkFailure struct {
	sink string
	err  error
}

type builtSinks struct {
	handlers  []slog.Handler
	shutdowns []func(context.Context) error
	failures  []sinkFailure
}

// buildSinks reports failures instead of logging them: it runs before the logger exists, and an
// optional sink that fails to open must not take the process down the way malformed config does.
func buildSinks(ctx context.Context, loadedSettings settings) builtSinks {
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
			assembledSinks.failures = append(assembledSinks.failures, sinkFailure{"file", err})
		}
	}

	if loadedSettings.openTelemetry != nil {
		otelHandler, shutdownProvider, err := newOpenTelemetryHandler(
			ctx,
			*loadedSettings.openTelemetry,
			loadedSettings.serviceVersion,
		)
		if err == nil {
			assembledSinks.handlers = append(assembledSinks.handlers, otelHandler)
			assembledSinks.shutdowns = append(assembledSinks.shutdowns, shutdownProvider)
		} else {
			assembledSinks.failures = append(assembledSinks.failures, sinkFailure{"otlp", err})
		}
	}

	return assembledSinks
}
