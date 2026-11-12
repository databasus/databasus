package physicaltest

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

type patroniScheduledBackup struct {
	ID          uuid.UUID
	Reason      string
	ScheduledAt time.Time
}

type patroniScheduleEvents struct {
	mu           sync.Mutex
	backups      []patroniScheduledBackup
	finishedRuns []uuid.UUID
}

type patroniScheduleObserver struct {
	slog.Handler
	Events *patroniScheduleEvents
	attrs  []slog.Attr
}

func (observer patroniScheduleObserver) Handle(_ context.Context, record slog.Record) error {
	if record.Message != "scheduled physical backup" && record.Message != "physical backup run finished" {
		return nil
	}

	backup := patroniScheduledBackup{ScheduledAt: record.Time}
	var kind string
	collect := func(attribute slog.Attr) bool {
		switch attribute.Key {
		case "backup_id":
			backup.ID, _ = uuid.Parse(attribute.Value.String())
		case "reason":
			backup.Reason = attribute.Value.String()
		case "backup_kind":
			kind = attribute.Value.String()
		}

		return true
	}

	for _, attribute := range observer.attrs {
		collect(attribute)
	}
	record.Attrs(collect)

	if record.Message == "physical backup run finished" {
		observer.Events.mu.Lock()
		defer observer.Events.mu.Unlock()
		observer.Events.finishedRuns = append(observer.Events.finishedRuns, backup.ID)

		return nil
	}

	if kind == "FULL" {
		observer.Events.mu.Lock()
		defer observer.Events.mu.Unlock()
		observer.Events.backups = append(observer.Events.backups, backup)
	}

	return nil
}

func (observer patroniScheduleObserver) WithAttrs(attrs []slog.Attr) slog.Handler {
	observer.attrs = append(slices.Clone(observer.attrs), attrs...)
	return observer
}

func (observer patroniScheduleObserver) WithGroup(name string) slog.Handler {
	observer.Handler = observer.Handler.WithGroup(name)
	return observer
}

func (events *patroniScheduleEvents) GetBackups() []patroniScheduledBackup {
	events.mu.Lock()
	defer events.mu.Unlock()
	return slices.Clone(events.backups)
}

func (events *patroniScheduleEvents) IsRunFinished(backupID uuid.UUID) bool {
	events.mu.Lock()
	defer events.mu.Unlock()
	return slices.Contains(events.finishedRuns, backupID)
}
