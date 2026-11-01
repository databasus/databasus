package background

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/clock"
)

func TestJobSnapshots(t *testing.T) {
	t.Parallel()
	t.Run("retains completed outcomes when another pass blocks", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
		passError := errors.New("private failure")
		isBlocked := false
		started := make(chan struct{})
		release := make(chan struct{})

		runner := NewRunner(RunnerDeps{Logger: slog.New(slog.DiscardHandler), Clock: manualClock}, Job{
			Name: "work", Interval: time.Minute, Run: func(context.Context, time.Time) error {
				if isBlocked {
					close(started)
					<-release
				}
				manualClock.Advance(time.Second)
				return passError
			},
		}, newIdleJob("alphabetical"))

		snapshots := runner.GetSnapshots()
		require.Equal(t, "alphabetical", snapshots[0].Name)
		require.Nil(t, snapshots[1].LastCompletedAt)
		require.Equal(t, manualClock.Now(), snapshots[1].ActivatedAt)

		require.ErrorIs(t, runner.RunPass(t.Context(), "work"), passError)
		completedAt := manualClock.Now()
		isBlocked = true
		var passes sync.WaitGroup
		passes.Go(func() { assert.ErrorIs(t, runner.RunPass(t.Context(), "work"), passError) })
		<-started

		snapshots = runner.GetSnapshots()
		assert.True(t, snapshots[1].IsRunning)
		assert.Equal(t, PassFailed, snapshots[1].LastOutcome)
		assert.Equal(t, completedAt, *snapshots[1].LastCompletedAt)
		require.ErrorIs(t, snapshots[1].LastError, passError)
		*snapshots[1].LastCompletedAt = time.Time{}
		assert.Equal(t, completedAt, *runner.GetSnapshots()[1].LastCompletedAt)

		close(release)
		passes.Wait()
		assert.False(t, runner.GetSnapshots()[1].IsRunning)
	})
	t.Run("records outcomes when passes panic or are canceled", func(t *testing.T) {
		t.Parallel()
		for _, scenario := range []struct {
			name    string
			outcome PassOutcome
			run     func(context.Context, time.Time) error
		}{
			{"panic", PassPanicked, func(context.Context, time.Time) error { panic("secret") }},
			{"cancellation", PassCanceled, func(context.Context, time.Time) error { return context.Canceled }},
			{"success", PassSucceeded, func(context.Context, time.Time) error { return nil }},
		} {
			runner := NewRunner(
				newRunnerDeps(slog.New(slog.DiscardHandler)),
				Job{Name: scenario.name, Interval: time.Second, Run: scenario.run},
			)
			_ = runner.RunPass(t.Context(), scenario.name)
			snapshot := runner.GetSnapshots()[0]
			assert.Equal(t, scenario.outcome, snapshot.LastOutcome)
			assert.NotNil(t, snapshot.LastCompletedAt)
			assert.False(t, snapshot.IsRunning)
		}
	})
	t.Run("replaces the panic outcome and clears its error when the same job succeeds", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
		shouldPanic := true
		runner := NewRunner(RunnerDeps{Logger: slog.New(slog.DiscardHandler), Clock: manualClock}, Job{
			Name: "recovering", Interval: time.Second, Run: func(context.Context, time.Time) error {
				if shouldPanic {
					panic("private failure")
				}
				return nil
			},
		})
		require.Error(t, runner.RunPass(t.Context(), "recovering"))

		panickedPass := runner.GetSnapshots()[0]
		require.Equal(t, PassPanicked, panickedPass.LastOutcome)
		require.Error(t, panickedPass.LastError)
		require.NotNil(t, panickedPass.LastCompletedAt)

		manualClock.Advance(time.Minute)
		shouldPanic = false
		require.NoError(t, runner.RunPass(t.Context(), "recovering"))

		recoveredPass := runner.GetSnapshots()[0]
		require.NotNil(t, recoveredPass.LastCompletedAt)
		assert.True(t, recoveredPass.LastCompletedAt.After(*panickedPass.LastCompletedAt))
		assert.Equal(t, manualClock.Now(), *recoveredPass.LastCompletedAt)
		assert.Equal(t, PassSucceeded, recoveredPass.LastOutcome)
		require.NoError(t, recoveredPass.LastError)
		assert.False(t, recoveredPass.IsRunning)
	})
	t.Run("permits concurrent reads when automatic and explicit passes run", func(t *testing.T) {
		t.Parallel()
		runner := NewRunner(newRunnerDeps(slog.New(slog.DiscardHandler)), newIdleJob("work"))
		startUntilTestEnds(t, runner)
		var readers sync.WaitGroup
		for range 20 {
			readers.Go(func() {
				for range 20 {
					require.Len(t, runner.GetSnapshots(), 1)
					require.NoError(t, runner.RunPass(t.Context(), "work"))
				}
			})
		}
		readers.Wait()
	})
	t.Run("resets activation when production starts before its first pass", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
		runner := NewRunner(RunnerDeps{Logger: slog.New(slog.DiscardHandler), Clock: manualClock}, newIdleJob("work"))
		manualClock.Advance(time.Hour)
		startUntilTestEnds(t, runner)
		assert.Equal(t, manualClock.Now(), runner.GetSnapshots()[0].ActivatedAt)
	})
}
