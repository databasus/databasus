package logging

import (
	"log/slog"
	"os"
)

const consoleTimeFormat = "2006/01/02 15:04:05"

func newStdoutHandler() slog.Handler {
	return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey && attr.Value.Kind() == slog.KindTime {
				attr.Value = slog.StringValue(attr.Value.Time().UTC().Format(consoleTimeFormat))
			}

			return attr
		},
	})
}
