package walstreaming_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type historyObserver struct {
	slog.Handler
	events chan string
}

func (observer historyObserver) Handle(_ context.Context, record slog.Record) error {
	if record.Message == "history upload failed; will retry next tick" ||
		record.Message == "history upload failed; full stays completed" ||
		record.Message == "timeline switch observed via .history" {
		select {
		case observer.events <- record.Message:
		default:
		}
	}

	return nil
}

func (observer historyObserver) WithAttrs(attrs []slog.Attr) slog.Handler {
	observer.Handler = observer.Handler.WithAttrs(attrs)
	return observer
}

func (observer historyObserver) WithGroup(name string) slog.Handler {
	observer.Handler = observer.Handler.WithGroup(name)
	return observer
}

func (observer historyObserver) Wait(t *testing.T, message string) {
	t.Helper()
	testwait.Until(t, message, func(context.Context) error {
		for {
			select {
			case event := <-observer.events:
				if event == message {
					return nil
				}
			default:
				return fmt.Errorf("history event has not arrived")
			}
		}
	})
}

func (observer historyObserver) RequireNoFailures(t *testing.T) {
	t.Helper()

	notificationCount := 0

	for {
		select {
		case event := <-observer.events:
			require.Equal(
				t,
				"timeline switch observed via .history",
				event,
				"both initial history publishers must succeed",
			)
			notificationCount++
		default:
			require.Positive(t, notificationCount, "the receiver must acknowledge the published history")

			return
		}
	}
}
