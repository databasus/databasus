// Package audit records who did what in the instance. An event keeps its author as they were at the
// time of the action, so it reads the same after the account changes or disappears.
package audit

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/clock"
)

const retentionYears = 1

type Deps struct {
	DB    *gorm.DB
	Clock clock.Source
	// AuditLogger writes the forensic copy of every event, which survives the loss of the database.
	AuditLogger *slog.Logger
}

// Log is the only code that touches lib_audit_events: it writes, reads and deletes the events.
type Log struct {
	repository  *eventRepository
	clock       clock.Source
	auditLogger *slog.Logger
}

func New(deps Deps) *Log {
	return &Log{
		repository:  &eventRepository{db: deps.DB},
		clock:       deps.Clock,
		auditLogger: deps.AuditLogger,
	}
}

// Record writes the event in the transaction ctx carries, so an event of an operation that rolls
// back is gone with it. The forensic copy is sent first and stays even then: it records an attempt.
func (l *Log) Record(ctx context.Context, event Event) error {
	attributes := []slog.Attr{slog.String("message", event.Message)}

	if event.Author != nil {
		attributes = append(attributes, slog.String("author_user_id", event.Author.UserID.String()))
	}

	if event.WorkspaceID != nil {
		attributes = append(attributes, slog.String("workspace_id", event.WorkspaceID.String()))
	}

	l.auditLogger.LogAttrs(ctx, slog.LevelInfo, "audit event recorded", attributes...)

	return l.repository.Insert(ctx, event, l.clock.Now())
}

// DeleteExpired deletes the events older than one year before now and returns how many it deleted.
func (l *Log) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return l.repository.DeleteCreatedBefore(ctx, now.AddDate(-retentionYears, 0, 0))
}

// ListEvents reads one page of events, newest first, with the ID ordering events of the same time.
// Limit must be positive; the caller checks its upper bound.
func (l *Log) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	var after *eventCursor

	if query.After != "" {
		cursor, err := decodeEventCursor(query.After)
		if err != nil {
			return EventPage{}, err
		}

		after = &cursor
	}

	listedEvents, err := l.repository.List(ctx, eventListFilter{
		workspaceID:  query.WorkspaceID,
		authorUserID: query.AuthorUserID,
		limit:        query.Limit + 1,
		after:        after,
	})
	if err != nil {
		return EventPage{}, err
	}

	page := EventPage{Events: listedEvents[:min(len(listedEvents), query.Limit)], NextCursor: ""}

	if len(listedEvents) > query.Limit {
		lastEvent := listedEvents[query.Limit-1]
		page.NextCursor = eventCursor{createdAt: lastEvent.CreatedAt, eventID: lastEvent.ID}.encode()
	}

	return page, nil
}
