package app

import (
	"context"
	"log/slog"
	"slices"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/engines/mysql"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

func newClientToolsChecker(clientToolsDir string) *clienttools.Checker {
	return clienttools.NewChecker(slices.Concat(
		pgshared.GetClientToolBundles(clientToolsDir),
		mysql.GetClientToolBundles(clientToolsDir),
		mariadb.GetClientToolBundles(clientToolsDir),
		mongodb.GetClientToolBundles(clientToolsDir),
	))
}

// getClientToolsChecker builds the checker on first use, from the bundle root a test application may
// have overridden. The same checker serves the health check, which reuses its verdicts.
func (e *Environment) getClientToolsChecker() *clienttools.Checker {
	if e.clientToolsChecker == nil {
		e.clientToolsChecker = newClientToolsChecker(e.ClientToolsDir)
	}

	return e.clientToolsChecker
}

// CheckClientTools logs the verdict of every bundle and returns ErrFatalClientTools when a fatal one
// fails, or the reason the bundle root was not found. A broken bundle that is not fatal only disables
// the versions it serves. The server runs it before it opens the database, so an image with a broken
// fatal bundle leaves the schema at the version the previous image can run.
func (e *Environment) CheckClientTools(ctx context.Context) error {
	logger := e.Logging.GetLogger()

	if e.clientToolsDirErr != nil {
		logger.ErrorContext(
			ctx,
			"client tools could not be found",
			slog.Any("error", e.clientToolsDirErr),
		)

		return e.clientToolsDirErr
	}

	isFatalBundleBroken := false

	for _, result := range e.getClientToolsChecker().Check(ctx) {
		bundleLogger := logger.With(
			slog.String("db", result.Bundle.Engine),
			slog.String("version", result.Bundle.Version),
			slog.String("path", result.Bundle.BinDir),
		)

		if result.IsPassed() {
			if e.Config.ShowDbInstallationVerificationLogs {
				bundleLogger.InfoContext(ctx, "client tools verified")
			}

			continue
		}

		for _, err := range result.Errors {
			if result.Bundle.IsFatal {
				bundleLogger.ErrorContext(ctx, "client tools check failed", slog.Any("error", err))
			} else {
				bundleLogger.WarnContext(ctx, "client tools check failed - support disabled", slog.Any("error", err))
			}
		}

		if result.Bundle.IsFatal {
			isFatalBundleBroken = true
		}
	}

	if isFatalBundleBroken {
		return ErrFatalClientTools
	}

	return nil
}
