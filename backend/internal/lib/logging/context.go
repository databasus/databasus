package logging

import (
	"context"
	"log/slog"
	"slices"
)

type contextAttrsKey struct{}

// With returns a context whose log records carry args, after the attributes ctx already carries.
// It takes the same arguments as slog.Logger.With. Services hold a process-wide logger, so the
// attributes of an operation ride the context the operation already passes down.
func With(ctx context.Context, args ...any) context.Context {
	addedAttrs := slog.Group("", args...).Value.Group()
	if len(addedAttrs) == 0 {
		return ctx
	}

	return context.WithValue(ctx, contextAttrsKey{}, append(slices.Clip(getContextAttrs(ctx)), addedAttrs...))
}

func getContextAttrs(ctx context.Context) []slog.Attr {
	contextAttrs, _ := ctx.Value(contextAttrsKey{}).([]slog.Attr)

	return contextAttrs
}
