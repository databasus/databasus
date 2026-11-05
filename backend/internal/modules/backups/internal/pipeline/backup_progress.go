package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"databasus-backend/internal/lib/background"
)

type progressReportSpec struct {
	interval        time.Duration
	getStoredSizeMb func() float64
	listener        func(storedSizeMb float64)
	logger          *slog.Logger
}

// startReportingProgress calls the listener on every tick, also when the size has not changed,
// because the listener records the duration too. The calls run apart from the goroutine that copies
// the dump, so the copy never waits for them; a tick that falls due during a call is dropped. The
// returned function stops the reporting and waits for a call in flight, so no call outlives it.
func startReportingProgress(ctx context.Context, spec progressReportSpec) (stopReporting func()) {
	reportCtx, cancel := context.WithCancel(ctx)
	group := background.New(background.Deps{Logger: spec.logger})

	group.Go(reportCtx, "logical backup progress", func(ctx context.Context) error {
		ticker := time.NewTicker(spec.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				spec.listener(spec.getStoredSizeMb())
			}
		}
	})

	return sync.OnceFunc(func() {
		cancel()
		_ = group.Wait(context.WithoutCancel(ctx))
	})
}
