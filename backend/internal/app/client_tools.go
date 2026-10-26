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

// checkClientTools logs the verdict of every bundle and returns ErrFatalClientTools when a fatal
// one fails, or the reason the bundle root was not found. A broken bundle that is not fatal only
// disables the versions it serves.
func (a *App) checkClientTools(ctx context.Context) error {
	if a.environment.clientToolsDirErr != nil {
		a.logger.ErrorContext(
			ctx,
			"client tools could not be found",
			slog.Any("error", a.environment.clientToolsDirErr),
		)

		return a.environment.clientToolsDirErr
	}

	isFatalBundleBroken := false

	for _, result := range a.clientToolsChecker.Check(ctx) {
		bundleLogger := a.logger.With(
			slog.String("db", result.Bundle.Engine),
			slog.String("version", result.Bundle.Version),
			slog.String("path", result.Bundle.BinDir),
		)

		if result.IsPassed() {
			if a.environment.Config.ShowDbInstallationVerificationLogs {
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
