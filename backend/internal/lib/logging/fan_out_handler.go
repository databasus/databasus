package logging

import (
	"context"
	"errors"
	"log/slog"
)

var _ slog.Handler = (*fanOutHandler)(nil)

// fanOutHandler owns every level decision. Its children are built at LevelDebug and are never
// asked whether they are enabled, so audit records can bypass the configured level by flipping
// isLevelBypassed without any child re-filtering them.
type fanOutHandler struct {
	children        []slog.Handler
	level           *slog.LevelVar
	isLevelBypassed bool
}

func newFanOutHandler(children []slog.Handler, level *slog.LevelVar) *fanOutHandler {
	return &fanOutHandler{
		children:        children,
		level:           level,
		isLevelBypassed: false,
	}
}

func (h *fanOutHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.isLevelBypassed || level >= h.level.Level()
}

func (h *fanOutHandler) Handle(ctx context.Context, record slog.Record) error {
	redactedRecord := slog.NewRecord(record.Time, record.Level, redactMessage(record.Message), record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		redactedRecord.AddAttrs(redactAttr(attr))

		return true
	})

	for _, contextAttr := range getContextAttrs(ctx) {
		redactedRecord.AddAttrs(redactAttr(contextAttr))
	}

	var sinkErrors []error
	for _, child := range h.children {
		if err := child.Handle(ctx, redactedRecord); err != nil {
			sinkErrors = append(sinkErrors, err)
		}
	}

	return errors.Join(sinkErrors...)
}

// WithAttrs redacts here as well as in Handle: scoped attrs are baked into the children at this
// point and would never pass through Handle's redaction again.
func (h *fanOutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redactedAttrs := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		redactedAttrs = append(redactedAttrs, redactAttr(attr))
	}

	children := make([]slog.Handler, 0, len(h.children))
	for _, child := range h.children {
		children = append(children, child.WithAttrs(redactedAttrs))
	}

	return &fanOutHandler{
		children:        children,
		level:           h.level,
		isLevelBypassed: h.isLevelBypassed,
	}
}

func (h *fanOutHandler) WithGroup(name string) slog.Handler {
	children := make([]slog.Handler, 0, len(h.children))
	for _, child := range h.children {
		children = append(children, child.WithGroup(name))
	}

	return &fanOutHandler{
		children:        children,
		level:           h.level,
		isLevelBypassed: h.isLevelBypassed,
	}
}

func (h *fanOutHandler) withLevelBypass() *fanOutHandler {
	return &fanOutHandler{
		children:        h.children,
		level:           h.level,
		isLevelBypassed: true,
	}
}
