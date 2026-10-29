package app

import (
	"context"

	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	"databasus-backend/internal/features/telemetry"
)

type backgroundLoop struct {
	name string
	run  func(ctx context.Context)
}

func (a *App) startBackgroundLoops(ctx context.Context) {
	loops := []backgroundLoop{
		{"healthcheck attempt background service", healthcheck_attempt.GetHealthcheckAttemptBackgroundService().Run},
		{"telemetry background service", telemetry.GetTelemetryBackgroundService().Run},
	}

	a.logger.InfoContext(ctx, "starting background tasks")

	for _, loop := range loops {
		a.backgroundGroup.Go(ctx, loop.name, func(loopCtx context.Context) error {
			loop.run(loopCtx)

			return nil
		})
	}
}
