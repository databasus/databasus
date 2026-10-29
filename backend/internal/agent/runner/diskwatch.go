package runner

import (
	"context"
	"log/slog"
	"time"
)

var diskWatchInterval = 3 * time.Second

// diskWatcher polls the job's written bytes and fires onLimitExceeded once when
// the footprint reaches the server-computed ceiling. It never trips on a probe
// error (a transient daemon hiccup must not fail a healthy restore). The caller
// owns the reaction: onLimitExceeded records the trip and cancels the job.
type diskWatcher struct {
	probe           diskUsageProber
	ceilingBytes    int64
	interval        time.Duration
	onLimitExceeded func()
	log             *slog.Logger
}

func newDiskWatcher(
	probe diskUsageProber, ceilingBytes int64, onLimitExceeded func(), log *slog.Logger,
) *diskWatcher {
	return &diskWatcher{
		probe:           probe,
		ceilingBytes:    ceilingBytes,
		interval:        diskWatchInterval,
		onLimitExceeded: onLimitExceeded,
		log:             log,
	}
}

func (w *diskWatcher) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		used, err := w.probe.GetDiskUsageBytes(ctx)
		switch {
		case err != nil:
			w.log.DebugContext(ctx, "disk watch probe failed, will retry next tick", slog.Any("error", err))
		case used >= w.ceilingBytes:
			w.log.WarnContext(
				ctx,
				"job disk usage reached ceiling; aborting restore",
				slog.Any("used_bytes", used),
				slog.Any("ceiling_bytes", w.ceilingBytes),
			)
			w.onLimitExceeded()

			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
