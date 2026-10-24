//go:build testhelpers

package task_cancellation

import (
	"io"
	"log/slog"
)

func NewTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
