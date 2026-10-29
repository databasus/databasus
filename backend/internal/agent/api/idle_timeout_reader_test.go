package api

import (
	"context"
	"fmt"
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadThroughIdleTimeoutReader(t *testing.T) {
	t.Run("keeps the context active when bytes keep flowing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)

			pr, pw := io.Pipe()

			idleReader := NewIdleTimeoutReader(pr, 100*time.Millisecond, cancel)
			defer idleReader.Stop()

			go func() {
				for range 5 {
					_, _ = pw.Write([]byte("data"))
					<-time.After(50 * time.Millisecond)
				}

				_ = pw.Close()
			}()

			data, err := io.ReadAll(idleReader)

			require.NoError(t, err)
			assert.Equal(t, "datadatadatadatadata", string(data))
			assert.NoError(t, ctx.Err(), "context should not be cancelled when bytes flow continuously")
		})
	})
	t.Run("cancels context when no bytes are transmitted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)

			pr, _ := io.Pipe()

			idleReader := NewIdleTimeoutReader(pr, 100*time.Millisecond, cancel)
			defer idleReader.Stop()

			<-ctx.Done()
			assert.Contains(t, context.Cause(ctx).Error(), "upload idle timeout")
		})
	})
	t.Run("cancels context when bytes stop mid-stream", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)

			pr, pw := io.Pipe()

			idleReader := NewIdleTimeoutReader(pr, 100*time.Millisecond, cancel)
			defer idleReader.Stop()

			go func() {
				_, _ = pw.Write([]byte("initial"))
				// Stop writing — simulate stalled source
			}()

			buf := make([]byte, 1024)
			n, _ := idleReader.Read(buf)
			assert.Equal(t, "initial", string(buf[:n]))

			<-ctx.Done()
			assert.Contains(t, context.Cause(ctx).Error(), "upload idle timeout")
		})
	})
	t.Run("propagates error when the reader returns an error", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)

			pr, pw := io.Pipe()

			idleReader := NewIdleTimeoutReader(pr, 5*time.Second, cancel)
			defer idleReader.Stop()

			readerErr := fmt.Errorf("test read error")
			_ = pw.CloseWithError(readerErr)

			buf := make([]byte, 1024)
			_, err := idleReader.Read(buf)

			require.ErrorIs(t, err, readerErr)

			<-time.After(10 * time.Second)
			assert.NoError(t, ctx.Err(), "context should not be cancelled after reader error stops the timer")
		})
	})
}

func TestStopIdleTimeoutReader(t *testing.T) {
	t.Run("keeps the context active when stopped before the timeout", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)

			pr, _ := io.Pipe()

			idleReader := NewIdleTimeoutReader(pr, 100*time.Millisecond, cancel)
			idleReader.Stop()

			<-time.After(200 * time.Millisecond)

			assert.NoError(t, ctx.Err(), "context should not be cancelled when reader is stopped before timeout")
		})
	})
}
