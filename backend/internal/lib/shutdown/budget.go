package shutdown

import (
	"context"
	"sync"
	"time"

	"databasus-backend/internal/lib/clock"
)

type Deps struct {
	Clock clock.Source
}

type ContextSpec struct {
	Timeout time.Duration
	Reserve time.Duration
}

type Budget struct {
	clock         clock.Source
	mutex         sync.Mutex
	drainDeadline time.Time
	contexts      map[*budgetContext]struct{}
}

type budgetContext struct {
	cancel   context.CancelFunc
	timer    *time.Timer
	reserve  time.Duration
	deadline time.Time
}

func New(deps Deps) *Budget {
	return &Budget{
		clock:         deps.Clock,
		mutex:         sync.Mutex{},
		drainDeadline: time.Time{},
		contexts:      make(map[*budgetContext]struct{}),
	}
}

func (b *Budget) Begin(timeout time.Duration) time.Time {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if !b.drainDeadline.IsZero() {
		return b.drainDeadline
	}

	b.drainDeadline = b.clock.Now().Add(timeout)
	for activeContext := range b.contexts {
		b.shortenContext(activeContext)
	}

	return b.drainDeadline
}

func (b *Budget) NewContext(ctx context.Context, spec ContextSpec) (context.Context, context.CancelFunc) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	deadline := b.clock.Now().Add(spec.Timeout)
	boundedCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), spec.Timeout)
	activeContext := &budgetContext{cancel: cancel, timer: nil, reserve: spec.Reserve, deadline: deadline}
	b.contexts[activeContext] = struct{}{}
	b.shortenContext(activeContext)

	return boundedCtx, func() {
		b.mutex.Lock()
		defer b.mutex.Unlock()

		delete(b.contexts, activeContext)
		if activeContext.timer != nil {
			activeContext.timer.Stop()
		}
		cancel()
	}
}

func (b *Budget) shortenContext(activeContext *budgetContext) {
	if b.drainDeadline.IsZero() {
		return
	}

	cutoff := b.drainDeadline.Add(-activeContext.reserve)
	if !cutoff.Before(activeContext.deadline) {
		return
	}

	remaining := cutoff.Sub(b.clock.Now())
	if remaining <= 0 {
		activeContext.cancel()
		return
	}

	activeContext.timer = time.AfterFunc(remaining, activeContext.cancel)
}
