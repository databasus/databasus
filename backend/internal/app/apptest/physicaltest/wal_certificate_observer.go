package physicaltest

import (
	"context"
	"log/slog"
	"slices"
)

type walCertificateObserver struct {
	slog.Handler
	Events chan<- string
	attrs  []slog.Attr
}

func (observer walCertificateObserver) Handle(_ context.Context, record slog.Record) error {
	if record.Message != "open verified WAL session" && record.Message != "wal streamer exited with error" {
		return nil
	}

	collect := func(attribute slog.Attr) bool {
		if attribute.Key == "error" {
			select {
			case observer.Events <- attribute.Value.String():
			default:
			}
		}

		return true
	}

	for _, attribute := range observer.attrs {
		collect(attribute)
	}
	record.Attrs(collect)

	return nil
}

func (observer walCertificateObserver) WithAttrs(attrs []slog.Attr) slog.Handler {
	observer.attrs = append(slices.Clone(observer.attrs), attrs...)
	return observer
}

func (observer walCertificateObserver) WithGroup(name string) slog.Handler {
	observer.Handler = observer.Handler.WithGroup(name)
	return observer
}
