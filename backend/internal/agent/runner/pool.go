package runner

import (
	"context"
	"sync"

	"databasus-backend/internal/lib/background"
)

type Pool struct {
	slots chan struct{}
	wg    sync.WaitGroup
}

func NewPool(maxConcurrentJobs int) *Pool {
	return &Pool{slots: make(chan struct{}, maxConcurrentJobs)}
}

func (p *Pool) IsSaturated() bool {
	return len(p.slots) == cap(p.slots)
}

func (p *Pool) Go(ctx context.Context, group *background.Group, fn func(context.Context) error) {
	select {
	case p.slots <- struct{}{}:
	case <-ctx.Done():
		return
	}
	p.wg.Add(1)
	group.Go(ctx, "verification_job", func(ctx context.Context) error {
		defer p.wg.Done()
		defer func() { <-p.slots }()

		return fn(ctx)
	})
}

func (p *Pool) Wait() {
	p.wg.Wait()
}
