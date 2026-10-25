package signin

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/ratelimit"
)

type failingRateLimitCounter struct{}

func (failingRateLimitCounter) RecordAttemptAndCheckIsAllowed(context.Context, ratelimit.Attempt) (bool, error) {
	return false, errors.New("counter failed")
}

func TestCheckRateLimit(t *testing.T) {
	t.Run("refuses the attempt and logs no address when the counter fails", func(t *testing.T) {
		var logOutput bytes.Buffer
		service := &Service{
			rateLimiter: failingRateLimitCounter{},
			logger:      slog.New(slog.NewTextHandler(&logOutput, nil)),
		}
		privateEmail := "private@example.com"

		err := service.checkRateLimit(t.Context(), ratelimit.Attempt{
			Scope:      signInRateLimitScope,
			Identifier: privateEmail,
			Limit:      signInRateLimit,
			Window:     time.Minute,
		})

		require.ErrorIs(t, err, ErrRateLimited)
		assert.Contains(t, logOutput.String(), "failed to evaluate a rate limit")
		assert.NotContains(t, logOutput.String(), privateEmail)
	})
}
