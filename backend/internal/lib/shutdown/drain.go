package shutdown

import (
	"context"
	"log/slog"
	"slices"
	"sync"
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
	// gets its own deadline, so a slow drain cannot hand it an already expired context.
	FlushLogs    func(ctx context.Context) error
	FlushTimeout time.Duration
	Logger       *slog.Logger
}

// The stop signal has already canceled ctx; draining still needs its own deadline.
func Drain(ctx context.Context, spec DrainSpec) {
	drainCtx, cancelDrain := context.WithDeadline(context.WithoutCancel(ctx), spec.Deadline)
	defer cancelDrain()

	var runningSteps sync.WaitGroup
	var stepsMutex sync.Mutex
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

	flushCtx, cancelFlush := context.WithTimeout(context.WithoutCancel(ctx), spec.FlushTimeout)
	defer cancelFlush()

	_ = spec.FlushLogs(flushCtx)
}
