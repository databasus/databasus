package shutdown

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Step struct {
	Name string
	Run  func(ctx context.Context) error
}

type DrainSpec struct {
	Steps        []Step
	DrainTimeout time.Duration
	// FlushLogs runs after the steps, so the logs of in-flight requests and tasks are included. It
	// gets its own deadline, so a slow drain cannot hand it an already expired context.
	FlushLogs    func(ctx context.Context) error
	FlushTimeout time.Duration
	Logger       *slog.Logger
}

// Drain runs the steps in parallel and waits for them until DrainTimeout. A step that has not
// finished by then is left behind, so the process exits on time. ctx only carries values: its
// cancellation, which is usually the stop signal itself, does not cut the drain short.
func Drain(ctx context.Context, spec DrainSpec) {
	drainCtx, cancelDrain := context.WithTimeout(context.WithoutCancel(ctx), spec.DrainTimeout)
	defer cancelDrain()

	var runningSteps sync.WaitGroup

	for _, step := range spec.Steps {
		runningSteps.Go(func() {
			if err := step.Run(drainCtx); err != nil {
				spec.Logger.WarnContext(drainCtx, "shutdown step did not finish cleanly",
					slog.String("step", step.Name), slog.Any("error", err))

				return
			}

			spec.Logger.InfoContext(drainCtx, "shutdown step finished", slog.String("step", step.Name))
		})
	}

	areStepsFinished := make(chan struct{})

	go func() {
		runningSteps.Wait()
		close(areStepsFinished)
	}()

	select {
	case <-areStepsFinished:
	case <-drainCtx.Done():
		spec.Logger.WarnContext(drainCtx, "shutdown steps did not finish within the drain limit",
			slog.Duration("drain_timeout", spec.DrainTimeout))
	}

	flushCtx, cancelFlush := context.WithTimeout(context.WithoutCancel(ctx), spec.FlushTimeout)
	defer cancelFlush()

	_ = spec.FlushLogs(flushCtx)
}
