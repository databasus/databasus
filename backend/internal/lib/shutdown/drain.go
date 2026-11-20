package shutdown

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type Step struct {
	Name string
	Run  func(ctx context.Context) error
}

type DrainSpec struct {
	Steps    []Step
	Deadline time.Time
	// FlushLogs runs after the steps, so the logs of in-flight requests and tasks are included. It
	// gets its own deadline, so a slow drain cannot hand it an already expired context. It is nil
	// when the process goes on after the drain and keeps logging.
	FlushLogs    func(ctx context.Context) error
	FlushTimeout time.Duration
	Logger       *slog.Logger
}

// Drain reports whether every step finished without an error before the deadline. The stop signal
// has already canceled ctx; draining still needs its own deadline.
func Drain(ctx context.Context, spec DrainSpec) bool {
	drainCtx, cancelDrain := context.WithDeadline(context.WithoutCancel(ctx), spec.Deadline)
	defer cancelDrain()

	var runningSteps sync.WaitGroup
	var stepsMutex sync.Mutex
	var hasFailedStep atomic.Bool
	unfinishedSteps := make(map[string]struct{}, len(spec.Steps))
	for _, step := range spec.Steps {
		unfinishedSteps[step.Name] = struct{}{}
	}

	for _, step := range spec.Steps {
		runningSteps.Go(func() {
			defer func() {
				stepsMutex.Lock()
				delete(unfinishedSteps, step.Name)
				stepsMutex.Unlock()
			}()
			if err := step.Run(drainCtx); err != nil {
				hasFailedStep.Store(true)
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

	isDrained := false

	select {
	case <-areStepsFinished:
		isDrained = !hasFailedStep.Load()
	case <-drainCtx.Done():
		var unfinishedNames []string
		stepsMutex.Lock()
		for stepName := range unfinishedSteps {
			unfinishedNames = append(unfinishedNames, stepName)
		}
		stepsMutex.Unlock()

		slices.Sort(unfinishedNames)
		spec.Logger.WarnContext(drainCtx, "shutdown steps did not finish within the drain limit",
			slog.Time("drain_deadline", spec.Deadline), slog.Any("unfinished_steps", unfinishedNames))
	}

	if spec.FlushLogs != nil {
		flushCtx, cancelFlush := context.WithTimeout(context.WithoutCancel(ctx), spec.FlushTimeout)
		defer cancelFlush()

		_ = spec.FlushLogs(flushCtx)
	}

	return isDrained
}
