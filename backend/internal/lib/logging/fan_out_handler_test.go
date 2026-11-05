package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedSinks struct {
	firstSink  *bytes.Buffer
	secondSink *bytes.Buffer
}

func newTestFanOutHandler(level slog.Level) (*fanOutHandler, capturedSinks) {
	sinks := capturedSinks{firstSink: &bytes.Buffer{}, secondSink: &bytes.Buffer{}}

	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	children := []slog.Handler{
		slog.NewJSONHandler(sinks.firstSink, &slog.HandlerOptions{Level: slog.LevelDebug}),
		slog.NewJSONHandler(sinks.secondSink, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}

	return newFanOutHandler(children, levelVar), sinks
}

func decodeSingleRecord(t *testing.T, sink *bytes.Buffer) map[string]any {
	t.Helper()

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(sink.Bytes(), &decoded))

	return decoded
}

func Test_Handle_WithMultipleSinks_WritesRecordToEverySink(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelInfo)

	slog.New(handler).InfoContext(t.Context(), "backup finished")

	assert.Contains(t, sinks.firstSink.String(), "backup finished")
	assert.Contains(t, sinks.secondSink.String(), "backup finished")
}

func Test_WithAttrs_WithMultipleSinks_ScopedAttrsReachEverySink(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelInfo)

	slog.New(handler).With(slog.String("database_id", "db-42")).InfoContext(t.Context(), "backup finished")

	assert.Equal(t, "db-42", decodeSingleRecord(t, sinks.firstSink)["database_id"])
	assert.Equal(t, "db-42", decodeSingleRecord(t, sinks.secondSink)["database_id"])
}

func Test_WithGroup_WithMultipleSinks_PreservesNestingInEverySink(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelInfo)

	slog.New(handler).WithGroup("backup").InfoContext(t.Context(), "finished", slog.Int("size_mb", 12))

	for _, sink := range []*bytes.Buffer{sinks.firstSink, sinks.secondSink} {
		group, isGroup := decodeSingleRecord(t, sink)["backup"].(map[string]any)

		require.True(t, isGroup)
		assert.EqualValues(t, 12, group["size_mb"])
	}
}

func Test_Handle_WhenLevelIsAboveRecord_SuppressesRecordInEverySink(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelError)

	slog.New(handler).InfoContext(t.Context(), "routine work")

	assert.Empty(t, sinks.firstSink.String())
	assert.Empty(t, sinks.secondSink.String())
}

func Test_Handle_WhenLevelBypassed_AuditRecordSurvivesErrorLevel(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelError)

	auditLogger := slog.New(handler.withLevelBypass()).With(slog.String(logTypeKey, logTypeAudit))
	auditLogger.InfoContext(t.Context(), "Database deleted: payments")

	auditRecord := decodeSingleRecord(t, sinks.firstSink)
	assert.Equal(t, logTypeAudit, auditRecord[logTypeKey])
	assert.Equal(t, "Database deleted: payments", auditRecord["msg"])
	assert.NotEmpty(t, sinks.secondSink.String())
}

// withLevelBypass must return a new handler, not flip a flag on the receiver: logger.go builds
// the audit logger and the application logger from the same fanOut, so a mutating receiver would
// make the whole application ignore LOG_LEVEL.
func Test_WithLevelBypass_WhenDerived_LeavesOriginalHandlerFiltering(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelError)

	bypassed := slog.New(handler.withLevelBypass())
	bypassed.InfoContext(t.Context(), "audit record")
	require.NotEmpty(t, sinks.firstSink.String(), "the derived handler must bypass the level")

	sinks.firstSink.Reset()
	sinks.secondSink.Reset()

	slog.New(handler).InfoContext(t.Context(), "routine work")

	assert.Empty(t, sinks.firstSink.String(), "the original handler must still filter")
	assert.Empty(t, sinks.secondSink.String())
}

func Test_Handle_WithSensitiveAttrs_RedactsBeforeReachingSinks(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelInfo)

	slog.New(handler).
		InfoContext(t.Context(), "connecting", slog.String("password", "hunter2"), slog.String("database_id", "db-42"))

	record := decodeSingleRecord(t, sinks.firstSink)
	assert.Equal(t, redactedValue, record["password"])
	assert.Equal(t, "db-42", record["database_id"])
}

func Test_WithAttrs_WithSensitiveAttrs_RedactsScopedAttrsToo(t *testing.T) {
	handler, sinks := newTestFanOutHandler(slog.LevelInfo)

	slog.New(handler).With(slog.String("api_key", "sk-live-1234")).InfoContext(t.Context(), "calling storage")

	assert.Equal(t, redactedValue, decodeSingleRecord(t, sinks.firstSink)["api_key"])
}
