// Package background runs goroutines that stop with their context and that shutdown can wait for.
package background

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

type Deps struct {
	Logger *slog.Logger
}

// Group starts goroutines and waits for them. A panic in one is logged as an error with its stack
// instead of killing the process.
type Group struct {
	logger       *slog.Logger
	waitGroup    sync.WaitGroup
	runningCount atomic.Int64
}

func New(deps Deps) *Group {
	return &Group{logger: deps.Logger}
}

// Go runs fn in a goroutine. An error fn returns is logged, unless it only reports that ctx ended.
func (g *Group) Go(ctx context.Context, name string, fn func(ctx context.Context) error) {
	g.runningCount.Add(1)

	g.waitGroup.Go(func() {
		defer g.runningCount.Add(-1)

		err := runRecovering(ctx, fn)
		if err == nil || (ctx.Err() != nil && errors.Is(err, ctx.Err())) {
			return
		}

		var panicError *PanicError
		if errors.As(err, &panicError) {
			g.logger.ErrorContext(ctx, "background goroutine panicked",
				slog.String("goroutine", name),
				slog.Any("error", panicError),
				slog.String("stack", panicError.Stack),
			)

			return
		}

		g.logger.ErrorContext(
			ctx,
			"background goroutine failed",
			slog.String("goroutine", name),
			slog.Any("error", err),
		)
	})
}

// Wait returns once every goroutine of the group has returned, or with ctx's error when ctx ends
// first.
func (g *Group) Wait(ctx context.Context) error {
	isDone := make(chan struct{})

	go func() {
		g.waitGroup.Wait()
		close(isDone)
	}()

	select {
	case <-isDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for background goroutines: %w", ctx.Err())
	}
}

func (g *Group) GetRunningCount() int {
	return int(g.runningCount.Load())
}

func runRecovering(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = &PanicError{Value: recovered, Stack: string(debug.Stack())}
		}
	}()

	return fn(ctx)
}
