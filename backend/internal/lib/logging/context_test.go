package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWith(t *testing.T) {
	t.Parallel()

	t.Run("adds the context attributes to the record in every sink", func(t *testing.T) {
		t.Parallel()
		handler, sinks := newTestFanOutHandler(slog.LevelInfo)

		ctx := With(t.Context(), slog.String("request_id", "6f1c2b7e-0a3d-4c8f-9b21-5d7e4a1c3f90"))
		slog.New(handler).InfoContext(ctx, "storage created")

		for _, sink := range []*bytes.Buffer{sinks.firstSink, sinks.secondSink} {
			assert.Equal(t, "6f1c2b7e-0a3d-4c8f-9b21-5d7e4a1c3f90", decodeSingleRecord(t, sink)["request_id"])
		}
	})

	t.Run("keeps the earlier attributes when called again on the returned context", func(t *testing.T) {
		t.Parallel()
		handler, sinks := newTestFanOutHandler(slog.LevelInfo)

		requestContext := With(t.Context(), slog.String("request_id", "request-1"))
		jobContext := With(
			requestContext,
			slog.String("job_name", "audit_events_cleanup"),
			slog.String("job_id", "job-1"),
		)
		slog.New(handler).InfoContext(jobContext, "pass finished")

		loggedRecord := decodeSingleRecord(t, sinks.firstSink)
		assert.Equal(t, "request-1", loggedRecord["request_id"])
		assert.Equal(t, "audit_events_cleanup", loggedRecord["job_name"])
		assert.Equal(t, "job-1", loggedRecord["job_id"])
	})

	t.Run("leaves the parent context unchanged when a child adds attributes", func(t *testing.T) {
		t.Parallel()
		handler, sinks := newTestFanOutHandler(slog.LevelInfo)

		parentContext := With(t.Context(), slog.String("request_id", "request-1"))
		_ = With(parentContext, slog.String("user_id", "user-1"))
		slog.New(handler).InfoContext(parentContext, "storage created")

		loggedRecord := decodeSingleRecord(t, sinks.firstSink)
		assert.Equal(t, "request-1", loggedRecord["request_id"])
		assert.NotContains(t, loggedRecord, "user_id")
	})

	t.Run("omits the attributes when the context carries none", func(t *testing.T) {
		t.Parallel()
		handler, sinks := newTestFanOutHandler(slog.LevelInfo)

		slog.New(handler).InfoContext(t.Context(), "storage created")

		assert.NotContains(t, decodeSingleRecord(t, sinks.firstSink), "request_id")
	})

	t.Run("masks a context attribute when its name marks a secret", func(t *testing.T) {
		t.Parallel()
		handler, sinks := newTestFanOutHandler(slog.LevelInfo)

		ctx := With(
			t.Context(),
			slog.String("api_token", "sk-live-1234"),
			slog.String("user_email", "alice@example.com"),
		)
		slog.New(handler).InfoContext(ctx, "storage created")

		loggedRecord := decodeSingleRecord(t, sinks.firstSink)
		assert.Equal(t, redactedValue, loggedRecord["api_token"])
		assert.Equal(t, "a***@example.com", loggedRecord["user_email"])
	})

	t.Run("adds the context attributes to the records of the audit logger", func(t *testing.T) {
		t.Parallel()
		sink := &bytes.Buffer{}
		assembledLogging := assemble(
			builtSinks{
				handlers: []slog.Handler{slog.NewJSONHandler(sink, &slog.HandlerOptions{Level: slog.LevelDebug})},
			},
			slog.LevelError,
		)

		ctx := With(t.Context(), slog.String("request_id", "request-1"))
		assembledLogging.GetAuditLogger().InfoContext(ctx, "audit event recorded")

		loggedRecord := decodeSingleRecord(t, sink)
		assert.Equal(t, "request-1", loggedRecord["request_id"])
		assert.Equal(t, logTypeAudit, loggedRecord[logTypeKey])
	})
}
