// Package app is the composition root: it loads the configuration, opens the database, applies the
// migrations, wires the old code and runs the server or a console command.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/util/logger"
)

type Environment struct {
	Config  config.Config
	Logging *logging.Logging
	Clock   clock.Source
}

// LoadEnvironment reads the configuration and builds the logging. When the configuration cannot be
// loaded, it reports the error through a stdout-only logger, because the settings of the other log
// sinks are part of what failed.
func LoadEnvironment(ctx context.Context, spec config.LoadSpec) (*Environment, error) {
	loadedConfig, report, err := config.Load(spec)
	if err != nil {
		stdoutLogger := logging.NewStdout().GetLogger()
		logReport(ctx, stdoutLogger, report)
		stdoutLogger.ErrorContext(ctx, "configuration could not be loaded", slog.Any("error", err))

		return nil, fmt.Errorf("load the configuration: %w", err)
	}

	environment, err := NewEnvironment(ctx, loadedConfig, clock.NewSystem())
	if err != nil {
		return nil, err
	}

	appLogger := environment.Logging.GetLogger()
	logReport(ctx, appLogger, report)
	appLogger.InfoContext(ctx, "environment variables loaded", slog.String("mode", string(loadedConfig.EnvMode)))

	return environment, nil
}

// NewEnvironment builds the logging for a configuration that is already loaded, such as the one a
// test application assembles with a clock it moves.
func NewEnvironment(ctx context.Context, loadedConfig config.Config, clockSource clock.Source) (*Environment, error) {
	loadedLogging, err := logging.New(ctx, loadedConfig)
	if err != nil {
		logging.NewStdout().GetLogger().ErrorContext(ctx, "logging could not be configured", slog.Any("error", err))

		return nil, fmt.Errorf("configure logging: %w", err)
	}

	logger.Install(loadedLogging)

	return &Environment{Config: loadedConfig, Logging: loadedLogging, Clock: clockSource}, nil
}

func logReport(ctx context.Context, reportLogger *slog.Logger, report config.LoadReport) {
	if report.IsEnvFileLoaded {
		reportLogger.InfoContext(ctx, "successfully loaded .env", slog.String("path", report.EnvFilePath))
	} else {
		reportLogger.InfoContext(ctx, "no .env file found, using the process environment only",
			slog.String("path", report.EnvFilePath))
	}

	if report.PublishedDatabaseDsnReadError != nil {
		reportLogger.WarnContext(ctx, "could not read the published database connection string",
			slog.Any("error", report.PublishedDatabaseDsnReadError))
	}

	if report.IsPublishedDatabaseDsnUsed {
		reportLogger.InfoContext(ctx, "using the database connection string published by container startup")
	}
}
