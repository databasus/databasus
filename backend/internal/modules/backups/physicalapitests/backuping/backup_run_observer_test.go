package backuping_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"databasus-backend/internal/lib/testwait"
)

type backupRunObserver struct {
	slog.Handler
	started chan context.Context
}

func (observer backupRunObserver) Handle(ctx context.Context, record slog.Record) error {
	if record.Message == "physical full backup started" || record.Message == "physical incremental backup started" {
		observer.started <- ctx
	}

	return nil
}

func (observer backupRunObserver) WaitForStart(t *testing.T) context.Context {
	t.Helper()
	var backupCtx context.Context
	testwait.Until(t, "the physical backup reports its run context", func(context.Context) error {
		select {
		case backupCtx = <-observer.started:
			return nil
		default:
			return fmt.Errorf("physical backup has not started")
		}
	})

	return backupCtx
}

func (observer backupRunObserver) WithAttrs(attrs []slog.Attr) slog.Handler {
	observer.Handler = observer.Handler.WithAttrs(attrs)
	return observer
}

func (observer backupRunObserver) WithGroup(name string) slog.Handler {
	observer.Handler = observer.Handler.WithGroup(name)
	return observer
}
