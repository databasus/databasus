package audit_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/testdb"
)

var errRolledBack = errors.New("operation failed")

func TestRecord(t *testing.T) {
	t.Parallel()

	t.Run("keeps the event when the surrounding transaction commits", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		auditLog := newAuditLog(db, newTestClock(), slog.New(slog.DiscardHandler))

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			return auditLog.Record(ctx, audit.Event{Author: nil, WorkspaceID: nil, Message: "storage created"})
		})

		require.NoError(t, err)
		assert.Len(t, listStoredEvents(t, db), 1)
	})

	t.Run("leaves no event when the surrounding transaction rolls back", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		auditLog := newAuditLog(db, newTestClock(), slog.New(slog.DiscardHandler))

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			recordErr := auditLog.Record(ctx, audit.Event{Author: nil, WorkspaceID: nil, Message: "storage created"})
			require.NoError(t, recordErr)

			return errRolledBack
		})

		require.ErrorIs(t, err, errRolledBack)
		assert.Empty(t, listStoredEvents(t, db))
	})

	t.Run("stores the author, the workspace and the message when the event has them", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		auditLog := newAuditLog(db, newTestClock(), slog.New(slog.DiscardHandler))
		author := audit.Author{UserID: uuid.New(), Email: "alice@example.com", Name: "Alice"}
		workspaceID := uuid.New()

		err := auditLog.Record(t.Context(), audit.Event{
			Author:      &author,
			WorkspaceID: &workspaceID,
			Message:     "storage created: nightly",
		})

		require.NoError(t, err)
		storedEvents := listStoredEvents(t, db)
		require.Len(t, storedEvents, 1)
		require.NotNil(t, storedEvents[0].AuthorUserID)
		assert.Equal(t, author.UserID, *storedEvents[0].AuthorUserID)
		assert.Equal(t, &author.Email, storedEvents[0].AuthorEmail)
		assert.Equal(t, &author.Name, storedEvents[0].AuthorName)
		assert.Equal(t, &workspaceID, storedEvents[0].WorkspaceID)
		assert.Equal(t, "storage created: nightly", storedEvents[0].Message)
	})

	t.Run("stores none of the author fields when the event has no author", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		auditLog := newAuditLog(db, newTestClock(), slog.New(slog.DiscardHandler))

		err := auditLog.Record(t.Context(), audit.Event{Author: nil, WorkspaceID: nil, Message: "sign-in failed"})

		require.NoError(t, err)
		storedEvents := listStoredEvents(t, db)
		require.Len(t, storedEvents, 1)
		assert.Nil(t, storedEvents[0].AuthorUserID)
		assert.Nil(t, storedEvents[0].AuthorEmail)
		assert.Nil(t, storedEvents[0].AuthorName)
	})

	t.Run("refuses a row with only part of the author", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)

		err := db.WithContext(t.Context()).Exec(
			`INSERT INTO lib_audit_events (id, author_user_id, author_email, message, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			uuid.New(), uuid.New(), "alice@example.com", "storage created", time.Now().UTC(),
		).Error

		require.ErrorContains(t, err, "ck_lib_audit_events_author")
	})

	t.Run("takes the event time from the clock", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		testClock := newTestClock()
		testClock.Advance(36 * time.Hour)
		auditLog := newAuditLog(db, testClock, slog.New(slog.DiscardHandler))

		err := auditLog.Record(t.Context(), audit.Event{Author: nil, WorkspaceID: nil, Message: "storage created"})

		require.NoError(t, err)
		storedEvents := listStoredEvents(t, db)
		require.Len(t, storedEvents, 1)
		assert.True(t, testClock.Now().Equal(storedEvents[0].CreatedAt))
	})

	t.Run("sends the audit logger the constant message with the event text in message", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		capturer := &recordCapturer{mutex: sync.Mutex{}, records: nil}
		auditLog := newAuditLog(db, newTestClock(), logging.NewWithSinks(capturer).GetAuditLogger())
		author := audit.Author{UserID: uuid.New(), Email: "alice@example.com", Name: "Alice"}
		workspaceID := uuid.New()

		err := auditLog.Record(t.Context(), audit.Event{
			Author:      &author,
			WorkspaceID: &workspaceID,
			Message:     "storage created: nightly",
		})

		require.NoError(t, err)
		auditRecord := capturer.requireSingle(t)
		assert.Equal(t, "audit event recorded", auditRecord.message)
		assert.Equal(t, "storage created: nightly", auditRecord.attributes["message"])
		assert.Equal(t, author.UserID.String(), auditRecord.attributes["author_user_id"])
		assert.Equal(t, workspaceID.String(), auditRecord.attributes["workspace_id"])
		assert.Equal(t, "audit", auditRecord.attributes["log_type"])
	})
}

func TestDeleteExpired(t *testing.T) {
	t.Parallel()

	t.Run("deletes the events older than one year and keeps the rest", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		testClock := newTestClock()
		now := testClock.Now()
		auditLog := newAuditLog(db, testClock, slog.New(slog.DiscardHandler))
		recordEventsAt(t, db, map[string]time.Time{
			"older than a year":     now.AddDate(-1, 0, 0).Add(-time.Second),
			"much older than year":  now.AddDate(-2, 0, 0),
			"exactly one year old":  now.AddDate(-1, 0, 0),
			"younger than one year": now.AddDate(0, -11, 0),
			"recorded now":          now,
		})

		deletedCount, err := auditLog.DeleteExpired(t.Context(), now)

		require.NoError(t, err)
		assert.Equal(t, int64(2), deletedCount)
		assert.ElementsMatch(t,
			[]string{"exactly one year old", "younger than one year", "recorded now"},
			listStoredMessages(t, db),
		)
	})

	t.Run("deletes nothing when no event has expired", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		testClock := newTestClock()
		auditLog := newAuditLog(db, testClock, slog.New(slog.DiscardHandler))
		recordEventsAt(t, db, map[string]time.Time{"recorded now": testClock.Now()})

		deletedCount, err := auditLog.DeleteExpired(t.Context(), testClock.Now())

		require.NoError(t, err)
		assert.Zero(t, deletedCount)
		assert.Equal(t, []string{"recorded now"}, listStoredMessages(t, db))
	})

	t.Run("deletes nothing when the table is empty", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		testClock := newTestClock()
		auditLog := newAuditLog(db, testClock, slog.New(slog.DiscardHandler))

		deletedCount, err := auditLog.DeleteExpired(t.Context(), testClock.Now())

		require.NoError(t, err)
		assert.Zero(t, deletedCount)
	})
}

func newAuditLog(db *gorm.DB, testClock clock.Source, auditLogger *slog.Logger) *audit.Log {
	return audit.New(audit.Deps{DB: db, Clock: testClock, AuditLogger: auditLogger})
}

func newTestClock() *clock.Manual {
	return clock.NewManual(time.Date(2026, 3, 1, 12, 0, 0, 123456000, time.UTC))
}

// recordEventsAt records each event with the clock set to its time, so the test needs no SQL to
// backdate an event.
func recordEventsAt(t *testing.T, db *gorm.DB, timesByMessage map[string]time.Time) {
	t.Helper()

	for message, recordedAt := range timesByMessage {
		auditLog := newAuditLog(db, clock.NewManual(recordedAt), slog.New(slog.DiscardHandler))
		require.NoError(t, auditLog.Record(t.Context(), audit.Event{Author: nil, WorkspaceID: nil, Message: message}))
	}
}

type storedEvent struct {
	ID           uuid.UUID
	AuthorUserID *uuid.UUID
	AuthorEmail  *string
	AuthorName   *string
	WorkspaceID  *uuid.UUID
	Message      string
	CreatedAt    time.Time
}

func listStoredEvents(t *testing.T, db *gorm.DB) []storedEvent {
	t.Helper()

	var storedEvents []storedEvent
	require.NoError(t, db.WithContext(t.Context()).Raw(
		`SELECT id, author_user_id, author_email, author_name, workspace_id, message, created_at
		FROM lib_audit_events`,
	).Scan(&storedEvents).Error)

	return storedEvents
}

func listStoredMessages(t *testing.T, db *gorm.DB) []string {
	t.Helper()

	var storedMessages []string
	for _, event := range listStoredEvents(t, db) {
		storedMessages = append(storedMessages, event.Message)
	}

	return storedMessages
}

func openMigratedDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), testdb.Open(t))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

type capturedRecord struct {
	message    string
	attributes map[string]string
}

type recordCapturer struct {
	mutex   sync.Mutex
	records []capturedRecord
}

func (c *recordCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (c *recordCapturer) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &scopedCapturer{capturer: c, attrs: attrs}
}

func (c *recordCapturer) WithGroup(string) slog.Handler { return c }

func (c *recordCapturer) Handle(_ context.Context, record slog.Record) error {
	return c.capture(record, nil)
}

func (c *recordCapturer) capture(record slog.Record, scopedAttrs []slog.Attr) error {
	captured := capturedRecord{message: record.Message, attributes: map[string]string{}}

	for _, attr := range scopedAttrs {
		captured.attributes[attr.Key] = attr.Value.String()
	}

	record.Attrs(func(attr slog.Attr) bool {
		captured.attributes[attr.Key] = attr.Value.String()

		return true
	})

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.records = append(c.records, captured)

	return nil
}

func (c *recordCapturer) requireSingle(t *testing.T) capturedRecord {
	t.Helper()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	require.Len(t, c.records, 1)

	return c.records[0]
}

// scopedCapturer keeps the attributes the audit logger is scoped with, such as log_type.
type scopedCapturer struct {
	capturer *recordCapturer
	attrs    []slog.Attr
}

func (s *scopedCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (s *scopedCapturer) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &scopedCapturer{capturer: s.capturer, attrs: append(s.attrs[:len(s.attrs):len(s.attrs)], attrs...)}
}

func (s *scopedCapturer) WithGroup(string) slog.Handler { return s }

func (s *scopedCapturer) Handle(_ context.Context, record slog.Record) error {
	return s.capturer.capture(record, s.attrs)
}
