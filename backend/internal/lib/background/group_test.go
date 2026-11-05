package background

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroup(t *testing.T) {
	t.Parallel()

	t.Run("logs a panic with its stack and keeps the process alive", func(t *testing.T) {
		t.Parallel()
		logOutput := &lockedBuffer{}
		group := New(Deps{Logger: slog.New(slog.NewJSONHandler(logOutput, nil))})

		group.Go(t.Context(), "exploding", func(context.Context) error {
			panic("boom")
		})

		require.NoError(t, group.Wait(t.Context()))
		assert.Contains(t, logOutput.String(), `"msg":"background goroutine panicked"`)
		assert.Contains(t, logOutput.String(), `"goroutine":"exploding"`)
		assert.Contains(t, logOutput.String(), "group_test.go")
	})

	t.Run("logs an error the function returns", func(t *testing.T) {
		t.Parallel()
		logOutput := &lockedBuffer{}
		group := New(Deps{Logger: slog.New(slog.NewJSONHandler(logOutput, nil))})

		group.Go(t.Context(), "failing", func(context.Context) error {
			return errors.New("disk is full")
		})

		require.NoError(t, group.Wait(t.Context()))
		assert.Contains(t, logOutput.String(), "disk is full")
	})

	t.Run("returns from wait after every function returned", func(t *testing.T) {
		t.Parallel()
		group := New(Deps{Logger: slog.New(slog.DiscardHandler)})
		loopContext, stopLoops := context.WithCancel(t.Context())

		for _, name := range []string{"first", "second"} {
			group.Go(loopContext, name, func(ctx context.Context) error {
				<-ctx.Done()

				return ctx.Err()
			})
		}

		assert.Equal(t, 2, group.GetRunningCount())

		stopLoops()

		require.NoError(t, group.Wait(t.Context()))
		assert.Zero(t, group.GetRunningCount())
	})

	t.Run("returns from wait when its context ends before a function returns", func(t *testing.T) {
		t.Parallel()
		group := New(Deps{Logger: slog.New(slog.DiscardHandler)})
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })

		group.Go(t.Context(), "stuck", func(context.Context) error {
			<-release

			return nil
		})

		waitContext, cancelWait := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancelWait()

		err := group.Wait(waitContext)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, 1, group.GetRunningCount())
	})
}

type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(line []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.buffer.Write(line)
}

func (b *lockedBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.buffer.String()
}
