package report

import (
	"context"
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand/v2"
	"time"

	"databasus-backend/internal/lib/background"
)

func (s *Service) GetJobs() []background.Job {
	return []background.Job{{Name: "telemetry_ping", Interval: 15 * time.Second, Run: s.runPass}}
}

func (s *Service) runPass(ctx context.Context, now time.Time) error {
	if s.deps.IsDisabled {
		return nil
	}

	if s.nextDueAt == nil {
		dueAt := now.Add(time.Minute)
		s.nextDueAt = &dueAt
		return nil
	}
	if now.Before(*s.nextDueAt) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	err := s.buildAndSend(ctx)
	delay := 24*time.Hour + time.Duration(rand.Int64N(int64(20*time.Minute))) - 10*time.Minute
	if err != nil {
		delay = s.backoff
		s.backoff = min(s.backoff*2, 24*time.Hour)
	} else {
		s.backoff = time.Minute
	}

	dueAt := s.deps.Clock.Now().UTC().Add(delay)
	s.nextDueAt = &dueAt

	return err
}
