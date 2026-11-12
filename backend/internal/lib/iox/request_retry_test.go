package iox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errStorageBusy  = errors.New("service unavailable")
	errAccessDenied = errors.New("access denied")
)

func TestRetryRequest(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when the storage accepts the request after refusals", func(t *testing.T) {
		t.Parallel()

		refusingStorage := &refusingStorage{refusalsLeft: 5, refusal: errStorageBusy}

		err := RetryRequest(t.Context(), newTestRequestRetrySpec(slog.New(slog.DiscardHandler), refusingStorage))

		require.NoError(t, err)
		assert.Equal(t, 6, refusingStorage.sentCount)
	})

	t.Run("returns an error naming the request and the attempts when refusals outlast the budget", func(t *testing.T) {
		t.Parallel()

		refusingStorage := &refusingStorage{refusalsLeft: 100, refusal: errStorageBusy}

		err := RetryRequest(t.Context(), newTestRequestRetrySpec(slog.New(slog.DiscardHandler), refusingStorage))

		require.ErrorIs(t, err, ErrRetryBudgetExhausted)
		require.ErrorIs(t, err, errStorageBusy)
		assert.Contains(t, err.Error(), "part 7 of backups/db.part000001 failed after 11 attempts")
		assert.Equal(t, 11, refusingStorage.sentCount)
	})

	t.Run("returns a permanent error after a single request", func(t *testing.T) {
		t.Parallel()

		refusingStorage := &refusingStorage{refusalsLeft: 100, refusal: errAccessDenied}

		err := RetryRequest(t.Context(), newTestRequestRetrySpec(slog.New(slog.DiscardHandler), refusingStorage))

		require.ErrorIs(t, err, errAccessDenied)
		require.NotErrorIs(t, err, ErrRetryBudgetExhausted)
		assert.Equal(t, 1, refusingStorage.sentCount)
	})

	t.Run("gives every request its own budget when refusals spread across requests exceed it", func(t *testing.T) {
		t.Parallel()

		var totalRefusals int

		for range 4 {
			refusingStorage := &refusingStorage{refusalsLeft: 6, refusal: errStorageBusy}

			err := RetryRequest(t.Context(), newTestRequestRetrySpec(slog.New(slog.DiscardHandler), refusingStorage))

			require.NoError(t, err)
			totalRefusals += refusingStorage.sentCount - 1
		}

		assert.Equal(t, 24, totalRefusals)
	})

	t.Run("logs a warning with the request attributes and the attempt for every repeat", func(t *testing.T) {
		t.Parallel()

		var logOutput bytes.Buffer
		refusingStorage := &refusingStorage{refusalsLeft: 2, refusal: errStorageBusy}

		err := RetryRequest(
			t.Context(),
			newTestRequestRetrySpec(slog.New(slog.NewJSONHandler(&logOutput, nil)), refusingStorage),
		)

		require.NoError(t, err)

		warnings := decodeLogRecords(t, &logOutput)
		require.Len(t, warnings, 2)

		for index, warning := range warnings {
			assert.Equal(t, "WARN", warning["level"])
			assert.Equal(t, "storage request refused, retrying", warning["msg"])
			assert.Equal(t, "backups/db.part000001", warning["object_key"])
			assert.InDelta(t, 7, warning["part_number"], 0)
			assert.InDelta(t, index+1, warning["attempt"], 0)
			assert.InDelta(t, 10, warning["max_attempts"], 0)
			assert.Equal(t, errStorageBusy.Error(), warning["error"])
		}
	})

	t.Run("returns without waiting out the delay when cancelled during the wait", func(t *testing.T) {
		t.Parallel()

		requestCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		refusingStorage := &refusingStorage{refusalsLeft: 100, refusal: errStorageBusy}
		retrySpec := newTestRequestRetrySpec(slog.New(slog.DiscardHandler), refusingStorage)
		retrySpec.RetryPolicySpec = TransferRetryPolicySpec{BaseDelay: time.Hour}

		waitStart := time.Now()
		err := RetryRequest(requestCtx, retrySpec)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, time.Since(waitStart), time.Minute)
		assert.Equal(t, 1, refusingStorage.sentCount)
	})
}

type refusingStorage struct {
	refusalsLeft int
	refusal      error
	sentCount    int
}

func (s *refusingStorage) send(context.Context) error {
	s.sentCount++

	if s.refusalsLeft == 0 {
		return nil
	}

	s.refusalsLeft--

	return s.refusal
}

func newTestRequestRetrySpec(logger *slog.Logger, storage *refusingStorage) RequestRetrySpec {
	return RequestRetrySpec{
		Logger:             logger,
		RetryPolicySpec:    TransferRetryPolicySpec{MaxAttemptsWithoutProgress: 0, BaseDelay: time.Millisecond},
		RequestDescription: "part 7 of backups/db.part000001",
		LogAttributes: []slog.Attr{
			slog.String("object_key", "backups/db.part000001"),
			slog.Int("part_number", 7),
		},
		IsRetryableError: isStorageBusy,
		Send:             storage.send,
	}
}

func decodeLogRecords(t *testing.T, logOutput *bytes.Buffer) []map[string]any {
	t.Helper()

	var records []map[string]any

	for line := range strings.Lines(logOutput.String()) {
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))

		records = append(records, record)
	}

	return records
}

func isStorageBusy(err error) bool {
	return errors.Is(err, errStorageBusy)
}
