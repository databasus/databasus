package background

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/logging"
)

// RunnerDeps is separate from Deps, because Group has no use for a clock.
type RunnerDeps struct {
	Logger *slog.Logger
	Clock  clock.Source
}

// Job is a periodic piece of work. Run does one pass, and now is the clock's time at its start.
type Job struct {
	Name       string
	Interval   time.Duration
	Run        func(ctx context.Context, now time.Time) error
	WakeSignal <-chan struct{}
}

// Runner runs every job in its own loop. The first pass runs right away, and later passes start
// after the interval or a wake signal. A signal buffered during a pass starts another pass after it
// finishes. Intervals use real time; tests normally run passes explicitly with RunPass.
type Runner struct {
	logger    *slog.Logger
	clock     clock.Source
	jobs      map[string]*scheduledJob
	isStarted atomic.Bool
	waitGroup sync.WaitGroup
}

type scheduledJob struct {
	job Job
	// passMutex keeps a pass from RunPass and a pass of the loop from overlapping.
	passMutex     sync.Mutex
	snapshotMutex sync.RWMutex
	snapshot      JobSnapshot
}

// NewRunner panics when two jobs share a name, because RunPass could not tell them apart.
func NewRunner(deps RunnerDeps, jobs ...Job) *Runner {
	scheduledJobs := make(map[string]*scheduledJob, len(jobs))

	for _, job := range jobs {
		if _, isDuplicate := scheduledJobs[job.Name]; isDuplicate {
			panic(fmt.Sprintf("background job %q is registered twice", job.Name))
		}

		scheduledJobs[job.Name] = &scheduledJob{
			job: job, passMutex: sync.Mutex{}, snapshotMutex: sync.RWMutex{},
			snapshot: JobSnapshot{Name: job.Name, Interval: job.Interval, ActivatedAt: deps.Clock.Now().UTC()},
		}
	}

	return &Runner{
		logger:    deps.Logger,
		clock:     deps.Clock,
		jobs:      scheduledJobs,
		isStarted: atomic.Bool{},
		waitGroup: sync.WaitGroup{},
	}
}

// Start starts the loop of every job until ctx ends. A second Start panics, because it would run
// every job twice.
func (r *Runner) Start(ctx context.Context) {
	if !r.isStarted.CompareAndSwap(false, true) {
		panic("background runner is started twice")
	}

	for _, job := range r.jobs {
		job.snapshotMutex.Lock()
		if job.snapshot.LastStartedAt == nil {
			job.snapshot.ActivatedAt = r.clock.Now().UTC()
		}
		job.snapshotMutex.Unlock()
		r.waitGroup.Go(func() {
			r.runLoop(ctx, job)
		})
	}
}

// RunPass runs one pass of the named job and returns its error.
func (r *Runner) RunPass(ctx context.Context, name string) error {
	job, isFound := r.jobs[name]
	if !isFound {
		return fmt.Errorf("%w: %s", ErrUnknownJob, name)
	}

	return r.runPass(ctx, job)
}

// Wait returns once every loop has returned, or with ctx's error when ctx ends first.
func (r *Runner) Wait(ctx context.Context) error {
	isDone := make(chan struct{})

	go func() {
		r.waitGroup.Wait()
		close(isDone)
	}()

	select {
	case <-isDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for background jobs: %w", ctx.Err())
	}
}

func (r *Runner) runLoop(ctx context.Context, job *scheduledJob) {
	for ctx.Err() == nil {
		_ = r.runPass(ctx, job)

		timer := time.NewTimer(job.job.Interval)

		select {
		case <-timer.C:
		case <-job.job.WakeSignal:
			timer.Stop()
		case <-ctx.Done():
			timer.Stop()

			return
		}
	}
}

func (r *Runner) runPass(ctx context.Context, job *scheduledJob) (passError error) {
	job.passMutex.Lock()
	defer job.passMutex.Unlock()

	passContext := logging.With(ctx,
		slog.String("job_name", job.job.Name),
		slog.String("job_id", uuid.NewString()),
	)

	now := r.clock.Now().UTC()
	job.snapshotMutex.Lock()
	job.snapshot.IsRunning = true
	job.snapshot.LastStartedAt = &now
	job.snapshotMutex.Unlock()

	defer func() { r.completePass(job, passError) }()

	passError = RunRecovering(passContext, func(ctx context.Context) error {
		return job.job.Run(ctx, now)
	})

	var panicError *PanicError

	switch {
	case passError == nil:
	case errors.As(passError, &panicError):
		r.logger.ErrorContext(passContext, "background job pass panicked",
			slog.Any("error", panicError),
			slog.String("stack", panicError.Stack),
		)
	case ctx.Err() != nil:
		r.logger.InfoContext(passContext, "background job pass interrupted", slog.Any("error", passError))
	default:
		r.logger.ErrorContext(passContext, "background job pass failed", slog.Any("error", passError))
	}

	return passError
}

func (r *Runner) completePass(job *scheduledJob, err error) {
	var panicError *PanicError
	outcome := PassSucceeded
	switch {
	case errors.As(err, &panicError):
		outcome = PassPanicked
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		outcome = PassCanceled
	case err != nil:
		outcome = PassFailed
	}

	completedAt := r.clock.Now().UTC()
	job.snapshotMutex.Lock()
	job.snapshot.IsRunning = false
	job.snapshot.LastCompletedAt = &completedAt
	job.snapshot.LastOutcome = outcome
	job.snapshot.LastError = err
	job.snapshotMutex.Unlock()
}
