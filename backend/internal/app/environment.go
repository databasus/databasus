// Package app is the composition root: it loads the configuration, opens the database, applies the
// migrations, wires the old code and runs the server or a console command.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/util/logger"
)

type Environment struct {
	Config  config.Config
	Logging *logging.Logging
	Clock   clock.Source
	// Mailer sends the instance's own mail through the operator's server. A test application
	// records the messages instead.
	Mailer mailer.Sender
	// ExternalEndpoints are the addresses of the services sign-in calls. A test application points
	// them at fakes.
	ExternalEndpoints ExternalEndpoints
}

// ExternalEndpoints are the addresses of the identity providers and of Cloudflare Turnstile.
type ExternalEndpoints struct {
	GitHubAuthURL      string
	GitHubTokenURL     string
	GitHubUserURL      string
	GitHubEmailsURL    string
	GoogleAuthURL      string
	GoogleTokenURL     string
	GoogleUserInfoURL  string
	TurnstileVerifyURL string
}

// GetPublicExternalEndpoints returns the real addresses of the services sign-in calls.
func GetPublicExternalEndpoints() ExternalEndpoints {
	return ExternalEndpoints{
		GitHubAuthURL:      github.Endpoint.AuthURL,
		GitHubTokenURL:     github.Endpoint.TokenURL,
		GitHubUserURL:      "https://api.github.com/user",
		GitHubEmailsURL:    "https://api.github.com/user/emails",
		GoogleAuthURL:      google.Endpoint.AuthURL,
		GoogleTokenURL:     google.Endpoint.TokenURL,
		GoogleUserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		TurnstileVerifyURL: "https://challenges.cloudflare.com/turnstile/v0/siteverify",
	}
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

// NewEnvironment builds the logging and the mailer for a configuration that is already loaded,
// such as the one a test application assembles with a clock it moves, and points sign-in at the
// public external endpoints.
func NewEnvironment(ctx context.Context, loadedConfig config.Config, clockSource clock.Source) (*Environment, error) {
	loadedLogging, err := logging.New(ctx, loadedConfig)
	if err != nil {
		logging.NewStdout().GetLogger().ErrorContext(ctx, "logging could not be configured", slog.Any("error", err))

		return nil, fmt.Errorf("configure logging: %w", err)
	}

	logger.Install(loadedLogging)

	return &Environment{
		Config:  loadedConfig,
		Logging: loadedLogging,
		Clock:   clockSource,
		Mailer: mailer.NewInstanceSender(mailer.Deps{
			Settings: loadedConfig.GetInstanceMailSettings(),
			Logger:   loadedLogging.GetLogger(),
		}),
		ExternalEndpoints: GetPublicExternalEndpoints(),
	}, nil
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
